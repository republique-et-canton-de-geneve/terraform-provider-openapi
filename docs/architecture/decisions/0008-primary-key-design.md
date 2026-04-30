# 8. Primary key design: single field vs composite key

Date: 2026-04-29


## Status

TBD


## Context

Every Terraform resource needs a stable identifier used for Read, Update, Delete, and
`terraform import`. The current implementation falls back to a field named `id`, which covers
all current use cases where APIs assign a surrogate `id` field.

Some REST APIs have no single surrogate key and require multiple fields to uniquely identify a
resource. The canonical example is the GitLab project variable: its identity is the combination
of `project`, `key`, and `environment_scope`, expressed as `project:key:environment_scope` in
the Terraform import ID. There is no single `id` field.

Django REST Framework APIs almost always use a single `pk` field, so composite keys are not
needed for current consumers. However, the design should anticipate them.


## Decision

An extension `x-primary-key` is drafted at schema level (consistent with `x-tf-status`),
supporting both single field names and composite format strings. See
[architecture/primary-key.md](../primary-key.md) for the full design, resolution priority, and
examples.

**Kept bare** because a primary key is an API-intrinsic concept meaningful to any spec consumer,
not only to this provider.

No decision on implementation timing.


## Consequences

* `x-primary-key` is documented as TBD in [extensions/planned/x-primary-key.md](../extensions/planned/x-primary-key.md) and
  [architecture/primary-key.md](../primary-key.md).
* Current code falls back to a field named `id`; `x-primary-key` will complete that heuristic.
* APIs with no surrogate `id` field (e.g. GitLab-style project variables) cannot be managed
  until this is implemented.
