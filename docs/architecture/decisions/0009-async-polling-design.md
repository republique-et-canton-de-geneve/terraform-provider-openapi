# 9. Async polling design: state machine detection and target state signalling

Date: 2026-04-30


## Status

Undecided


## Context

When an API returns `202 Accepted` for a create, update, or delete operation, the provider must
poll the resource until it reaches a target state. This requires three things:

1. **Detecting that an operation is async**: OAS3 already covers this; a `202` response on the
   operation signals async behaviour. No extension needed.
2. **Finding the status field**: the field in the response body whose value indicates the current
   state of the resource. Drafted: `x-tf-status` on the schema object (not the field), with a
   fallback heuristic for fields named `status` or `state`. See the `x-tf-status` entry in
   [extensions/planned/x-tf-status.md](../extensions/planned/x-tf-status.md).
3. **Knowing which values are target, error, or pending**: OAS3 exposes all possible values via
   `enum` on the status field, but does not distinguish the three categories the provider needs:
    * **Target states**: values that stop polling and signal success (e.g. `running`, `active`).
    * **Error states**: values that stop polling and raise an error (e.g. `error`, `failed`).
    * **Pending states**: values that keep polling (e.g. `creating`, `provisioning`, `deleting`).

Note that a target state is not necessarily a "final" state in the lifecycle sense. `running` is a
long-lived operational state, not an end-of-life state; it is a target state only in the context of
a create operation. For soft-delete APIs, `deleted` may be the target state of a delete operation
even though the resource record still exists.

### OAS3 boundary: what spec-driven polling cannot cover

OAS3 describes individual operations, not workflows. It tells you that `POST /vms/` creates a VM and
that `DELETE /vms/{id}` removes it, but it does not describe:

* **Action endpoints**: `POST /vms/{id}/start` and `POST /vms/{id}/stop` are transitions in a
  state machine. OAS3 does not express that calling `/start` moves the resource from `stopped`
  to `running`, nor that the provider should poll for `running` after that call.
* **Soft delete**: if `DELETE /vms/{id}` sets `state: deleted` rather than removing the record,
  OAS3 does not signal this. The provider has no way to know whether a 200/204 response means
  "gone" or "marked deleted".
* **Multi-step workflows**: powering off a VM before snapshotting it, or waiting for a
  dependency to reach a ready state before creating a dependent resource. None of this is
  expressible in OAS3.

The [magodo/restful provider](https://registry.terraform.io/providers/magodo/restful/latest/docs)
takes the opposite approach: it reads no spec at all and requires the operator to declare all
polling and lifecycle behaviour explicitly in HCL per resource. This makes it flexible but not
dynamic. Reviewing its design may inform what edge cases a complete polling implementation needs
to handle.

The dynamic provider can automate polling for the standard CRUD lifecycle: create, update, and
delete each poll until the resource reaches its target state or is gone. Workflows beyond that
boundary are out of scope for a spec-driven approach.

### Future direction: transition routing

The problems above share a deeper gap: knowing which operation to call to move a resource from one
state to another. Static providers solve this by hardcoding the state machine in provider code
(how other providers handle this needs investigation). The long-term ambition for this provider is
to express state machine transitions in the spec itself, so the provider can build the transition
table, and API to call, dynamically at runtime.

This is not a commitment. The design must be investigated and will evolve; no guarantee is made that
it can be fully realised within a spec-driven model. The initial implementation covers polling
only.

See also [architecture/state-machine.md](../state-machine.md) for a broader discussion of how
OAS3 relates to resource state machines and what design we will implement to fullfill requirements.


## Decision

* First implementation will be designed by working on internal Django REST Framework APIs,
  which follow consistent OAS3 conventions and provide a controlled reference for validating
  the state-machine design primarily for provisioning purposes (desired vs current state).

No decision recorded yet.


## Consequences

* `x-tf-status` remains in TBD status until this ADR is resolved.
* Implementation of async polling is blocked on this decision.
