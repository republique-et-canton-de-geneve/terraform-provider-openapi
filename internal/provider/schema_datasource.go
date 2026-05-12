package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// buildDataSourceSchema returns a data source schema with a single computed
// "items" list attribute whose element type mirrors the resource item schema.
func buildDataSourceSchema(fields []*spec.FieldSpec) dsschema.Schema {
	itemAttrs := make(map[string]dsschema.Attribute, len(fields))
	for _, f := range fields {
		itemAttrs[f.Name] = fieldToDataSourceAttr(f)
	}
	return dsschema.Schema{
		Attributes: map[string]dsschema.Attribute{
			"items": dsschema.ListNestedAttribute{
				Computed:     true,
				NestedObject: dsschema.NestedAttributeObject{Attributes: itemAttrs},
			},
		},
	}
}

// buildDataSourceAttrTypes builds the attrTypes map for a data source using
// fieldToDataSourceAttrType so that ID fields keep their natural API type.
func buildDataSourceAttrTypes(fields []*spec.FieldSpec) map[string]attr.Type {
	m := make(map[string]attr.Type, len(fields))
	for _, f := range fields {
		m[f.Name] = fieldToDataSourceAttrType(f)
	}
	return m
}

// fieldToDataSourceAttr converts a FieldSpec to a data source schema attribute.
// All attributes are Computed since data sources are read-only.
func fieldToDataSourceAttr(f *spec.FieldSpec) dsschema.Attribute {
	switch f.Type {
	case "integer":
		return dsschema.Int64Attribute{Computed: true}
	case "number":
		return dsschema.Float64Attribute{Computed: true}
	case "boolean":
		return dsschema.BoolAttribute{Computed: true}
	case "untyped":
		return dsschema.StringAttribute{CustomType: jsontypes.NormalizedType{}, Computed: true}
	case "object":
		nested := make(map[string]dsschema.Attribute, len(f.Nested))
		for _, nf := range f.Nested {
			nested[nf.Name] = fieldToDataSourceAttr(nf)
		}
		return dsschema.SingleNestedAttribute{Computed: true, Attributes: nested}
	case "array":
		if f.Unordered && f.UniqueItems {
			// Set: unordered + unique.
			if f.ItemSpec != nil && f.ItemSpec.Type == "object" {
				nested := make(map[string]dsschema.Attribute, len(f.ItemSpec.Nested))
				for _, nf := range f.ItemSpec.Nested {
					nested[nf.Name] = fieldToDataSourceAttr(nf)
				}
				return dsschema.SetNestedAttribute{
					Computed:     true,
					NestedObject: dsschema.NestedAttributeObject{Attributes: nested},
				}
			}
			elemType := attr.Type(types.StringType)
			if f.ItemSpec != nil {
				elemType = fieldToDataSourceAttrType(f.ItemSpec)
			}
			return dsschema.SetAttribute{Computed: true, ElementType: elemType}
		}
		// All other cases: List (sorted on read when x-unordered; validated when uniqueItems).
		if f.ItemSpec != nil && f.ItemSpec.Type == "object" {
			nested := make(map[string]dsschema.Attribute, len(f.ItemSpec.Nested))
			for _, nf := range f.ItemSpec.Nested {
				nested[nf.Name] = fieldToDataSourceAttr(nf)
			}
			return dsschema.ListNestedAttribute{
				Computed:     true,
				NestedObject: dsschema.NestedAttributeObject{Attributes: nested},
			}
		}
		elemType := attr.Type(types.StringType)
		if f.ItemSpec != nil {
			elemType = fieldToDataSourceAttrType(f.ItemSpec)
		}
		return dsschema.ListAttribute{Computed: true, ElementType: elemType}
	default:
		return dsschema.StringAttribute{Computed: true, Sensitive: f.Sensitive}
	}
}

// fieldToDataSourceAttrType returns the attr.Type used for data source state encoding.
// Unlike fieldToResourceAttrType it does not override ID fields, because data sources do not
// support terraform import and must match the API's actual type.
func fieldToDataSourceAttrType(f *spec.FieldSpec) attr.Type {
	switch f.Type {
	case "integer":
		return types.Int64Type
	case "number":
		return types.Float64Type
	case "boolean":
		return types.BoolType
	case "untyped":
		return jsontypes.NormalizedType{}
	case "object":
		nested := make(map[string]attr.Type, len(f.Nested))
		for _, nf := range f.Nested {
			nested[nf.Name] = fieldToDataSourceAttrType(nf)
		}
		return types.ObjectType{AttrTypes: nested}
	case "array":
		if f.Unordered && f.UniqueItems {
			if f.ItemSpec != nil {
				return types.SetType{ElemType: fieldToDataSourceAttrType(f.ItemSpec)}
			}
			return types.SetType{ElemType: types.StringType}
		}
		if f.ItemSpec != nil {
			return types.ListType{ElemType: fieldToDataSourceAttrType(f.ItemSpec)}
		}
		return types.ListType{ElemType: types.StringType}
	default:
		return types.StringType
	}
}
