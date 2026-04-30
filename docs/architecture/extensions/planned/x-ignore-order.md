# `x-ignore-order`


## Properties

* **Scope**: field
* **Value**: `true`


## Description

Marks a list field where the API may return items in a different order than submitted. Without
this extension, Terraform reports drift every plan when the API reorders items even though the
logical content is unchanged.


## Example

```yaml
groups:
  type: array
  items:
    type: string
  x-ignore-order: true
```


## Prior art

**Kept bare** because array ordering is a property of the API contract, not a Terraform concept.
Any diff tool consuming the spec could apply the same semantics.

dikhan `x-terraform-ignore-order` (identical semantic, verbose name) · No equivalent found in
Azure, Kubernetes, or HashiCorp codegen.
