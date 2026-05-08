# Resource state machines and OAS3


## What OAS3 expresses

An OpenAPI spec describes individual HTTP operations: their inputs, outputs, and response
codes. For a resource with a status field, it typically provides:

* A `202 Accepted` response on async operations (create, update, delete), signalling that the
  operation was accepted but not yet complete.
* An `enum` on the status field listing all possible values (e.g. `creating`, `running`,
  `stopped`, `error`, `deleted`).
* `readOnly: true` on the status field, indicating it is server-managed.

This is enough to detect async operations and enumerate the state universe.
It is not enough to drive a complete state machine.


## What OAS3 does not express

### Target vs pending vs error states

The `enum` lists all possible values but does not classify them. The provider needs to know:

* **Target states**: values that mean the operation succeeded and polling should stop
  (e.g. `running` after a create, `deleted` after a soft delete).
* **Error states**: values that mean the operation failed and the provider should raise an error
  (e.g. `error`, `failed`).
* **Pending states**: transitioning values that mean polling should continue
  (e.g. `creating`, `provisioning`, `deleting`).

A target state is not necessarily a "final" state in the resource lifecycle. `running` is a
long-lived operational state that a VM will leave again when stopped; it is a target state only
in the context of a create or start operation.

### Action endpoints

OAS3 does not describe that `POST /vms/{id}/start` transitions a VM from `stopped` to `running`,
nor that the provider should poll for `running` after that call. The operation exists in the spec
as a valid HTTP endpoint, but its relationship to the state machine is implicit.

The provider has no way to discover action endpoints automatically or know which state transition
they trigger. Supporting them would require explicit annotation beyond what OAS3 provides.

### Soft delete

If `DELETE /vms/{id}` marks the record as `state: deleted` rather than removing it, OAS3 does
not express this. A `200` or `204` response looks the same whether the resource is gone or merely
flagged. The provider cannot distinguish the two cases from the spec alone.

### Multi-step workflows

Sequencing operations (powering off a VM before snapshotting it, waiting for a network to be
ready before attaching a VM) is not expressible in OAS3. These workflows require knowledge of
the API's business logic that lives outside the spec. They are also outside Terraform's
responsibility: Terraform declares desired state; orchestrating a sequence of actions to reach
it is the concern of a workflow engine, not an infrastructure provider.


## What the provider has to automate

From what OAS3 does provide, the CRUD lifecycle can be partially automated:

* **Create**: POST → `202` detected → poll status field → target state reached → success.
* **Update**: PUT/PATCH → `202` detected → poll → target state reached → success.
* **Delete**: DELETE → `202` detected → poll → target state or resource gone → success.

Everything beyond this boundary requires information that OAS3 does not provide. The
[magodo/restful provider](https://registry.terraform.io/providers/magodo/restful/latest/docs)
is a useful reference: it covers a wide range of lifecycle patterns by pushing configuration
into HCL rather than deriving it from the spec, which is the trade-off when the spec is not
expressive enough.

This project defines a metadata scheme, expressed as vendor extensions in any OAS3 spec, that
allows the provider to identify the relevant endpoints, target states, and error conditions for
each lifecycle operation. API authors embed the extensions; the provider consumes them.

To be continued...


## Open questions

How to signal target, error, and pending states to the provider is unresolved. See
[ADR 0009](architecture/decisions/0009-async-polling-design.md) for the options under consideration.
