# `x-tf-name`


## Properties

* **Scope**: resource (on the collection POST operation)
* **Value**: string


## Description

Overrides the Terraform resource type name derived from the path
(`openapi_cdns` → `openapi_content_delivery_networks`). Not yet scheduled; the same outcome is
achievable today by naming API paths deliberately or via `OPENAPI_PREFIX`. Will be reconsidered
if the provider gains consumers whose path naming is outside their control.


## Prior art

dikhan `x-terraform-resource-name` · Speakeasy `x-speakeasy-name-override`.
