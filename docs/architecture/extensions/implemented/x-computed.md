# `x-computed`


## Properties

* **Scope**: field
* **Value**: `true`


## Description

Marks an optional field whose default value is computed by the server and is not known at plan
time. Without this extension the provider cannot distinguish between a truly optional field and
one that gets a silent server-generated default, causing drift after the first apply.


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
