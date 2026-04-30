# `x-terraform-provider-multiregion-fqdn` + `x-terraform-provider-regions`


## Description

Parameterises the provider's base URL with a region variable so one spec serves multiple regional
endpoints.


## Decision

**Rejected.** See [ADR 0007](../../decisions/0007-no-multiregion-no-per-resource-host.md).

Terraform already solves this natively: declare multiple provider blocks with `alias` and
different `url` values, then assign resources to the appropriate provider via the `provider`
meta-argument. The extension approach hard-codes region topology into the spec, coupling API
documentation to a deployment concern.


## Prior art

dikhan only; no equivalent found in any other provider or tool.
