# `x-timeout`


## Properties

* **Scope**: operation (GET, POST, PUT/PATCH, DELETE) per resource
* **Value**: duration string e.g. `"30m"`


## Description

Sets the recommended default timeout for the corresponding Terraform action (create, read,
update, delete). Each HTTP operation carries its own value, mapping directly to Terraform's
per-action timeout block. When absent, Terraform applies a 20-minute default for all actions.
Only meaningful once async polling (`x-tf-status`) is active; without polling, operations
complete synchronously and the timeout is never reached.


## Example

```yaml
/vms/:
  post:
    x-timeout: "30m"   # create
/vms/{id}:
  get:
    x-timeout: "5m"    # read
  put:
    x-timeout: "15m"   # update
  delete:
    x-timeout: "10m"   # delete
```

Users can still override any action via the `timeouts` block in the resource declaration:

```hcl
resource "openapi_vm" "my_vm" {
  timeouts {
    create = "60m"
    read   = "10m"
    update = "20m"
    delete = "15m"
  }
}
```


## Prior art

No equivalent found. dikhan `x-terraform-resource-timeout` is a coarser single-value variant
(per resource, not per action); the per-action design here is a new improvement.
