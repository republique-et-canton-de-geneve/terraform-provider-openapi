# 5. OAS3 only, no Swagger 2 support

Date: 2026-04-24


## Status

Accepted


## Context

The OpenAPI ecosystem has two major versions in production use:

* **Swagger / OpenAPI 2.x**: the older format, still dominant in legacy systems. dikhan's
  provider targets this exclusively and rejects OAS3 specs at runtime. No fork in its ecosystem
  has ever added OAS3 support.
* **OAS3 (3.0.x, 3.1.x)**: the current standard, with native Bearer auth schemes, richer schema
  composition, and `components/` reuse patterns.

Internal APIs this provider was built to consume expose OAS3 specs.


## Decision

Support OAS3 only. Swagger 2 specs are not handled and will produce a parse error at startup.


## Consequences

* The parser targets OAS3 structures exclusively; no version detection or branching is needed.
* Several extensions from dikhan's provider become unnecessary: `x-terraform-authentication-scheme-bearer`
  exists only because OAS2 has no native Bearer scheme; OAS3's `http/bearer` security scheme
  covers it natively.
* Consumers with OAS2 specs must migrate before using this provider. No compatibility shim will
  be added.
