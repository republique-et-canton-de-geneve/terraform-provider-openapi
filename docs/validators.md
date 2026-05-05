# Validators

The provider automatically derives Terraform validators from OAS3 schema constraints.

Constraints present in the spec are applied to the corresponding resource attributes.

Validations are available at plan time and help in guarding from misconfigurations.

## String constraints

| OAS3 keyword | Terraform validator | Example |
|---|---|---|
| `maxLength` | `LengthAtMost(n)` | `maxLength: 255` |
| `minLength` | `LengthAtLeast(n)` | `minLength: 2` |
| `pattern` | `RegexMatches(re)` | `pattern: ^[0-9]{4,5}$` |
| `enum` | `OneOf(values...)` | see [Enum patterns](#enum-patterns) |

## Integer constraints

| OAS3 keyword | Terraform validator | Example |
|---|---|---|
| `minimum` | `AtLeast(n)` | `minimum: 0` |
| `maximum` | `AtMost(n)` | `maximum: 4096` |

## Enum patterns

Enumerations are commonly expressed using `$ref` indirection rather than inline values.

The provider handles three patterns, all producing a `OneOf` validator:

### Direct enum

```yaml
status:
  type: string
  enum: [active, inactive, pending]
```

### allOf + $ref (Django REST Framework pattern, our main use case)

DRF serializers emit a wrapping `allOf` with a single `$ref` to a named enum schema:

```yaml
components:
  schemas:
    StatusEnum:
      type: string
      enum: [active, inactive, pending]

# field definition
status:
  allOf:
  - $ref: '#/components/schemas/StatusEnum'
```

### oneOf + $ref (nullable pattern)

A field that accepts either an enum value or a blank string uses `oneOf` with two schemas.

One for the values, one for the empty string:

```yaml
components:
  schemas:
    StatusEnum:
      type: string
      enum: [active, inactive, pending]
    BlankEnum:
      enum: ['']

# field definition
status:
  oneOf:
  - $ref: '#/components/schemas/StatusEnum'
  - $ref: '#/components/schemas/BlankEnum'
```

The provider merges all values from both schemas into a single `OneOf` validator, so `""` is
accepted alongside the named values.

## Read-only fields

Validators are extracted from the schema regardless of whether a field is read-only.

Since Terraform does not validate computed values by design, it will not validate server-side
generated data.

## Test fixture

The file `internal/spec/testdata/validation.yaml` is an OAS3 spec containing the examples listed
above. It's used as a fixture for testing purposes.
