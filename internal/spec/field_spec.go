package spec

import (
	"log"
	"strings"
	"unicode"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	yaml "go.yaml.in/yaml/v4"
)

// toSnakeCase converts a camelCase or PascalCase string to snake_case.
// "photoUrls" -> "photo_urls", "petId" -> "pet_id", "APIKey" -> "api_key".
// Already-snake strings pass through unchanged.
func toSnakeCase(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				// Standard camelCase boundary (lower->upper): "petId" -> "pet_Id"
				// Acronym end (upper+upper->lower): "APIKey" -> "API_Key"
				if unicode.IsLower(prev) ||
					(unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
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

// buildFieldSpecs converts an OAS3 object schema into a slice of FieldSpecs,
// marking fields absent from writeFields as computed.
func buildFieldSpecs(
	resourceName string,
	schema *base.Schema,
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
				resourceName,
				propName,
				propProxy.Schema(),
				writeFields[propName],
				requiredSet[propName]))
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
	resourceName string,
	name string,
	schema *base.Schema,
	writable bool,
	required bool,
) *FieldSpec {
	f := &FieldSpec{OASName: name, Name: toSnakeCase(name)}

	// Type
	if schema == nil {
		f.Type = "string"
		f.Computed = !writable
		return f
	}
	f.Type = detectType(schema)
	f.Format = schema.Format

	// Behaviour: Writable set first as Computed and Required derive from it
	f.Writable = writable
	f.Required = required && writable
	// readOnly: true means server-set; not writable regardless of POST body presence.
	if schema.ReadOnly != nil && *schema.ReadOnly {
		f.Computed = true
		f.Writable = false
	}
	if schema.Extensions != nil {
		if node, ok := schema.Extensions.Get("x-immutable"); ok && node != nil &&
			node.Value == "true" {
			f.Immutable = true
		}
	}
	if !f.Writable && !writable {
		f.Computed = true
	}
	if schema.Extensions != nil {
		if node, ok := schema.Extensions.Get("x-computed"); ok && node != nil {
			if node.Value == "true" {
				f.Computed = true
			} else {
				log.Printf(
					"[WARN] resource %q field %q: x-computed: %q is not supported; "+
						"only x-computed: true is recognised",
					resourceName, name, node.Value)
			}
		}
	}
	f.Sensitive = isSensitiveField(name, schema)

	// Metadata
	f.Description = schema.Description

	// Validation constraints
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
	f.Default = decodeDefaultNode(schema.Default, f.Type)

	// Nested fields for objects
	if f.Type == "object" && schema.Properties != nil {
		nestedRequired := map[string]bool{}
		for _, r := range schema.Required {
			nestedRequired[r] = true
		}
		for propName, propProxy := range schema.Properties.FromOldest() {
			f.Nested = append(f.Nested,
				buildFieldSpec(
					resourceName,
					propName,
					propProxy.Schema(),
					true,
					nestedRequired[propName]))
		}
	}

	// Element type for arrays
	if f.Type == "array" && schema.Items != nil && schema.Items.IsA() && schema.Items.A != nil {
		f.ItemSpec = buildFieldSpec(resourceName, "item", schema.Items.A.Schema(), true, false)
	}

	return f
}

// decodeDefaultNode converts a yaml.Node (from OAS3 `default:`) into a typed Go value.
// Returns nil for unsupported shapes (null, mappings, non-empty sequences).
func decodeDefaultNode(node *yaml.Node, fieldType string) any {
	if node == nil || node.Tag == "!!null" {
		return nil
	}
	if node.Kind == yaml.SequenceNode {
		if fieldType == "array" {
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

func enumFromSchema(schema *base.Schema) []string {
	var vals []string
	for _, node := range schema.Enum {
		if node != nil {
			vals = append(vals, node.Value)
		}
	}
	return vals
}
