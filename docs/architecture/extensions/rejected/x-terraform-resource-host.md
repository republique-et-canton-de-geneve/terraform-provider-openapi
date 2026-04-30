# `x-terraform-resource-host`


## Description

Overrides the base URL for a single resource's CRUD operations. Exists for APIs split across
multiple hostnames.


## Decision

**Rejected.** See [ADR 0007](../../decisions/0007-no-multiregion-no-per-resource-host.md).

The correct Terraform pattern is two provider blocks with different `url` values and
resource-level `provider` meta-arguments; that approach is explicit, standard, and requires no
provider-specific extension.


## Prior art

dikhan only.
