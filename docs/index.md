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
  token        = var.api_token   # or OPENAPI_TOKEN
  insecure     = false
  prefix       = "openapi"       # must match OPENAPI_PREFIX
  untyped_mode = "json"          # must match OPENAPI_UNTYPED_MODE
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
| `x-immutable: true` | Stable after creation: prior value preserved in plan, changing forces replace |
| `x-sensitive: true` | Value redacted in plan and state |
| name contains `password`, `secret`, `token`, `api_key`, … | Auto-marked sensitive |


## Typing

OAS3 types are mapped to Terraform attribute types at startup. Fields with no declared `type:`
are treated as untyped and handled according to the `untyped_mode` setting.

See [typing.md](guides/typing.md) for the full type mapping and untyped field modes.


## Validation

OAS3 schema constraints (`maxLength`, `minLength`, `pattern`, `minimum`, `maximum`, `enum`) are
automatically translated into Terraform validators applied at plan time. Enum values expressed via
`$ref`, `allOf`, or `oneOf` are all recognised.

See [validators.md](guides/validators.md) for the full list and enum pattern details.


## OAS3 extensions

| Extension | Scope | Description |
|---|---|---|
| `x-computed` | field | Server sets or updates this field independently of user input |
| `x-immutable` | field | Stable after creation: prior value kept in plan, change forces replace |
| `x-sensitive` | field | Field value is redacted in plan and state |

Full extension documentation, naming rationale, and the planned roadmap extensions
(`x-ignore-order`, `x-primary-key`, `x-tf-status`, …) are in
[architecture/extensions/index.md](architecture/extensions/index.md).


## Example

Given a spec with:

```yaml
paths:
  /vlans/:
    post:
      requestBody:
        content:
          application/json:
            schema:
              properties:
                name: { type: string }
                vlan_id: { type: integer, x-immutable: "true" }
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
                  vlan_id: { type: integer, x-immutable: "true" }
    patch: {}
    delete: {}
```

The provider exposes:

```hcl
resource "openapi_vlan" "core" {
  name    = "core-network"
  vlan_id = 100   # immutable: changing this forces replacement
}
```
