# 4. Terraform Plugin Framework over Plugin SDK v2

Date: 2026-04-24


## Status

Accepted


## Context

HashiCorp maintains two provider development libraries:

* **`terraform-plugin-sdk/v2`**: the original library, stable, widely used, but considered legacy.
  HashiCorp has stated it will not receive new features.
* **`terraform-plugin-framework`**: the current recommended library, type-safe, supports modern
  Terraform features (deferred actions, write-only attributes, plan modifiers), and is where all
  new HashiCorp investment goes.

dikhan/terraform-provider-openapi, the closest existing alternative, uses `terraform-plugin-sdk/v2`.


## Decision

Use `terraform-plugin-framework` (v1). This is the library recommended by HashiCorp for all new
providers.


## Consequences

* The provider is aligned with HashiCorp's current direction and will benefit from new framework
  features without migration cost.
* The `resource.Resource` interface (typed methods: `Metadata`, `Schema`, `Create`, `Read`,
  `Update`, `Delete`) is more explicit than the SDK's map-based `schema.Resource` struct.
* dikhan's provider cannot be used as a direct reference implementation; its patterns do not
  translate to the framework API.
