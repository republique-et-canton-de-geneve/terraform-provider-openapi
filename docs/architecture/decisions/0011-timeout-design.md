# 11. Timeout design: custom block over framework `resource/timeouts`

Date: 2026-05-11


## Status

Accepted


## Context

Per-operation timeouts serve two purposes:

* **Safety ceiling**: prevent a slow or hung API call from blocking a Terraform run indefinitely.
* **Expectation contract**: document how long each operation is expected to take, so Terraform
  does not exit too early on legitimately slow operations (e.g. a VM provisioning that takes 25
  minutes is not a bug; a 20-minute default timeout would incorrectly abort it).

Both sides matter: a timeout set too high masks real hangs; one set too low causes false failures
on slow but correct operations. The `x-timeout` extension lets API authors encode the expected
duration per operation so the provider applies a realistic default rather than a generic one.

Two implementation mechanisms were considered:

* **Framework `resource/timeouts` package**: ships a ready-made schema block, `timeouts.Value`
  state type, and `.Create(ctx, default)` / `.Read()` / … extraction helpers.
* **Custom `SingleNestedBlock` with string attributes**: a block with four optional string
  attributes (create, read, update, delete) validated by a `positiveDuration` validator.

The framework package was ruled out for three reasons:

1. It models only four operations (create/read/update/delete) and has no slot for the `list`
   timeout (collection GET, used by data sources and by resources whose Read calls the collection
   endpoint because the API offers no item-level GET).
2. It is designed for static schemas where state is a known Go struct. Our provider builds schemas
   dynamically at runtime; state is a single `types.Object`. Embedding `timeouts.Value` inside it
   requires encoding/decoding glue that erases the package's benefit.
3. It does not validate that user-supplied values are greater than zero; `positiveDuration` would
   be needed regardless.

Timeout defaults are sourced from the `x-timeout` OAS3 extension on each operation. Invalid
values (unparseable or not greater than zero) cause the provider to exit at startup.


## Decision

Use a custom `SingleNestedBlock` with four optional string attributes (create, read, update,
delete) for the resource `timeouts` block. Add a separate `list` timeout read from the collection
GET and applied to data source reads (and, in the future, to resource Read on APIs that have no
item-level GET endpoint).

See [extensions/implemented/x-timeout.md](../extensions/implemented/x-timeout.md) for the full
specification.


## Consequences

* The `timeouts` block in generated resources is not the standard framework block; it behaves
  identically but is declared manually.
* Adding a new operation type (e.g. `list` for resource Read) requires only adding a field to
  `ResourceTimeouts` and wiring it through; no framework migration needed.
* Positive-duration validation is enforced both at provider startup (spec values via the early-error
  check in `provider.go`) and at plan time (user input via `positiveDuration`).
