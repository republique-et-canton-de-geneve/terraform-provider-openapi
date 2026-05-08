# Discoverability

Since this provider is fully dynamic, there are no pre-generated resource docs.

Two built-in mechanisms let you discover what resources and data sources are available for a given
OpenAPI spec. Helping you debugging any issues you could face.


## 1. Terraform `<prefix>_manifest` data source

A built-in data source is always registered alongside the dynamic ones.

It returns structured data about every discovered resource, w/o calling the API except the call for
retrieving the OAS3 spec if not provided as file.

```hcl
data "openapi_manifest" "all" {}

output "available_resources" {
  value = data.openapi_manifest.all.resources
}
```

Each entry in `resources` has:

| Attribute | Description |
|---|---|
| `resource_type` | Terraform resource type name, e.g. `openapi_linux_vm_instance` |
| `datasource_type` | Terraform data source type name; empty if no list endpoint |
| `item_path` | OpenAPI item path template |
| `list_path` | OpenAPI collection path; empty if no list endpoint |
| `can_create` | Supports POST |
| `can_update` | Supports PATCH or PUT |
| `can_delete` | Supports DELETE |

Run `terraform plan` then `terraform apply` to see the output, or use it in CI to validate what
types are available before writing resource blocks.


## 2. Terraform debug logging

Set `TF_LOG=DEBUG` before any Terraform command to see a log line for each registered type:

```sh
TF_LOG=DEBUG terraform plan 2>&1 | grep "registered"
```

Output:

```
@level=debug @message="registered resource"   type=openapi_linux_vm_instance
@level=debug @message="registered data source" type=openapi_linux_vm_instances
@level=debug @message="registered data source" type=openapi_manifest
```

This requires no extra configuration and works in the environment where Terraform runs the provider.
