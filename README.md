# Terraform Provider OpenAPI (Dynamic)

A dynamic Terraform provider that generates resource types at runtime from an OpenAPI specification.

Point it at any OAS3 (version 3) spec and it exposes every discoverable resources as Terraform
resources and data sources, with support for custom (`x-immutable`, `x-sensitive`, …) extensions.

The provider is [published here][registry].

It has been developped by the Cloud & Platform Engineering Team from the IT department of the
State of Geneva (Switzerland).

_This provider is built on the [Terraform Plugin Framework][tpf].
See [Which SDK Should I Use?][sdk] in the Terraform documentation for additional information._

For usage documentation see [docs/index.md](docs/index.md) or the
[Terraform Registry page][registry].


## Why not dikhan/terraform-provider-openapi?

[dikhan/terraform-provider-openapi](https://github.com/dikhan/terraform-provider-openapi) was
evaluated but not used for three reasons:

1. **Legacy SDK.** It is built on `terraform-plugin-sdk/v2`, which Hashicorp considers superseded.
   This provider uses the current [Terraform Plugin Framework][tpf], the recommended path for new
   and maintained providers.

2. **OpenAPI 2 (Swagger) only.** dikhan's provider explicitly rejects OAS3 specs at runtime, and
   no fork in its ecosystem has ever added OAS3 support. This provider targets OAS3 only, which
   is what our internal APIs expose.

3. **No active maintenance.** The upstream project has seen very little activity in recent years
   and does not track the plugin-framework migration that Hashicorp has been pushing.


## Requirements

* [Terraform][terraform-downloads] >= 1.8
* [Go][go-install] >= 1.25


## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```


## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.


## Developing the Provider

If you wish to work on the provider, you'll first need [Go][go-install] installed on your machine
(see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider
binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate ./...`.
To format the code run `make fmt`.

### Unit tests

```shell
make test
```

### Linting

```shell
make lint
```

### Acceptance tests

Acceptance tests create and destroy real resources on a live API instance.

Set the required environment variables:

```shell
export OPENAPI_SPEC=/path/to/spec.yaml
export OPENAPI_URL=https://api.example.com/v1
export OPENAPI_TOKEN=my-token
```

Then run:

```shell
make testacc
```

### Running locally

`go run . -debug` starts the provider as a long-running process and prints a
`TF_REATTACH_PROVIDERS` value. Terraform picks that up and connects to your
process instead of launching its own binary -- no installation step needed.

**Terminal 1** -- start the provider:

```shell
go run . -debug
# Provider server started; to attach Terraform, set the TF_REATTACH_PROVIDERS
# environment variable in your terminal session:
#
#   TF_REATTACH_PROVIDERS='{"registry.terraform.io/republique-et-canton-de-geneve/openapi":{"Protocol":"grpc","ProtocolVersion":6,"Pid":12345,"Test":true,"Addr":{"Network":"unix","String":"/tmp/plugin-123.sock"}}}'
```

**Terminal 2** -- export the value printed above, then run Terraform normally:

```shell
# Using the public Swagger Petstore as a ready-made OAS3 target
export OPENAPI_SPEC=https://petstore3.swagger.io/api/v3/openapi.json
export OPENAPI_URL=https://petstore3.swagger.io/api/v3
export TF_REATTACH_PROVIDERS='...'   # paste from terminal 1

terraform init
terraform plan
```

The provider discovers `openapi_pet`, `openapi_store_order`, and `openapi_user` from the
Petstore spec at init time. A matching `main.tf`:

```hcl
terraform {
  required_providers {
    openapi = {
      source = "registry.terraform.io/republique-et-canton-de-geneve/openapi"
    }
  }
}

provider "openapi" {}

resource "openapi_pet" "clifford" {
  name       = "Clifford"
  photo_urls = ["https://example.com/clifford.jpg"]
  status     = "available"
  category   = {
    id   = 1
    name = "dog"
  }
  tags = [
    { id = 1, name = "big" },
    { id = 2, name = "red" },
  ]
}

resource "openapi_store_order" "first" {
  pet_id   = openapi_pet.clifford.id
  quantity = 1
  status   = "placed"
}
```

OAS3 property names are converted to snake_case (`photoUrls` -> `photo_urls`, `petId` -> `pet_id`).
The provider translates back to camelCase when writing to the API.

The provider process in terminal 1 stays alive across multiple `terraform plan` or `apply` calls.
Restart it (Ctrl-C, then `go run . -debug` again) whenever you rebuild after a code change.

Use `TF_LOG=DEBUG` to see structured API call logs from the provider.

[go-install]: https://golang.org/doc/install
[registry]: https://registry.terraform.io/providers/republique-et-canton-de-geneve/openapi/latest
[sdk]: https://developer.hashicorp.com/terraform/plugin/framework-benefits
[terraform-downloads]: https://developer.hashicorp.com/terraform/downloads
[tpf]: https://github.com/hashicorp/terraform-plugin-framework
