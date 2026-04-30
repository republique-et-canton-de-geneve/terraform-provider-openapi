# `x-sensitive`


## Properties

* **Scope**: field
* **Value**: `true` | `false`


## Description

Marks a field whose value is redacted in `terraform plan` output and state files.

The provider applies a three-way rule:

1. **`x-sensitive: true`** — sensitive regardless of the field name.
2. **`x-sensitive: false`** — not sensitive; suppresses the name heuristic for fields whose
   names would otherwise match (e.g. a field named `token_count` that stores a plain integer).
3. **Extension absent** — falls back to the name heuristic: the lowercase field name is tested
   for any of `password`, `passwd`, `secret`, `private_key`, `privatekey`,
   `api_key`, `apikey`, `token`, `credential`, `passphrase`.


## Example

```yaml
# Explicit opt-in — the name alone would not trigger the heuristic.
vault_key:
  type: string
  x-sensitive: true

# Explicit opt-out — suppress the heuristic for a non-secret "token" field.
token_count:
  type: integer
  x-sensitive: false
```


## Prior art

**Kept bare** because sensitivity is an intrinsic property of the data, meaningful to any tool
that handles the spec (documentation generators, SDK generators, audit tools), not only Terraform.

dikhan `x-terraform-sensitive` (identical semantic) · No other major tool uses a dedicated
extension; most rely on field-name heuristics (as this provider also does) or `writeOnly: true`
(OAS3 native, but only prevents reads and does not redact in-memory).
