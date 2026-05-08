# `x-primary-key`


## Properties

* **Scope**: schema
* **Value**: field name or format string e.g. `"id"`, `"{org_id}/{user_id}"`


## Description

Declares the primary key of the resource at the schema level. The value is either a plain field
name (single key) or a format string with field names in `{braces}` (composite key). The
separator embedded in the format string is also the separator used for `terraform import`.

When absent, the provider falls back to a field named `id`.

See [primary-key.md](../../../primary-key.md) for a detailed discussion and
[ADR 0008](../../decisions/0008-primary-key-design.md) for the decision record.


## Example

```yaml
components:
  schemas:
    # Single key with non-conventional name
    User:
      type: object
      x-primary-key: "guid"

    # Composite key — separator is also the terraform import separator
    ProjectVariable:
      type: object
      x-primary-key: "{project}:{key}:{environment_scope}"
```


## Prior art

**Kept bare** because a primary key is an API-intrinsic concept meaningful to any consumer of
the spec: SDK generators, ORM mappers, documentation tools. It is not Terraform-specific.

Django `primary_key=True` (field-level, single key only) · database `PRIMARY KEY` constraint ·
No prior art found for a schema-level composite key extension in OpenAPI · dikhan
`x-terraform-id` (field-level only, no composite support).
