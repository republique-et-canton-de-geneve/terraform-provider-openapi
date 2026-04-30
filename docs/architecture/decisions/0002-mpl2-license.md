# 2. MPL-2.0 license

Date: 2026-04-24


## Status

Accepted


## Context

The provider is published as open source on the Terraform Registry under the State of Geneva.

Three licenses were considered:

* **Apache-2.0** permissive, no copyleft, used by most HashiCorp tooling. Anyone can incorporate
  the code into proprietary products without publishing changes.
* **MPL-2.0** weak copyleft at the file level. Modified source files must be published under
  MPL-2.0, but the license does not infect the consuming project. Compatible with Apache-2.0.
* **GPL-3.0** strong copyleft. Any project linking against the code must be GPL.
  Too restrictive for provider libraries.

MPL-2.0 is also used by HashiCorp's own `terraform-plugin-framework` and
`terraform-plugin-codegen-openapi`.


## Decision

License the provider under MPL-2.0.


## Consequences

* Anyone who modifies the provider's source files must publish those modifications under MPL-2.0.
  Forking and improving the provider privately without contributing back is not permitted.
* The provider can incorporate Apache-2.0 code (e.g. from dikhan's provider) by keeping those
  files under their original license.
* Compatible with the Terraform ecosystem: HashiCorp's own tooling uses MPL-2.0, so there is no
  license friction for consumers.
