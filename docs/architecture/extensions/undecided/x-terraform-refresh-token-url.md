# `x-terraform-refresh-token-url`


## Description

Implements a refresh-token exchange flow inside the provider: before each operation the provider
POSTs to a token endpoint and uses the returned session token. This solves short-lived tokens
expiring mid-run without requiring the operator to rotate `OPENAPI_TOKEN` manually.


## Prior art

dikhan only.
