# `x-tf-exclude`


## Properties

* **Scope**: resource (on the collection POST operation)
* **Value**: `true`


## Description

Prevents a resource from being registered in the provider. Useful for hiding internal admin
endpoints, action endpoints, or paths that exist in the spec but are not intended for Terraform
management.


## Example

```yaml
/internal/admin/:
  post:
    x-tf-exclude: true
```


## Prior art

**Namespaced** because the decision to hide something from a Terraform provider is purely a
Terraform concern; it has no meaning for documentation generators or SDK tools.

dikhan `x-terraform-exclude-resource` (resource-level only) · HashiCorp codegen addresses the
same need via an `ignores` list in its generator config file rather than an in-spec extension.
