# `x-immutable`


## Properties

* **Scope**: field
* **Value**: `true`


## Description

Marks a field that can be set at creation time but cannot be changed afterward. The provider
attaches the `RequiresReplace` plan modifier, so Terraform will destroy and recreate the resource
when the field value changes.


## Example

```yaml
vlan_id:
  type: integer
  x-immutable: true
```


## Prior art

**Kept bare** because this semantic is API-intrinsic (the field is write-once by the API's own
rules) and has community convergence: ReDoc renders it, and OAS issue #2720 proposes standardising
it. The name has 640+ real-world usages with compatible meaning.

dikhan `x-terraform-immutable` and `x-terraform-force-new` (collapsed into one; the distinction
was conceptual only, the Terraform behaviour is identical) · ReDoc renders `x-immutable: true`
in generated documentation · Azure equivalent: `x-ms-mutability: ["create"]` (array-based, also
covers read-only and update-only combinations) · OAS issue [#2720] proposes native support in a
future spec version.

[#2720]: https://github.com/OAI/OpenAPI-Specification/issues/2720
