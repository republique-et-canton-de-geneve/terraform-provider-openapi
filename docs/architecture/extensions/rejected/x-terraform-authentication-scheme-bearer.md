# `x-terraform-authentication-scheme-bearer`


## Description

Instructs the provider to format the token as `Authorization: Bearer <token>`.


## Decision

**Rejected.** See [ADR 0005](../../decisions/0005-oas3-only.md).

In OAS2 there is no native way to express Bearer auth, so it had to be bolted on via an
extension. OAS3 has a first-class `http` security scheme with `scheme: bearer` that conveys
exactly this. No extension needed.


## Prior art

dikhan only; exists solely to paper over OAS2's lack of a native Bearer scheme.
