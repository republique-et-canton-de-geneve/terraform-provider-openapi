package spec

import (
	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
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

	// Timeouts
	if listInfo != nil {
		rs.Timeouts.List = operationTimeout(listInfo.item.Get)
		rs.Timeouts.Create = operationTimeout(listInfo.item.Post)
	}
	rs.Timeouts.Read = operationTimeout(itemInfo.item.Get)
	if itemInfo.item.Patch != nil {
		rs.Timeouts.Update = operationTimeout(itemInfo.item.Patch)
	} else {
		rs.Timeouts.Update = operationTimeout(itemInfo.item.Put)
	}
	rs.Timeouts.Delete = operationTimeout(itemInfo.item.Delete)

	// Schema: use GET /{id}/ 200 response
	itemSchema := extractResponseSchema(itemInfo.item.Get)
	if itemSchema == nil {
		return nil
	}
	writeFields := map[string]bool{}
	if listInfo != nil && listInfo.item.Post != nil {
		writeFields = extractRequestBodyFields(listInfo.item.Post)
	}
	rs.Fields = buildFieldSpecs(itemSchema, name, writeFields)

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

// operationTimeout reads the x-timeout extension value from an operation's extensions.
// Returns an empty string if the extension is absent or the operation is nil.
func operationTimeout(op *v3high.Operation) string {
	if op == nil || op.Extensions == nil {
		return ""
	}
	node, ok := op.Extensions.Get("x-timeout")
	if !ok || node == nil {
		return ""
	}
	return node.Value
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
