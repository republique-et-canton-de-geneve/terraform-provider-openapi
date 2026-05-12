# Default values

The provider reads the `default:` keyword from writable OAS3 schema properties and applies it as a
Terraform attribute default.

When a default is present the attribute is automatically marked `Optional + Computed`, so Terraform
knows the server will fill the value in if the user omits it from their configuration.

| OAS3 type | Go type stored | Terraform default |
|---|---|---|
| `string` | `string` | `stringdefault.StaticString` |
| `integer` | `int64` | `int64default.StaticInt64` |
| `number` | `float64` | `float64default.StaticFloat64` |
| `boolean` | `bool` | `booldefault.StaticBool` |
| `array` (empty `[]`) | `[]any{}` | `listdefault.StaticValue` (empty list); `setdefault.StaticValue` when `x-unordered: true` + `uniqueItems: true` |

```yaml
disk_d_size:
  type: integer
  default: 30
install_antivirus:
  type: boolean
  default: false
emails:
  type: array
  items:
    type: string
  default: []
```

Default value of read-only fields are ignored, they are already `Computed`.

Non-empty array defaults and object defaults are not supported and are silently ignored.
