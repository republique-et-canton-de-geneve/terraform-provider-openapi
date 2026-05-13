# Changelog


## Release v0.1.3 (2026-05-13)

Diff: https://github.com/republique-et-canton-de-geneve/terraform-provider-openapi/compare/v0.1.2...v0.1.3

### Features

* Add acceptance test suite for the `widget` resource; see [Acceptance tests](README.md#acceptance-tests)
  for setup and local-run instructions
* Add GitHub Actions CI workflow running acceptance tests against Terraform 1.13, 1.14, and 1.15
  with all action steps pinned to commit SHAs

### Fix and enhancements

* Treat HTTP 404 as success on `DELETE`: a resource already absent is considered destroyed,
  preventing spurious errors on `terraform destroy` or the `disappears` test
* Fix `SA5011` staticcheck false positives in test files: add `return` after `t.Fatal` nil guards


## Release v0.1.2 (2026-05-12)

Diff: https://github.com/republique-et-canton-de-geneve/terraform-provider-openapi/compare/v0.1.1...v0.1.2

### Features

* Implement `x-unordered` extension for array fields: API may return items in any order without
  causing Terraform drift; combined with the standard OAS `uniqueItems` keyword it selects the
  appropriate Terraform type and validation strategy:
    * `x-unordered: true` alone → sorted `List` (elements sorted on read and at plan time)
    * `uniqueItems: true` alone → `List` with a `UniqueItems` validator that rejects duplicate elements
    * `x-unordered: true` + `uniqueItems: true` → Terraform `Set` (unordered and unique by construction)

### Fix and enhancements

* Refactor field-spec helpers into focused functions:
    * `isComputedField`
    * `isImmutableField`
    * `isWritableField`
    * `isUnorderedField`
    * `boolExtension`
* Move `toSnakeCase` to a dedicated `strings.go` file
* Add comprehensive unit tests for all new and refactored field-spec behaviour


## Release v0.1.1 (2026-05-11)

### Features

* Implement `x-timeout` extension: per-operation timeout defaults on OAS3 operations exposed
  as a `timeouts` block on each resource; values are applied as context deadlines on all HTTP calls
* Add `list` timeout read from the collection-path GET and applied to data source reads
* Validate `x-timeout` spec values at provider startup; non-positive or unparseable values
  cause an immediate error
* Validate user-supplied `timeouts` block values at plan time; values must be valid Go durations
  greater than zero (`positiveDuration` validator)


## Release v0.1.0 (2026-05-08)

Initial release.
