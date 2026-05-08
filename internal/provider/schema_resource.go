package provider

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// buildResourceSchema converts a slice of FieldSpecs to a Terraform schema and a parallel
// attrTypes map used for state encoding/decoding.
func buildResourceSchema(fields []*spec.FieldSpec, mode UntypedFieldMode) (schema.Schema, map[string]attr.Type) {
	attributes := make(map[string]schema.Attribute, len(fields))
	attrTypes := make(map[string]attr.Type, len(fields))
	for _, f := range fields {
		attributes[f.Name] = fieldToResourceSchemaAttr(f, mode)
		attrTypes[f.Name] = fieldToResourceAttrType(f, mode)
	}
	return schema.Schema{Attributes: attributes}, attrTypes
}

// fieldToResourceSchemaAttr converts a FieldSpec to the appropriate Terraform schema attribute,
// applying plan modifiers for immutable and computed fields.
func fieldToResourceSchemaAttr(f *spec.FieldSpec, mode UntypedFieldMode) schema.Attribute {
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
			nestedAttrs[nf.Name] = fieldToResourceSchemaAttr(nf, mode)
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
				nestedAttrs[nf.Name] = fieldToResourceSchemaAttr(nf, mode)
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

// fieldToResourceAttrType returns the attr.Type used for resource state encoding.
// ID fields are coerced to StringType so that terraform import works regardless of the API type.
func fieldToResourceAttrType(f *spec.FieldSpec, mode UntypedFieldMode) attr.Type {
	if f.IsID {
		return types.StringType
	}
	return fieldToDataSourceAttrType(f, mode)
}
