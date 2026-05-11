# OAS3 Extensions

This provider recognises a set of OpenAPI vendor extensions (`x-`) that give spec authors
fine-grained control over how fields and resources and workflows are exposed in Terraform.


## Naming

See [ADR 0006](../decisions/0006-extension-naming-bare-vs-x-tf.md) for the full decision record.
In summary, extensions fall into two categories:

* **Bare names** (`x-immutable`, `x-sensitive`, `x-computed`, `x-ignore-order`, `x-primary-key`,
  `x-timeout`): semantics that are API-intrinsic and tool-agnostic. These names have community
  convergence (ReDoc, open OAS issues, or no prior conflicting usage) and are meaningful to any
  consumer of the spec, not just Terraform.

* **`x-tf-` prefix** (`x-tf-exclude`, `x-tf-status`): semantics that are specific to how this
  provider maps the API to Terraform's resource model. Another tool reading the same spec would
  have no use for these extensions.


## Summary

| Behavior | Extension | Scope | Status | dikhan equivalent | Prior art / equivalents |
|---|---|---|---|---|---|
| Primary key (single or composite) | [`x-primary-key`](planned/x-primary-key.md) | schema | planned | `x-terraform-id` | Django `primary_key=True` · database PRIMARY KEY |
| Field holding async operation status | [`x-tf-status`](undecided/x-tf-status.md) | schema | undecided | `x-terraform-field-status` + 3 companions | Azure `x-ms-long-running-operation` |
| Exclude resource from the provider | [`x-tf-exclude`](undecided/x-tf-exclude.md) | resource | undecided | `x-terraform-exclude-resource` | HashiCorp codegen `ignores` config |
| Override resource type name in Terraform | [`x-tf-name`](undecided/x-tf-name.md) | resource | undecided | `x-terraform-resource-name` | Speakeasy `x-speakeasy-name-override` |
| ~~Per-resource base URL override~~ | [`x-terraform-resource-host`](rejected/x-terraform-resource-host.md) | resource | rejected | `x-terraform-resource-host` | — |
| Recommended default timeout per resource action | [`x-timeout`](implemented/x-timeout.md) | operation | implemented | — | — |
| Optional field whose default is set by the server | [`x-computed`](implemented/x-computed.md) | field | implemented | `x-terraform-computed` | — |
| Field cannot be changed after creation | [`x-immutable`](implemented/x-immutable.md) | field | implemented | `x-terraform-immutable`, `x-terraform-force-new` | 640+ usages · ReDoc · OAS [#2720] · Azure `x-ms-mutability: ["create"]` |
| Array field where item order is insignificant | [`x-ignore-order`](planned/x-ignore-order.md) | field | planned | `x-terraform-ignore-order` | — |
| Field value is redacted in plan and state | [`x-sensitive`](implemented/x-sensitive.md) | field | implemented | `x-terraform-sensitive` | No prior art found; name heuristics widely used instead |
| Refresh token exchange flow | [`x-terraform-refresh-token-url`](undecided/x-terraform-refresh-token-url.md) | security | undecided | `x-terraform-refresh-token-url` | — |
| ~~Bearer token formatting~~ | [`x-terraform-authentication-scheme-bearer`](rejected/x-terraform-authentication-scheme-bearer.md) | security | rejected | `x-terraform-authentication-scheme-bearer` | OAS3 `http/bearer` scheme is the standard |
| ~~Multi-region base URL parameterisation~~ | [`x-terraform-provider-multiregion-fqdn`](rejected/x-terraform-provider-multiregion-fqdn.md) | provider | rejected | `x-terraform-provider-multiregion-fqdn` + companions | Terraform provider aliases |

Sorted by scope (schema, resource, operation, field, security, provider), then alphabetically.

[#2720]: https://github.com/OAI/OpenAPI-Specification/issues/2720
