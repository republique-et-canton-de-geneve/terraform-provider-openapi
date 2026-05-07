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
| `array` (with `items:`) | `ListAttribute` / `ListNestedAttribute` |

Structural cues take precedence over the declared `type`: a schema with an `items:` key is treated
as an array even if `type: object` is also present, and a schema with a `properties:` key is
treated as an object regardless of other declarations.


## Untyped fields

OAS3 allows schemas with no `type:` declaration: the field is then valid for any JSON value. When
a field has no declared type and no structural cues (`items:`, `properties:`, `allOf`, `oneOf`,
`anyOf`, `enum`), the provider maps it to the Terraform `dynamic` pseudo-type.

```yaml
# No type declaration: becomes a dynamic attribute.
emails:
  default: []
payload:
  description: Arbitrary extra data.
```

A `dynamic` field with a `default:` is automatically marked `Computed`, because the presence of a
default signals that the server will initialise the value when the client omits it.


## `dynamic` vs `object`

`object` requires the attribute names and their types to be known at schema-build time. An untyped
field with no `properties:` has no schema to build from, so `object` is not an option. `dynamic`
accepts any value at runtime (string, number, list, or map), which is the correct semantics for
a truly untyped field.

Fields that always carry a JSON object *and* declare their properties in the spec are detected as
`object` automatically through the `properties:` structural cue; no annotation is needed.


## Overriding type inference

The `x-terraform-type` extension may be evaluated as an escape hatch for spec authors who need to
override the inferred type explicitly. It is not yet implemented.
