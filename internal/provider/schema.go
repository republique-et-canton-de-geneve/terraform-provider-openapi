package provider

import (
	"math"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// buildDataSourceSchema returns a data source schema with a single computed
// "items" list attribute whose element type mirrors the resource item schema.
func buildDataSourceSchema(fields []*spec.FieldSpec) dsschema.Schema {
	itemAttrs := make(map[string]dsschema.Attribute, len(fields))
	for _, f := range fields {
		itemAttrs[f.Name] = fieldToDSAttr(f)
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

// fieldToDSAttr converts a FieldSpec to a data source schema attribute.
// All attributes are Computed since data sources are read-only.
func fieldToDSAttr(f *spec.FieldSpec) dsschema.Attribute {
	switch f.Type {
	case "integer":
		return dsschema.Int64Attribute{Computed: true}
	case "number":
		return dsschema.Float64Attribute{Computed: true}
	case "boolean":
		return dsschema.BoolAttribute{Computed: true}
	case "object":
		nested := make(map[string]dsschema.Attribute, len(f.Nested))
		for _, nf := range f.Nested {
			nested[nf.Name] = fieldToDSAttr(nf)
		}
		return dsschema.SingleNestedAttribute{Computed: true, Attributes: nested}
	case "array":
		if f.ItemSpec != nil && f.ItemSpec.Type == "object" {
			nested := make(map[string]dsschema.Attribute, len(f.ItemSpec.Nested))
			for _, nf := range f.ItemSpec.Nested {
				nested[nf.Name] = fieldToDSAttr(nf)
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

// buildSchema converts a slice of FieldSpecs to a Terraform schema and a parallel attrTypes map
// used for state encoding/decoding.
func buildSchema(fields []*spec.FieldSpec) (schema.Schema, map[string]attr.Type) {
	attributes := make(map[string]schema.Attribute, len(fields))
	attrTypes := make(map[string]attr.Type, len(fields))
	for _, f := range fields {
		attributes[f.Name] = fieldToSchemaAttr(f)
		attrTypes[f.Name] = fieldToResourceAttrType(f)
	}
	return schema.Schema{Attributes: attributes}, attrTypes
}

// fieldToResourceAttrType returns the attr.Type used for resource state encoding.
// ID fields are coerced to StringType so that terraform import works regardless of the API type.
func fieldToResourceAttrType(f *spec.FieldSpec) attr.Type {
	if f.IsID {
		return types.StringType
	}
	return fieldToDataSourceAttrType(f)
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
	case "object":
		nested := make(map[string]attr.Type, len(f.Nested))
		for _, nf := range f.Nested {
			nested[nf.Name] = fieldToDataSourceAttrType(nf)
		}
		return types.ObjectType{AttrTypes: nested}
	case "array":
		if f.ItemSpec != nil {
			return types.ListType{ElemType: fieldToDataSourceAttrType(f.ItemSpec)}
		}
		return types.ListType{ElemType: types.StringType}
	default:
		return types.StringType
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

// stringValidators builds string validators from a FieldSpec's constraints.
func stringValidators(f *spec.FieldSpec) []validator.String {
	var vals []validator.String
	if f.MinLength != nil {
		vals = append(vals, stringvalidator.LengthAtLeast(int(*f.MinLength)))
	}
	if f.MaxLength != nil {
		vals = append(vals, stringvalidator.LengthAtMost(int(*f.MaxLength)))
	}
	if f.Pattern != "" {
		vals = append(vals, stringvalidator.RegexMatches(regexp.MustCompile(f.Pattern), ""))
	}
	if len(f.Enum) > 0 {
		vals = append(vals, stringvalidator.OneOf(f.Enum...))
	}
	return vals
}

// int64Validators builds int64 validators from a FieldSpec's constraints.
func int64Validators(f *spec.FieldSpec) []validator.Int64 {
	var vals []validator.Int64
	if f.Minimum != nil && *f.Minimum > math.MinInt64 {
		vals = append(vals, int64validator.AtLeast(int64(*f.Minimum)))
	}
	if f.Maximum != nil && *f.Maximum < math.MaxInt64 {
		vals = append(vals, int64validator.AtMost(int64(*f.Maximum)))
	}
	return vals
}

// fieldToSchemaAttr converts a FieldSpec to the appropriate Terraform schema attribute,
// applying plan modifiers for immutable and computed fields.
func fieldToSchemaAttr(f *spec.FieldSpec) schema.Attribute {
	// ID field: always computed string, preserved across plan/apply cycles.
	if f.IsID {
		return schema.StringAttribute{
			MarkdownDescription: f.Description,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseNonNullStateForUnknown(),
			},
		}
	}

	computed := f.Computed || !f.Writable
	optional := f.Writable && (!f.Required || computed)
	required := f.Required && f.Writable && !computed

	switch f.Type {
	case "integer":
		planMods := []planmodifier.Int64{}
		if computed {
			planMods = append(planMods, int64planmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, int64planmodifier.RequiresReplace())
		}
		return schema.Int64Attribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			PlanModifiers:       planMods,
			Validators:          int64Validators(f),
		}
	case "number":
		planMods := []planmodifier.Float64{}
		if computed {
			planMods = append(planMods, float64planmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, float64planmodifier.RequiresReplace())
		}
		return schema.Float64Attribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			PlanModifiers:       planMods,
		}
	case "boolean":
		planMods := []planmodifier.Bool{}
		if computed {
			planMods = append(planMods, boolplanmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, boolplanmodifier.RequiresReplace())
		}
		return schema.BoolAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			PlanModifiers:       planMods,
		}
	case "object":
		nestedAttrs := make(map[string]schema.Attribute, len(f.Nested))
		for _, nf := range f.Nested {
			nestedAttrs[nf.Name] = fieldToSchemaAttr(nf)
		}
		return schema.SingleNestedAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			Attributes:          nestedAttrs,
		}
	case "array":
		if f.ItemSpec != nil && f.ItemSpec.Type == "object" {
			nestedAttrs := make(map[string]schema.Attribute, len(f.ItemSpec.Nested))
			for _, nf := range f.ItemSpec.Nested {
				nestedAttrs[nf.Name] = fieldToSchemaAttr(nf)
			}
			return schema.ListNestedAttribute{
				MarkdownDescription: f.Description,
				Required:            required,
				Optional:            optional,
				Computed:            computed,
				NestedObject: schema.NestedAttributeObject{
					Attributes: nestedAttrs,
				},
			}
		}
		elemType := attr.Type(types.StringType)
		if f.ItemSpec != nil {
			elemType = fieldToResourceAttrType(f.ItemSpec)
		}
		return schema.ListAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			ElementType:         elemType,
		}
	default: // string + fallback
		planMods := []planmodifier.String{}
		if computed {
			planMods = append(planMods, stringplanmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, stringplanmodifier.RequiresReplace())
		}
		return schema.StringAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			Sensitive:           f.Sensitive,
			PlanModifiers:       planMods,
			Validators:          stringValidators(f),
		}
	}
}
