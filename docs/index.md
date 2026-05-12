# Terraform Provider OpenAPI (Dynamic)

A dynamic Terraform provider that generates resource types at runtime from an OpenAPI 3
specification. Point it at any OAS3 spec and it exposes every discoverable resource as a Terraform
resource and data source, with support for custom extensions (`x-immutable`, `x-computed`, …).


## Requirements

* Terraform >= 1.8
* `OPENAPI_SPEC` pointing at your OAS3 spec before running `terraform init`


## Environment variables

| Variable | Required | Description |
|---|---|---|
| `OPENAPI_SPEC` | yes | Path or HTTPS URL to the OpenAPI 3 spec |
| `OPENAPI_URL` | yes | Base URL of the API (e.g. `https://api.example.com/v1`) |
| `OPENAPI_TOKEN` | no | Bearer token sent as `Authorization: Bearer …` |
| `OPENAPI_INSECURE` | no | Set to `true` to skip TLS certificate verification |
| `OPENAPI_PREFIX` | no | Resource type name prefix (default `openapi` -> `openapi_<name>`) |
| `OPENAPI_UNTYPED_MODE` | no | How fields with no OAS type are handled (default `json`; see [typing.md](guides/typing.md)) |
| `OPENAPI_OK_LOG_LEVEL` | no | Log level for successful API calls (default `TRACE`) |
| `OPENAPI_KO_LOG_LEVEL` | no | Log level for failed API calls (default `ERROR`) |


## Provider configuration

```hcl
provider "openapi" {
  url          = "https://api.example.com/v1"
  token        = var.api_token  # or OPENAPI_TOKEN
  insecure     = false
  prefix       = "openapi"      # must match OPENAPI_PREFIX
  untyped_mode = "json"         # must match OPENAPI_UNTYPED_MODE
}
```

All attributes are optional in the provider block if the corresponding environment variable is set.

`prefix` and `untyped_mode` are special: resource schemas are built at init time from environment
variables before the provider block is evaluated. Declaring them here lets Terraform detect a
mismatch at configure time instead of silently using the wrong schema.


## Resource discovery

At startup the provider reads the spec and walks all paths. A **collection path** (`/vlans/`)
paired with an **item path** (`/vlans/{id}/`) becomes `resource "openapi_vlan"` and
`data "openapi_vlans"`. The GET `200` response schema drives the Terraform schema; the POST
request body determines which fields are writable.

* Multi-segment paths (`/linux-vm/instances/{id}/`) become `openapi_linux_vm_instance`.
* A common path prefix shared by all paths (e.g. `/api/v1/`) is stripped before naming.
* Resources without a GET `/{id}/` 200 response are silently skipped.
* A `put` or `patch` on the item path enables Update; without either, any field change forces
  replacement. The request body sent to PATCH/PUT contains only the writable fields (those present
  in the POST body and not marked `x-immutable`).
* A `get` on the collection path enables the data source; without it, no data source is generated.

### Singular vs plural naming

The last path segment is inflected automatically:

* **Resources** use the **singular** form: `/vlans/{id}/` -> `resource "openapi_vlan"`
* **Data sources** use the **plural** form: `/vlans/` -> `data "openapi_vlans"`

Hyphens in path segments are replaced with underscores.

See [discoverability.md](guides/discoverability.md) for the built-in manifest data source and
debug tips.


## Field mapping

| OAS3 property | Terraform behaviour |
|---|---|
| camelCase name (e.g. `photoUrls`) | Converted to `snake_case` (`photo_urls`) |
| `readOnly: true` | `Computed: true` — server-managed, never sent in requests |
| present in POST body | `Optional` / `Required` depending on OAS3 `required` |
| absent from POST body | `Computed: true` |
| `default:` | `Optional + Computed` with a static default; see [defaults.md](guides/defaults.md) |
| no declared `type:` | `jsontypes.Normalized` string or startup error; see [typing.md](guides/typing.md) |
| `x-computed: true` | `Computed: true`; plan shows `(known after apply)` on every write |
| `x-unordered: true` | API may return items in any order; combined with `uniqueItems` drives Set vs sorted-List (see [x-unordered.md](architecture/extensions/implemented/x-unordered.md)) |
| `x-immutable: true` | Stable after creation: prior value preserved in plan, changing forces replace |
| `x-sensitive: true` | Value redacted in plan and state |
| `x-sensitive: false` | Opt out of auto-sensitive detection (e.g. `num_tokens` contains `token` but is not a secret) |
| name contains `password`, `secret`, `token`, `api_key`, … | Auto-marked sensitive unless `x-sensitive: false` |


## Typing

OAS3 types are mapped to Terraform attribute types at startup. Fields with no declared `type:`
are treated as untyped and handled according to the `untyped_mode` setting.

See [typing.md](guides/typing.md) for the full type mapping and untyped field modes.


## Validation

OAS3 schema constraints (`maxLength`, `minLength`, `pattern`, `minimum`, `maximum`, `enum`) are
automatically translated into Terraform validators applied at plan time. Enum values expressed via
`$ref`, `allOf`, or `oneOf` are all recognised.

See [validators.md](guides/validators.md) for the full list and enum pattern details.


## Timeouts

Per-operation timeouts are set via `x-timeout` on each OAS3 operation. For resources, users can
override any value in the `timeouts` block; user-supplied values must be valid durations greater
than zero. The fallback is 20 minutes (Terraform's standard default).

The collection-path GET timeout (`/resources/` GET) applies to data source reads; the item-path
GET timeout (`/resources/{id}/` GET) applies to resource Read and to each polling GET during
delete.

See [architecture/extensions/implemented/x-timeout.md](architecture/extensions/implemented/x-timeout.md)
for the full specification.


## OAS3 extensions

| Extension | Scope | Description |
|---|---|---|
| `x-computed` | field | Server sets or updates this field independently of user input |
| `x-unordered` | field | API may return items in any order; combined with `uniqueItems` drives Set vs sorted-List |
| `x-immutable` | field | Stable after creation: prior value kept in plan, change forces replace |
| `x-sensitive` | field | Field value is redacted in plan and state |
| `x-timeout` | operation | Default timeout for the corresponding Terraform action (list/create/read/update/delete) |

Full extension documentation, naming rationale, and the planned roadmap extensions
(`x-primary-key`, `x-tf-status`, …) are in
[architecture/extensions/index.md](architecture/extensions/index.md).

`uniqueItems: true` on an array (standard OAS keyword, no extension needed) adds a
uniqueness validator to the list; combined with `x-unordered: true` it selects a Set instead.


## Example: create-only resource (no update)

When an item path declares no `put` or `patch`, every field is implicitly immutable: any change
triggers a destroy-then-recreate cycle, equivalent to putting `x-immutable: "true"` on every
field individually.

```yaml
paths:
  /vlans/:
    post:
      requestBody:
        content:
          application/json:
            schema:
              properties:
                name:    { type: string }
                vlan_id: { type: integer }
  /vlans/{id}/:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                properties:
                  id:      { type: integer }
                  name:    { type: string }
                  vlan_id: { type: integer }
    delete: {}
```

The provider exposes only a resource; no data source is generated because the collection path
has no `get` operation:

```hcl
# no patch/put: any change forces replacement
resource "openapi_vlan" "core" {
  name    = "core-network"
  vlan_id = 100
}
```


## Example: full CRUD with extensions and data source

A more complete spec using `x-immutable`, `x-sensitive`, `x-computed`, and `x-timeout`:

```yaml
paths:
  /vms/:
    get:
      x-timeout: "30s"  # list: bounds the data source read
    post:
      x-timeout: "30m"  # create: VM provisioning can be slow
      requestBody:
        content:
          application/json:
            schema:
              properties:
                name:    { type: string }
                image:   { type: string, x-immutable: "true" }
                api_key: { type: string, x-sensitive: "true" }

                # "token" in the name triggers auto-sensitive; x-sensitive: false opts out
                num_tokens: { type: integer, x-sensitive: "false" }
  /vms/{id}/:
    get:
      x-timeout: "10s"  # read: also used for each delete polling GET
      responses:
        "200":
          content:
            application/json:
              schema:
                properties:
                  id:         { type: integer }
                  name:       { type: string }
                  image:      { type: string, x-immutable: "true" }
                  api_key:    { type: string, x-sensitive: "true" }
                  num_tokens: { type: integer, x-sensitive: "false" }
                  created_at: { type: string, x-computed: "true" }
    patch:
      x-timeout: "15m"  # update: name and api_key are writable; image forces replace
    delete:
      x-timeout: "10m"  # delete: includes polling until gone
```

The provider exposes:

```hcl
resource "openapi_vm" "web" {
  name       = "web-01"
  image      = "ubuntu-24.04"  # immutable: changing this forces replacement
  api_key    = var.vm_api_key  # sensitive: redacted in plan and state
  num_tokens = 100             # not sensitive despite "token" in the name

  # override x-timeout spec defaults when needed
  timeouts {
    create = "45m"
  }
}

# created_at is set by the server and read-only in Terraform (x-computed)
output "created_at" {
  value = openapi_vm.web.created_at
}

# data source: lists all VMs using the collection GET (x-timeout: 30s)
data "openapi_vms" "all" {}

output "vm_names" {
  value = [for v in data.openapi_vms.all.items : v.name]
}
```
