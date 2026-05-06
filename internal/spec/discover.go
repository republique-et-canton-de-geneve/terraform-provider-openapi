package spec

import (
	"log"
	"strings"
	"unicode"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	yaml "go.yaml.in/yaml/v4"
)

type pathInfo struct {
	path  string
	hasID bool
	item  *v3high.PathItem
}

type resourceGroup struct {
	listPath *pathInfo
	itemPath *pathInfo
}

// DiscoverResources walks the OAS3 paths and returns a ResourceSpec for each detected resource
// (collection + item path pair with a readable item schema).
func DiscoverResources(model *libopenapi.DocumentModel[v3high.Document]) []*ResourceSpec {
	if model == nil || model.Model.Paths == nil {
		return nil
	}

	var allPaths []string
	for path := range model.Model.Paths.PathItems.FromOldest() {
		allPaths = append(allPaths, path)
	}
	prefix := findCommonPathPrefix(allPaths)

	groups := map[string]*resourceGroup{}
	for path, pathItem := range model.Model.Paths.PathItems.FromOldest() {
		resource, hasID := splitResourcePath(path, prefix)
		if resource == "" {
			continue
		}
		if _, ok := groups[resource]; !ok {
			groups[resource] = &resourceGroup{}
		}
		info := &pathInfo{path: path, hasID: hasID, item: pathItem}
		if hasID {
			groups[resource].itemPath = info
		} else {
			groups[resource].listPath = info
		}
	}

	var specs []*ResourceSpec
	for name, g := range groups {
		if g.itemPath == nil {
			continue // need the item path for Read
		}
		spec := buildResourceSpec(name, g.listPath, g.itemPath)
		if spec != nil && len(spec.Fields) > 0 {
			specs = append(specs, spec)
		}
	}
	return specs
}

// buildResourceSpec assembles a ResourceSpec from the list and item path entries.
func buildResourceSpec(name string, listInfo, itemInfo *pathInfo) *ResourceSpec {
	singular := singularizeName(name)
	rs := &ResourceSpec{SingularName: singular, PluralName: pluralizeName(singular)}

	// Paths
	if listInfo != nil {
		rs.ListPath = normalizePath(listInfo.path)
	}
	rs.ItemPath = normalizePath(itemInfo.path)
	matches := pathParamRE.FindAllStringSubmatch(rs.ItemPath, -1)
	if len(matches) > 0 {
		rs.IDPathParam = matches[len(matches)-1][1]
	}

	// Capabilities
	if listInfo != nil && listInfo.item.Post != nil {
		rs.HasCreate = true
	}
	if itemInfo.item.Patch != nil {
		rs.HasUpdate = true
		rs.UpdateMethod = "PATCH"
	} else if itemInfo.item.Put != nil {
		rs.HasUpdate = true
		rs.UpdateMethod = "PUT"
	}
	if itemInfo.item.Delete != nil {
		rs.HasDelete = true
	}

	// Schema: use GET /{id}/ 200 response
	itemSchema := extractResponseSchema(itemInfo.item.Get)
	if itemSchema == nil {
		return nil
	}
	writeFields := map[string]bool{}
	if listInfo != nil && listInfo.item.Post != nil {
		writeFields = extractRequestBodyFields(listInfo.item.Post)
	}
	rs.Fields = buildFieldSpecs(name, itemSchema, writeFields)

	rs.IDField = "id"

	return rs
}

// extractResponseSchema returns the JSON schema from the first 200/201 response of an operation.
func extractResponseSchema(op *v3high.Operation) *base.Schema {
	if op == nil || op.Responses == nil || op.Responses.Codes == nil {
		return nil
	}
	for _, code := range []string{"200", "201"} {
		resp, ok := op.Responses.Codes.Get(code)
		if !ok || resp == nil || resp.Content == nil {
			continue
		}
		mt, ok := resp.Content.Get("application/json")
		if !ok || mt == nil || mt.Schema == nil {
			continue
		}
		if s := mt.Schema.Schema(); s != nil {
			return s
		}
	}
	return nil
}

// extractRequestBodyFields returns the top-level property names from the JSON request body schema.
func extractRequestBodyFields(op *v3high.Operation) map[string]bool {
	fields := map[string]bool{}
	if op.RequestBody == nil || op.RequestBody.Content == nil {
		return fields
	}
	mt, ok := op.RequestBody.Content.Get("application/json")
	if !ok || mt == nil || mt.Schema == nil {
		return fields
	}
	schema := mt.Schema.Schema()
	if schema == nil || schema.Properties == nil {
		return fields
	}
	for propName := range schema.Properties.FromOldest() {
		fields[propName] = true
	}
	return fields
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
