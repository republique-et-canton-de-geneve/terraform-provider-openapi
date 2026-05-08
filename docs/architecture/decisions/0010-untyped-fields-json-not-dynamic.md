# 10. Untyped OAS fields as jsontypes.Normalized, not Terraform dynamic

Date: 2026-05-08


## Status

Accepted


## Context

OAS3 allows property schemas with no `type:` declaration. Such a field accepts any valid JSON
value: a string, number, boolean, array, or object. The provider must map these **untyped fields**
to some Terraform attribute type.

The Terraform Plugin Framework provides a `dynamic` pseudo-type (`schema.DynamicAttribute`,
`types.Dynamic`) specifically for values whose type is not known at schema-build time. This was
the first approach tried (in the `-wip` branch). In practice it proved unreliable in several ways:

* **Nested type inference breaks.** When a dynamic field appears inside an object or array, the
  framework must infer a concrete type for the value during planning. Mixed-type arrays and
  object-typed values produced inconsistent diffs and plan errors that were difficult to diagnose.
* **State round-trips are fragile.** The framework serialises a `dynamic` value by embedding a
  type tag alongside the value. If the inferred type changes between plan and apply (e.g. an
  empty list `[]` inferred as `list<string>` vs. `list<dynamic>`), Terraform rejects the state
  with a type mismatch error.
* **Limited ecosystem support.** As of 2026, `dynamic` attributes are a relatively new addition
  to the Plugin Framework and several features (plan modifiers, defaults, validators) are more
  constrained for dynamic attributes than for typed ones.

The alternative is `jsontypes.Normalized` from the
`terraform-plugin-framework-jsontypes` package. This is a `StringAttribute` with a custom type
that validates and normalises its content as JSON. The field is stored as a JSON string in state.

This approach was first applied in
[terraform-provider-aria](https://github.com/davidfischer-ch/terraform-provider-aria), a
provider for VMware Aria (formerly vRealize Automation), and is ported here.


## Decision

Map untyped OAS fields to `jsontypes.Normalized` (mode `json`), which is the default mode.

A second mode, `error`, is also provided for spec authors who want a hard guarantee that no
untyped fields exist: the provider aborts at startup with a clear message identifying the
offending resource and field.

The mode is controlled by the `OPENAPI_UNTYPED_MODE` environment variable and the `untyped_mode`
provider configuration attribute (which validates agreement with the env var at configure time,
following the same pattern as `prefix`).

The Terraform `dynamic` type is not exposed as a configurable option.


## Consequences

* Untyped fields are stored as JSON strings in Terraform state. Users write values with
  `jsonencode(...)` or a JSON literal.
* Diffs are stable: `jsontypes.Normalized` normalises key ordering and whitespace before
  comparison, so semantically equivalent JSON does not produce a spurious plan diff.
* The `error` mode can be used in CI to enforce complete type annotations in the OAS spec.
* Users can no longer write native HCL objects or lists for untyped fields; they must use
  `jsonencode`. This is a deliberate trade-off: ergonomics are slightly reduced, but correctness
  and predictability are greatly improved.
* The `dynamic` type experiment in the `-wip` branch is abandoned.
