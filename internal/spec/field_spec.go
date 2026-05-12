package spec

import (
	"log"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	yaml "go.yaml.in/yaml/v4"
)

// buildFieldSpecs converts an OAS3 object schema into a slice of FieldSpecs,
// marking fields absent from writeFields as computed.
func buildFieldSpecs(
	schema *base.Schema,
	resourceName string,
	writeFields map[string]bool,
) []*FieldSpec {
	if schema == nil || schema.Properties == nil {
		return nil
	}

	requiredSet := map[string]bool{}
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	var fields []*FieldSpec
	for propName, propProxy := range schema.Properties.FromOldest() {
		fields = append(fields,
			buildFieldSpec(
				propProxy.Schema(),
				resourceName,
				propName,
				requiredSet[propName],
				writeFields[propName]))
	}

	for _, f := range fields {
		if f.Name == "id" {
			f.IsID = true
			break
		}
	}

	return fields
}

// buildFieldSpec converts a single OAS3 property schema into a FieldSpec.
func buildFieldSpec(
	schema *base.Schema,
	resourceName string,
	name string,
	required bool,
	writable bool,
) *FieldSpec {
	f := &FieldSpec{OASName: name, Name: toSnakeCase(name)}

	if schema == nil {
		f.Type = "string"
		f.Computed = !writable
		return f
	}

	// Type

	f.Type = detectType(schema)
	f.Format = schema.Format

	// Fields with no declared type and no structural hints accept any JSON value.
	if f.Type == "string" && len(schema.Type) == 0 &&
		len(schema.Enum) == 0 &&
		len(schema.AllOf) == 0 &&
		len(schema.OneOf) == 0 &&
		len(schema.AnyOf) == 0 {
		f.Type = "untyped"
	}

	// Behaviour

	fieldDesc := resourceName + " field " + name
	f.Writable = isWritableField(schema, writable) // must be first: others depend on it
	f.Computed = isComputedField(schema, f.Type, f.Writable, fieldDesc)
	f.Immutable = isImmutableField(schema, fieldDesc)
	f.Required = required && f.Writable
	f.Sensitive = isSensitiveField(schema, name)
	f.Unordered = isUnorderedField(schema, fieldDesc)
	if f.Type == "array" && schema.UniqueItems != nil && *schema.UniqueItems {
		f.UniqueItems = true
	}

	// Metadata

	f.Description = schema.Description

	// Miscellaneous

	f.Default = decodeDefaultNode(schema.Default, f.Type)

	// Validation

	if schema.MaxLength != nil {
		f.MaxLength = schema.MaxLength
	}
	if schema.MinLength != nil {
		f.MinLength = schema.MinLength
	}
	if schema.Pattern != "" {
		f.Pattern = schema.Pattern
	}
	if schema.Minimum != nil {
		f.Minimum = schema.Minimum
	}
	if schema.Maximum != nil {
		f.Maximum = schema.Maximum
	}
	f.Enum = extractEnumValues(schema)

	// Nested fields for objects
	if f.Type == "object" && schema.Properties != nil {
		nestedRequired := map[string]bool{}
		for _, r := range schema.Required {
			nestedRequired[r] = true
		}
		for propName, propProxy := range schema.Properties.FromOldest() {
			f.Nested = append(f.Nested,
				buildFieldSpec(
					propProxy.Schema(),
					resourceName,
					propName,
					nestedRequired[propName],
					true))
		}
	}

	// Element type for arrays
	if f.Type == "array" && schema.Items != nil && schema.Items.IsA() && schema.Items.A != nil {
		f.ItemSpec = buildFieldSpec(schema.Items.A.Schema(), resourceName, "item", false, true)
	}

	return f
}

// boolExtension reads a boolean OAS vendor extension from a schema.
// Returns (true, true) for "true", (false, true) for "false".
// Logs a warning and returns (false, false) when absent or any other value.
func boolExtension(schema *base.Schema, key, desc string) (value bool, found bool) {
	if schema == nil || schema.Extensions == nil {
		return false, false
	}
	node, ok := schema.Extensions.Get(key)
	if !ok || node == nil {
		return false, false
	}
	switch node.Value {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		log.Printf("[WARN] %s: %s: %q is not supported; only true or false are recognised",
			desc, key, node.Value)
		return false, false
	}
}

// decodeDefaultNode converts a yaml.Node (from OAS3 `default:`) into a typed Go value.
// Returns nil for unsupported shapes (null, mappings, non-empty sequences).
func decodeDefaultNode(node *yaml.Node, fieldType string) any {
	if node == nil || node.Tag == "!!null" {
		return nil
	}
	if node.Kind == yaml.SequenceNode {
		if fieldType == "array" || fieldType == "untyped" {
			return []any{} // only empty-array defaults are supported
		}
		return nil
	}
	if node.Kind != yaml.ScalarNode {
		return nil
	}
	switch fieldType {
	case "integer":
		var v int64
		if node.Decode(&v) == nil {
			return v
		}
	case "number":
		var v float64
		if node.Decode(&v) == nil {
			return v
		}
	case "boolean":
		var v bool
		if node.Decode(&v) == nil {
			return v
		}
	}
	return node.Value
}

// detectType infers the primary OAS3 type from a schema, preferring structural cues
// (items means array, properties means object) over the declared type string.
func detectType(schema *base.Schema) string {
	// Items present: treat as array (checked before properties to handle array-of-objects)
	if schema.Items != nil {
		return "array"
	}
	// Properties present: treat as object
	if schema.Properties != nil {
		return "object"
	}
	// Use declared type; skip "null" for nullable fields
	for _, t := range schema.Type {
		if t != "null" {
			return t
		}
	}
	return "string"
}

func enumFromSchema(schema *base.Schema) []string {
	var vals []string
	for _, node := range schema.Enum {
		if node != nil {
			vals = append(vals, node.Value)
		}
	}
	return vals
}

// extractEnumValues collects allowed string values from a schema's enum, allOf, or oneOf.
// Handles the DRF pattern of allOf:[{$ref:SomeEnum}] and oneOf:[{$ref:A},{$ref:B}].
func extractEnumValues(schema *base.Schema) []string {
	if schema == nil {
		return nil
	}
	if vals := enumFromSchema(schema); len(vals) > 0 {
		return vals
	}
	if len(schema.AllOf) == 1 {
		if inner := schema.AllOf[0].Schema(); inner != nil {
			if vals := enumFromSchema(inner); len(vals) > 0 {
				return vals
			}
		}
	}
	if len(schema.OneOf) > 0 {
		var result []string
		for _, proxy := range schema.OneOf {
			if inner := proxy.Schema(); inner != nil {
				result = append(result, enumFromSchema(inner)...)
			}
		}
		return result
	}
	return nil
}

// isComputedField returns whether a field should be marked Computed in Terraform.
// Priority (highest wins, no lower rule can override a higher one):
//  1. !writable — server owns the value; always computed
//  2. x-computed extension — spec author explicit override on a writable field
//  3. untyped + default — server will initialise; lowest priority, suppressible by x-computed: false
func isComputedField(schema *base.Schema, fieldType string, writable bool, desc string) bool {
	if !writable {
		return true
	}
	if v, found := boolExtension(schema, "x-computed", desc); found {
		return v
	}
	return fieldType == "untyped" && schema.Default != nil
}

// isImmutableField returns true when the spec declares x-immutable: true.
func isImmutableField(schema *base.Schema, desc string) bool {
	v, _ := boolExtension(schema, "x-immutable", desc)
	return v
}

// isUnorderedField returns true when the spec declares x-unordered: true.
func isUnorderedField(schema *base.Schema, desc string) bool {
	v, _ := boolExtension(schema, "x-unordered", desc)
	return v
}

// isWritableField returns whether a field is writable by the Terraform user.
// readOnly: true always wins: the server owns the value regardless of the POST body.
func isWritableField(schema *base.Schema, paramWritable bool) bool {
	if schema.ReadOnly != nil && *schema.ReadOnly {
		return false
	}
	return paramWritable
}

