# 3. Dynamic provider over code generation

Date: 2026-04-24


## Status

Accepted


## Context

Two approaches exist for building a Terraform provider from an OpenAPI spec:

1. **Code generation**: (HashiCorp `terraform-plugin-codegen-openapi`): the spec is processed
   offline, Go source code is emitted, and the resulting binary is compiled and deployed. Every
   spec change requires a regen + recompile + release cycle.

2. **Dynamic provider**: a single binary reads the spec at startup and registers resource types
   at runtime. A spec change takes effect on the next `terraform init` with no provider release.

The target use case is internal APIs that evolve frequently, where the same team owns both the
API and the Terraform configuration. HashiCorp's codegen tooling is still in tech preview as of
2024 and is not intended for production use.


## Decision

Build a dynamic provider. The spec is loaded at startup via `OPENAPI_SPEC` and all resource
types are derived from it at runtime.


## Consequences

* Spec changes are immediately reflected without a provider release cycle.
* The provider binary is generic: one binary serves any OAS3-compliant API.
* Type safety is weaker than generated code; schema errors surface at runtime rather than compile
  time.
* HashiCorp's code-generation tooling evolution does not affect this provider.
