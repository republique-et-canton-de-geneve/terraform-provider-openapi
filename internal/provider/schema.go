package provider

import (
	"encoding/json"
	"math"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// UntypedFieldMode controls how OAS fields with no declared type are exposed in the schema.
type UntypedFieldMode string

const (
	UntypedFieldModeJSON  UntypedFieldMode = "json"
	UntypedFieldModeError UntypedFieldMode = "error"
)

// buildDataSourceSchema returns a data source schema with a single computed
// "items" list attribute whose element type mirrors the resource item schema.
func buildDataSourceSchema(fields []*spec.FieldSpec, mode UntypedFieldMode) dsschema.Schema {
	itemAttrs := make(map[string]dsschema.Attribute, len(fields))
	for _, f := range fields {
		itemAttrs[f.Name] = fieldToDataSourceAttr(f, mode)
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

// fieldToDataSourceAttr converts a FieldSpec to a data source schema attribute.
// All attributes are Computed since data sources are read-only.
func fieldToDataSourceAttr(f *spec.FieldSpec, mode UntypedFieldMode) dsschema.Attribute {
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
			nested[nf.Name] = fieldToDataSourceAttr(nf, mode)
		}
		return dsschema.SingleNestedAttribute{Computed: true, Attributes: nested}
	case "array":
		if f.ItemSpec != nil && f.ItemSpec.Type == "object" {
			nested := make(map[string]dsschema.Attribute, len(f.ItemSpec.Nested))
			for _, nf := range f.ItemSpec.Nested {
				nested[nf.Name] = fieldToDataSourceAttr(nf, mode)
			}
			return dsschema.ListNestedAttribute{
				Computed:     true,
				NestedObject: dsschema.NestedAttributeObject{Attributes: nested},
			}
		}
		elemType := attr.Type(types.StringType)
		if f.ItemSpec != nil {
			elemType = fieldToDataSourceAttrType(f.ItemSpec, mode)
		}
		return dsschema.ListAttribute{Computed: true, ElementType: elemType}
	default:
		return dsschema.StringAttribute{Computed: true, Sensitive: f.Sensitive}
	}
}

// buildSchema converts a slice of FieldSpecs to a Terraform schema and a parallel attrTypes map
// used for state encoding/decoding.
func buildSchema(fields []*spec.FieldSpec, mode UntypedFieldMode) (schema.Schema, map[string]attr.Type) {
	attributes := make(map[string]schema.Attribute, len(fields))
	attrTypes := make(map[string]attr.Type, len(fields))
	for _, f := range fields {
		attributes[f.Name] = fieldToSchemaAttr(f, mode)
		attrTypes[f.Name] = fieldToResourceAttrType(f, mode)
	}
	return schema.Schema{Attributes: attributes}, attrTypes
}

// fieldToResourceAttrType returns the attr.Type used for resource state encoding.
// ID fields are coerced to StringType so that terraform import works regardless of the API type.
func fieldToResourceAttrType(f *spec.FieldSpec, mode UntypedFieldMode) attr.Type {
	if f.IsID {
		return types.StringType
	}
	return fieldToDataSourceAttrType(f, mode)
}

// fieldToDataSourceAttrType returns the attr.Type used for data source state encoding.
// Unlike fieldToResourceAttrType it does not override ID fields, because data sources do not
// support terraform import and must match the API's actual type.
func fieldToDataSourceAttrType(f *spec.FieldSpec, mode UntypedFieldMode) attr.Type {
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
			nested[nf.Name] = fieldToDataSourceAttrType(nf, mode)
		}
		return types.ObjectType{AttrTypes: nested}
	case "array":
		if f.ItemSpec != nil {
			return types.ListType{ElemType: fieldToDataSourceAttrType(f.ItemSpec, mode)}
		}
		return types.ListType{ElemType: types.StringType}
	default:
		return types.StringType
	}
}

// buildDataSourceAttrTypes builds the attrTypes map for a data source using
// fieldToDataSourceAttrType so that ID fields keep their natural API type.
func buildDataSourceAttrTypes(fields []*spec.FieldSpec, mode UntypedFieldMode) map[string]attr.Type {
	m := make(map[string]attr.Type, len(fields))
	for _, f := range fields {
		m[f.Name] = fieldToDataSourceAttrType(f, mode)
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
func fieldToSchemaAttr(f *spec.FieldSpec, mode UntypedFieldMode) schema.Attribute {
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

	// A writable field with a default implies Computed so the framework accepts the Default.
	hasDefault := f.Default != nil && f.Writable
	computed := f.Computed || !f.Writable || hasDefault
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
		a := schema.Int64Attribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			PlanModifiers:       planMods,
			Validators:          int64Validators(f),
		}
		if hasDefault {
			if v, ok := f.Default.(int64); ok {
				a.Default = int64default.StaticInt64(v)
			}
		}
		return a
	case "number":
		planMods := []planmodifier.Float64{}
		if computed {
			planMods = append(planMods, float64planmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, float64planmodifier.RequiresReplace())
		}
		a := schema.Float64Attribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			PlanModifiers:       planMods,
		}
		if hasDefault {
			if v, ok := f.Default.(float64); ok {
				a.Default = float64default.StaticFloat64(v)
			}
		}
		return a
	case "boolean":
		planMods := []planmodifier.Bool{}
		if computed {
			planMods = append(planMods, boolplanmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, boolplanmodifier.RequiresReplace())
		}
		a := schema.BoolAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			PlanModifiers:       planMods,
		}
		if hasDefault {
			if v, ok := f.Default.(bool); ok {
				a.Default = booldefault.StaticBool(v)
			}
		}
		return a
	case "untyped":
		planMods := []planmodifier.String{}
		if computed {
			planMods = append(planMods, stringplanmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, stringplanmodifier.RequiresReplace())
		}
		a := schema.StringAttribute{
			CustomType:          jsontypes.NormalizedType{},
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			PlanModifiers:       planMods,
		}
		if hasDefault {
			if b, err := json.Marshal(f.Default); err == nil {
				a.Default = stringdefault.StaticString(string(b))
			}
		}
		return a
	case "object":
		nestedAttrs := make(map[string]schema.Attribute, len(f.Nested))
		for _, nf := range f.Nested {
			nestedAttrs[nf.Name] = fieldToSchemaAttr(nf, mode)
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
				nestedAttrs[nf.Name] = fieldToSchemaAttr(nf, mode)
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
			elemType = fieldToResourceAttrType(f.ItemSpec, mode)
		}
		a := schema.ListAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			ElementType:         elemType,
		}
		if hasDefault {
			if _, ok := f.Default.([]any); ok {
				a.Default = listdefault.StaticValue(types.ListValueMust(elemType, []attr.Value{}))
			}
		}
		return a
	default: // string + fallback
		planMods := []planmodifier.String{}
		if computed {
			planMods = append(planMods, stringplanmodifier.UseNonNullStateForUnknown())
		}
		if f.Immutable {
			planMods = append(planMods, stringplanmodifier.RequiresReplace())
		}
		a := schema.StringAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			Sensitive:           f.Sensitive,
			PlanModifiers:       planMods,
			Validators:          stringValidators(f),
		}
		if hasDefault {
			if v, ok := f.Default.(string); ok {
				a.Default = stringdefault.StaticString(v)
			}
		}
		return a
	}
}
