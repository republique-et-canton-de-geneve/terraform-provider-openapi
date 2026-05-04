// Package spec parses OpenAPI 3 specs and extracts resource definitions.
package spec

import "strings"

// FieldSpec describes one field of a resource for Terraform.
type FieldSpec struct {

	// Identity

	Name    string // Terraform attribute name (snake_case)
	OASName string // original OAS3 property name (camelCase)
	IsID    bool   // field named "id"; x-primary-key will supersede this when implemented

	// Type

	Type   string // "string", "integer", "number", "boolean", "object", "array"
	Format string // "date-time", "uuid", "int64", etc.

	// Behaviour

	Computed  bool // readOnly: true OR not in write body
	Immutable bool // x-immutable: true triggers RequiresReplace
	Required  bool // required in OAS schema and writable
	Sensitive bool // x-sensitive: true
	Writable  bool // present in POST request body

	// Metadata

	Description string

	// Nested

	Nested   []*FieldSpec // for type == "object"
	ItemSpec *FieldSpec   // for type == "array"
}

// ResourceSpec describes a discovered API resource.
type ResourceSpec struct {

	// Identity

	Name    string
	IDField string // name of the ID field, default "id"

	// Paths

	ListPath    string // collection endpoint, e.g. "/apidsi/v1/vlans/"
	ItemPath    string // item endpoint, e.g. "/apidsi/v1/vlans/{id}/"
	IDPathParam string // path parameter name in ItemPath, e.g. "id"

	// Capabilities

	HasCreate    bool
	HasUpdate    bool
	UpdateMethod string // "PATCH" or "PUT"
	HasDelete    bool

	// Schema

	Fields []*FieldSpec
}

// ResolvedItemPath substitutes the ID into the item path template.
func (self *ResourceSpec) ResolvedItemPath(id string) string {
	if self.IDPathParam == "" {
		return self.ItemPath
	}
	return strings.ReplaceAll(self.ItemPath, "{"+self.IDPathParam+"}", id)
}
