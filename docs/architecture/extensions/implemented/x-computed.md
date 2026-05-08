# `x-computed`


## Properties

* **Scope**: field
* **Value**: `true`


## Description

Marks a field whose value is set or updated by the server independently of the user. The provider
exposes it as `Computed: true` with no plan modifiers, so the plan always shows it as
`(known after apply)` after any change.

Use this on fields that can change server-side on every write (e.g. `status`, `last_updated_at`).
For fields that are set once at creation and never change afterward, use `x-immutable` instead.


## Example

```yaml
priority:
  type: integer
  x-computed: true
```


## Prior art

**Kept bare** because the concept of a server-computed default is an API-intrinsic property
independent of any consumer tooling.

dikhan `x-terraform-computed` (identical semantic) · HashiCorp codegen detects this pattern
implicitly via `readOnly` fallback heuristics but has no explicit extension.
