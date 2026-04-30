# OAS3 concepts and terminology

A reference for the vocabulary used throughout this codebase and its docs. The word "resource"
is notably absent from OAS3 itself. It is a REST concept used informally, not a formal OAS term.

Official specification: [OpenAPI Specification 3.0](https://spec.openapis.org/oas/v3.0.3)


## Schema

A JSON Schema-based definition of a data shape: properties, types, and validation rules. Schemas
describe the structure of request/response bodies and parameters. Stored under
`components/schemas/` when reused.


## Path

A URL template such as `/users/{id}`. The top-level `paths` object maps path templates to one or
more operations.


## Operation

A specific HTTP method bound to a path (`GET /users`, `POST /users`). Each operation carries an
optional `operationId`, a list of parameters, an optional request body, and a map of responses.


## Parameter

Input passed outside the request body, via one of four locations:

* `path`: a segment of the URL template, e.g. `{id}`
* `query`: a URL query string value
* `header`: an HTTP request header
* `cookie`: an HTTP cookie

Each parameter has a schema that describes its type and constraints.


## Request body

The payload sent with `POST`, `PUT`, or `PATCH`. Defined under `requestBody` as a `content` map
from media type to schema.


## Response

The reply for a given HTTP status code. Also defined as a `content` map, plus optional `headers`.


## Content

Maps media types (`application/json`, `multipart/form-data`, ...) to a schema. Used inside both
`requestBody` and `responses`.


## Component

A named, reusable definition stored under `components/`. The sub-sections most relevant to this
provider are:

| Sub-section | What it holds |
|---|---|
| `schemas` | Data shape definitions |
| `parameters` | Reusable parameter definitions |
| `responses` | Reusable response definitions |
| `requestBodies` | Reusable request body definitions |
| `headers` | Reusable header definitions |
| `securitySchemes` | Auth mechanism definitions |
| `pathItems` | Reusable path-level operation sets |


## Reference (`$ref`)

A JSON Pointer that splices a component inline, e.g. `$ref: '#/components/schemas/User'`.
References keep the spec DRY and are the main mechanism by which the provider resolves schema
definitions.


## Tag

A label that groups related operations. Tags are cosmetic; they affect tooling like Swagger UI
but carry no semantic meaning in the spec itself.


## Security scheme

Defines an authentication mechanism. OAS3 supports four types:

* `apiKey`: a key passed in a header, query parameter, or cookie
* `http`: HTTP auth schemes such as `basic` or `bearer`
* `oauth2`: OAuth 2.0 flows
* `openIdConnect`: OpenID Connect discovery


## Server

A base URL from which the API is served. A spec may list multiple servers (production, staging,
local) and operations may override the server list.


## Info / ExternalDocs

Metadata about the API: title, version, license, contact details, and a link to external
documentation. Not used by this provider at runtime.
