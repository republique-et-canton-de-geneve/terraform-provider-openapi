# `x-tf-status`


## Properties

* **Scope**: schema
* **Value**: field name string e.g. `"state"`


## Description

Identifies the single field holding the resource's async operation status, used by the polling
mechanism when an API returns `202 Accepted`. The provider watches this field after
create/update/delete until it reaches a non-transitive value.

Placed on the schema object (not the field), the value is the name of the status field. The
provider resolves the status field using the following priority:

1. Schema carries `x-tf-status: "<field_name>"` — explicit pointer.
2. Schema has a field named `status`.
3. Schema has a field named `state`.
4. None found; async polling is disabled for this resource.

The extension is only needed when the field has an unconventional name. The companion extensions
`x-tf-status-complete` and `x-tf-status-pending` (or an alternative approach) are still
undecided; see [ADR 0009](../../decisions/0009-async-polling-design.md).


## Example

```yaml
components:
  schemas:
    VM:
      type: object
      x-tf-status: "operational_state"
      properties:
        operational_state:
          type: string
          readOnly: true
```


## Prior art

**Namespaced** because synchronous polling to make async calls sync is a Terraform execution
mechanic with no meaning outside the provider runtime.

dikhan `x-terraform-field-status` + `x-terraform-resource-poll-enabled` +
`x-terraform-resource-poll-completed-statuses` + `x-terraform-resource-poll-pending-statuses`
(four extensions for the same feature, consolidated here into `x-tf-status` with companion
`x-tf-status-complete` / `x-tf-status-pending`) · Azure `x-ms-long-running-operation` (boolean
on the operation, status tracked via `Location` / `Azure-AsyncOperation` response headers rather
than a response body field) · magodo/restful provider (resource-level polling configuration,
not spec-driven).
