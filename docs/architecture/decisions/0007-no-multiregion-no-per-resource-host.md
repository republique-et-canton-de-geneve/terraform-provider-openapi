# 8. No multi-region extension, no per-resource host override

Date: 2026-04-29


## Status

Accepted


## Context

dikhan's provider includes two extensions for routing resources to different base URLs:

* **`x-terraform-resource-host`**: overrides the base URL for a single resource.
* **`x-terraform-provider-multiregion-fqdn` + `x-terraform-provider-regions`**: parameterises the
  provider's base URL with a region variable so one spec serves multiple regional endpoints.

Both exist because dikhan's provider uses a single global base URL with no other routing
mechanism. Terraform itself solves this natively via provider aliases: multiple provider blocks
with different `url` values, with resources assigned via the `provider` meta-argument.


## Decision

Do not implement either extension. Multi-endpoint scenarios are handled by declaring multiple
provider blocks with `alias` and different `url` values.


## Consequences

* Consumers with resources spread across multiple base URLs must use multiple provider
  declarations. This is more verbose but explicit and consistent with all other Terraform
  providers.
* Region topology is expressed in Terraform configuration, not in the OpenAPI spec, keeping the
  spec free of deployment concerns.
* The spec remains portable to other tools that have no concept of provider aliases.
