# Primary keys and resource identity


## What a primary key is

A primary key is the minimal set of fields that uniquely identifies a resource. In a relational
database it is declared with `PRIMARY KEY`; in Django it is `primary_key=True` on a model field.
In a REST API it is the value (or values) used to address a specific resource instance in a URL
and to uniquely identify it in Terraform state.


## Single primary keys

Most well-designed REST APIs assign a single primary key to every resource, regardless of the
resource's natural identity. The API generates an opaque `id` (integer, UUID, slug) at creation
time and uses it everywhere:

```
GET /api/variables/{id}/
```

This is the standard Django REST Framework pattern. The provider handles it with no configuration:
it falls back to a field named `id` automatically.


## Composite keys

Some APIs have no primary key. The resource's identity is the combination of multiple fields.

The canonical example is a GitLab project variable, whose identity is
`(project, key, environment_scope)`. There is no generated `id`; the API addresses the resource as:

```
GET /api/projects/{project}/variables/{key}?filter[environment_scope]={scope}
```

In Terraform, composite keys are expressed as a single import string with a separator:

```
terraform import gitlab_project_variable.example '12345:MY_VAR:*'
```

The separator is a convention chosen by the provider author; `:` and `/` are both common.


## How the provider handles primary keys

The provider resolves the primary key using the following priority:

1. Schema carries `x-primary-key` (single name or format string with composite fields in
   braces): `x-primary-key: "{project}:{key}:{environment_scope}"`
2. Schema has a field named `id`.
3. None found; CRUD operations fail at runtime with an explicit error.

The `x-primary-key` format string serves two purposes: it tells the provider which fields to
combine, and it documents the `terraform import` ID format directly in the spec.


## Why composite keys are rare in DRF APIs

Django REST Framework uses a single `pk` (primary key) as the default lookup field. A composite
lookup requires a custom `lookup_field` and a custom URL conf, which is uncommon enough to be a
deliberate design choice. APIs built on DRF almost always have a surrogate `id`.

Composite keys are more common in:

* APIs that expose relational junction tables directly (e.g. role-permission assignments).
* Cloud provider APIs that pre-date REST conventions (e.g. AWS IAM, some Azure resources).
* APIs where the natural key is human-meaningful and stable (e.g. GitLab project variables,
  where `key` is the variable name and changing it creates a new variable).


## Open questions

The implementation of `x-primary-key` is not yet scheduled. See
[ADR 0008](decisions/0008-primary-key-design.md) for the decision record.
