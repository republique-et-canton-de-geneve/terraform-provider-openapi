# `x-unordered`


## Properties

* **Scope**: field (array type)
* **Value**: `true`


## Description

Marks an array field where the API may return items in a different order than submitted.

Without this extension, Terraform reports drift every plan when the API reorders items even though
the logical content is unchanged.

The Terraform type and any extra processing depend on the combination of `x-unordered` and the
standard OAS `uniqueItems` keyword:

| `uniqueItems` | `x-unordered` | Terraform type | Processing | Validation |
|---|---|---|---|---|
| false / absent | false / absent | `List` | — | — |
| false / absent | true | `List` | elements sorted on read + plan | — |
| true | false / absent | `List` | — | uniqueness validator |
| true | true | `Set` | — | uniqueness (indirectly via Set) |

### `x-unordered: true` without `uniqueItems`

The provider keeps the field as a Terraform **list** but sorts elements before storing them in
state. Sorting is deterministic regardless of the API's response order, so Terraform sees no drift.
A plan modifier applies the same sort to the config value so that `["ops", "dev"]` in HCL produces
the same plan as `["dev", "ops"]`.

Duplicate elements are allowed: `x-unordered` is about ordering only.

### `uniqueItems: true` without `x-unordered`

The field stays an ordered list. A validator rejects plans where the same element appears more than
once, matching the API contract expressed by `uniqueItems: true`.

### `x-unordered: true` + `uniqueItems: true`

The provider uses a Terraform **set**. Sets are inherently unordered and enforce element uniqueness,
which maps exactly to the API's combined contract. No extra sorting or validation is needed.

HCL syntax for lists and sets is identical (`["a", "b", "c"]`), so users do not need to change their
configuration when moving between cases.


## Examples

### Unordered list without uniqueness

```yaml
groups:
  type: array
  items:
    type: string
  x-unordered: true
```

Terraform plan after the API reorders items from `["dev", "ops"]` to `["ops", "dev"]`:

```
No changes. Your infrastructure matches the configuration.
```

### Ordered unique list

```yaml
roles:
  type: array
  items:
    type: string
  uniqueItems: true
```

Terraform plan when the config contains a duplicate:

```
Error: Duplicate list element
  Element "admin" appears more than once (uniqueItems: true).
```

### Unordered unique list → Set

```yaml
tags:
  type: array
  items:
    type: string
  x-unordered: true
  uniqueItems: true
```

Terraform exposes this as a `Set`. The framework enforces uniqueness and order is ignored.


## Prior art

**Kept bare** because array ordering is a property of the API contract, not a Terraform concept.
Any diff tool consuming the spec could apply the same semantics.

The standard OAS `uniqueItems` keyword covers uniqueness (supported and validated by most tooling,
including validators and codegens like openapi-generator). OAS 3.1 aligns fully with JSON Schema
2020-12. No standard OAS keyword exists for ordering, which is the gap `x-unordered` fills.

dikhan `x-terraform-ignore-order` (identical semantic, verbose name) · No equivalent found
in Azure, Kubernetes, or HashiCorp codegen.
