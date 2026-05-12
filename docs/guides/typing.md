# Type mapping

The provider maps OAS3 schema types to Terraform attribute types at startup, when the spec is
parsed. The mapping is applied per field; nested fields (object properties, array item schemas)
follow the same rules recursively.


## Standard types

| OAS3 declared `type` | Terraform attribute type |
|---|---|
| `string` | `StringAttribute` |
| `integer` | `Int64Attribute` |
| `number` | `Float64Attribute` |
| `boolean` | `BoolAttribute` |
| `object` (with `properties:`) | `SingleNestedAttribute` |
| `array` (with `items:`) | `ListAttribute` / `ListNestedAttribute`; `SetAttribute` / `SetNestedAttribute` when `x-unordered: true` + `uniqueItems: true` |

Structural cues take precedence over the declared `type`: a schema with an `items:` key is treated
as an array even if `type: object` is also present, and a schema with a `properties:` key is
treated as an object regardless of other declarations.


## Untyped fields

OAS3 allows schemas with no `type:` declaration: the field is then valid for any JSON value. When
a field has no declared type and no structural cues (`items:`, `properties:`, `allOf`, `oneOf`,
`anyOf`, `enum`), the provider marks it as **untyped**.

```yaml
# No type declaration: untyped field.
emails:
  default: []
payload:
  description: Arbitrary extra data.
```

How untyped fields are exposed is controlled by the `untyped_mode` provider configuration
attribute (or the `OPENAPI_UNTYPED_MODE` environment variable).


For the rationale behind these two modes and the decision to drop Terraform's `dynamic` type,
see [ADR 0010](../architecture/decisions/0010-untyped-fields-json-not-dynamic.md).


## `untyped_mode` values

### `json` (default)

Untyped fields are exposed as `jsontypes.Normalized` (a `StringAttribute` whose value must be
valid, normalised JSON). Users write JSON literals in HCL:

```hcl
resource "openapi_widget" "example" {
  payload = jsonencode({ key = "value", count = 3 })
}
```

Terraform diffs the field as a JSON string. Key ordering and whitespace are normalised before
comparison, so semantically equivalent JSON does not produce a spurious diff.

An untyped field with a `default:` in the spec is automatically marked `Computed`, because the
presence of a default signals that the server will initialise the value when the client omits it.

### `error`

The provider aborts at startup if any field in any discovered resource resolves to untyped:

```
error: resource "widget" has untyped fields and OPENAPI_UNTYPED_MODE=error
```

Use this mode to enforce that every field in the spec carries an explicit type annotation.


## Provider configuration

```hcl
provider "openapi" {
  url          = "https://api.example.com/v1"
  untyped_mode = "json"   # or "error"
}
```

The `untyped_mode` attribute must agree with the `OPENAPI_UNTYPED_MODE` environment variable,
because resource schemas are built from the spec before the provider block is read. Setting the
attribute lets Terraform detect a mismatch early with a clear error instead of silently using
the wrong schema.


## Overriding type inference

The `x-terraform-type` extension may be evaluated as an escape hatch for spec authors who need to
override the inferred type explicitly. It is not yet implemented.
