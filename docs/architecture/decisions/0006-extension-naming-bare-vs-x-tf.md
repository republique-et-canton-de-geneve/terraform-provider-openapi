# 6. Extension naming: bare names vs `x-tf-` prefix

Date: 2026-04-25


## Status

Accepted


## Context

OpenAPI vendor extensions require an `x-` prefix by spec. Beyond that, naming is a choice.
dikhan's provider uses `x-terraform-` for all extensions. The OAS extension registry
(Mermade, 194k real-world specs) shows that major vendors namespace consistently:
`x-ms-`, `x-amazon-`, `x-google-`, `x-kubernetes-`.

Two categories of extension were identified:

1. **API-intrinsic semantics**: properties of the data or the API contract meaningful to any
   consumer (documentation generators, SDK generators, audit tools), not only Terraform.
   Examples: a field that is write-once, a field whose value is a secret.
2. **Terraform-specific mechanics**: decisions about how this provider maps the API to
   Terraform's resource model. Another tool reading the same spec would have no use for these.
   Examples: which field to use as the Terraform resource ID, whether to hide a resource.

Bare name collision risk was also assessed: `x-immutable` has 640+ existing usages in the wild
with compatible semantics (safe to keep bare). `x-identifier` has 22+ usages with incompatible
semantics (must be renamed).


## Decision

* **Bare `x-` names** for API-intrinsic extensions with community convergence or no collision
  risk: `x-immutable`, `x-sensitive`, `x-computed`, `x-ignore-order`, `x-primary-key`,
  `x-timeout`.
* **`x-tf-` prefix** for Terraform-specific mechanics: `x-tf-exclude`, `x-tf-status`,
  `x-tf-name`.


## Consequences

* Bare extensions (`x-immutable`, `x-sensitive`) may be recognised by other tools (ReDoc renders
  `x-immutable`), which is intentional: the spec remains useful beyond Terraform.
* `x-tf-` extensions are clearly scoped and have zero hits in the 194k-spec registry analysis.
* `x-identifier` was avoided due to semantic collision; `x-primary-key` (schema-level) is the
  primary key extension instead.
