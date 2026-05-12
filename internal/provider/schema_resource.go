package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// sortListModifier sorts list elements at plan time so that config order never differs
// from the sorted state written by Read, preventing spurious diffs on x-unordered fields.
type sortListModifier struct{}

func (sortListModifier) Description(_ context.Context) string         { return "sorts list elements" }
func (sortListModifier) MarkdownDescription(_ context.Context) string { return "sorts list elements" }

func (sortListModifier) PlanModifyList(_ context.Context,
	req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	elems := req.PlanValue.Elements()
	sort.Slice(elems, func(i, j int) bool {
		return fmt.Sprint(elems[i]) < fmt.Sprint(elems[j])
	})
	sorted, diags := types.ListValue(req.PlanValue.ElementType(context.Background()), elems)
	resp.Diagnostics.Append(diags...)
	resp.PlanValue = sorted
}

// timeoutsAttrTypes is the fixed type map for the timeouts block attributes.
var timeoutsAttrTypes = map[string]attr.Type{
	"create": types.StringType,
	"read":   types.StringType,
	"update": types.StringType,
	"delete": types.StringType,
}

// buildResourceSchema converts a slice of FieldSpecs to a Terraform schema, a parallel attrTypes
// map used for resource-field state encoding, and the timeouts block object type.
func buildResourceSchema(
	fields []*spec.FieldSpec,
	timeouts spec.ResourceTimeouts,
) (schema.Schema, map[string]attr.Type, types.ObjectType) {
	attributes := make(map[string]schema.Attribute, len(fields))
	attrTypes := make(map[string]attr.Type, len(fields))
	for _, f := range fields {
		attributes[f.Name] = fieldToResourceSchemaAttr(f)
		attrTypes[f.Name] = fieldToResourceAttrType(f)
	}
	timeoutsBlock := buildTimeoutsBlock(timeouts)
	timeoutsType := types.ObjectType{AttrTypes: timeoutsAttrTypes}
	return schema.Schema{
		Attributes: attributes,
		Blocks:     map[string]schema.Block{"timeouts": timeoutsBlock},
	}, attrTypes, timeoutsType
}

// buildTimeoutsBlock returns a SingleNestedBlock for per-operation timeouts.
// x-timeout values from the spec are surfaced in the attribute descriptions.
func buildTimeoutsBlock(t spec.ResourceTimeouts) schema.Block {
	timeoutAttr := func(op, specDefault string) schema.Attribute {
		desc := "Timeout for the " + op + " operation (e.g. \"30m\", \"1h\")."
		if specDefault != "" {
			desc += " API spec default: " + specDefault + "."
		} else {
			desc += " Provider default: 20m."
		}
		return schema.StringAttribute{
			MarkdownDescription: desc,
			Optional:            true,
			Validators:          []validator.String{positiveDuration{}},
		}
	}
	return schema.SingleNestedBlock{
		MarkdownDescription: "Configures per-operation timeouts. " +
			"Overrides the `x-timeout` values from the API spec.",
		Attributes: map[string]schema.Attribute{
			"create": timeoutAttr("create", t.Create),
			"read":   timeoutAttr("read", t.Read),
			"update": timeoutAttr("update", t.Update),
			"delete": timeoutAttr("delete", t.Delete),
		},
	}
}

// fieldToResourceSchemaAttr converts a FieldSpec to the appropriate Terraform schema attribute,
// applying plan modifiers for immutable and computed fields.
func fieldToResourceSchemaAttr(f *spec.FieldSpec) schema.Attribute {
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
		// Int64; supports min/max validators and immutability.
		planMods := []planmodifier.Int64{}
		if f.Immutable {
			planMods = append(planMods, int64planmodifier.UseNonNullStateForUnknown())
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
		// Float64; no validators beyond immutability.
		planMods := []planmodifier.Float64{}
		if f.Immutable {
			planMods = append(planMods, float64planmodifier.UseNonNullStateForUnknown())
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
		// Bool; no validators beyond immutability.
		planMods := []planmodifier.Bool{}
		if f.Immutable {
			planMods = append(planMods, boolplanmodifier.UseNonNullStateForUnknown())
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
		// No declared OAS type: stored as jsontypes.Normalized (JSON string).
		planMods := []planmodifier.String{}
		if f.Immutable {
			planMods = append(planMods, stringplanmodifier.UseNonNullStateForUnknown())
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
		// Single nested object; child fields recurse into this function.
		nestedAttrs := make(map[string]schema.Attribute, len(f.Nested))
		for _, nf := range f.Nested {
			nestedAttrs[nf.Name] = fieldToResourceSchemaAttr(nf)
		}
		return schema.SingleNestedAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			Attributes:          nestedAttrs,
		}
	case "array":
		// Four cases driven by x-unordered and uniqueItems:
		//   x-unordered + uniqueItems → Set (unordered, enforces uniqueness)
		//   x-unordered only          → sorted List (sort on read + plan modifier)
		//   uniqueItems only          → List + uniqueness validator
		//   neither                   → plain List
		unordered := f.Unordered
		unique := f.UniqueItems

		if unordered && unique {
			// Set: framework enforces uniqueness and order is irrelevant.
			if f.ItemSpec != nil && f.ItemSpec.Type == "object" {
				nestedAttrs := make(map[string]schema.Attribute, len(f.ItemSpec.Nested))
				for _, nf := range f.ItemSpec.Nested {
					nestedAttrs[nf.Name] = fieldToResourceSchemaAttr(nf)
				}
				return schema.SetNestedAttribute{
					MarkdownDescription: f.Description,
					Required:            required,
					Optional:            optional,
					Computed:            computed,
					NestedObject:        schema.NestedAttributeObject{Attributes: nestedAttrs},
				}
			}
			elemType := attr.Type(types.StringType)
			if f.ItemSpec != nil {
				elemType = fieldToResourceAttrType(f.ItemSpec)
			}
			a := schema.SetAttribute{
				MarkdownDescription: f.Description,
				Required:            required,
				Optional:            optional,
				Computed:            computed,
				ElementType:         elemType,
			}
			if hasDefault {
				if _, ok := f.Default.([]any); ok {
					a.Default = setdefault.StaticValue(types.SetValueMust(elemType, []attr.Value{}))
				}
			}
			return a
		}

		// All remaining cases use List; build nested attrs or elem type first.
		var listValidators []validator.List
		var listPlanMods []planmodifier.List
		if unordered {
			listPlanMods = append(listPlanMods, sortListModifier{})
		}
		if unique {
			listValidators = append(listValidators, uniqueListValidator{})
		}

		if f.ItemSpec != nil && f.ItemSpec.Type == "object" {
			nestedAttrs := make(map[string]schema.Attribute, len(f.ItemSpec.Nested))
			for _, nf := range f.ItemSpec.Nested {
				nestedAttrs[nf.Name] = fieldToResourceSchemaAttr(nf)
			}
			return schema.ListNestedAttribute{
				MarkdownDescription: f.Description,
				Required:            required,
				Optional:            optional,
				Computed:            computed,
				PlanModifiers:       listPlanMods,
				Validators:          listValidators,
				NestedObject:        schema.NestedAttributeObject{Attributes: nestedAttrs},
			}
		}
		elemType := attr.Type(types.StringType)
		if f.ItemSpec != nil {
			elemType = fieldToResourceAttrType(f.ItemSpec)
		}
		a := schema.ListAttribute{
			MarkdownDescription: f.Description,
			Required:            required,
			Optional:            optional,
			Computed:            computed,
			ElementType:         elemType,
			PlanModifiers:       listPlanMods,
			Validators:          listValidators,
		}
		if hasDefault {
			if _, ok := f.Default.([]any); ok {
				a.Default = listdefault.StaticValue(types.ListValueMust(elemType, []attr.Value{}))
			}
		}
		return a
	default:
		// String (and unrecognised types fall back to string).
		planMods := []planmodifier.String{}
		if f.Immutable {
			planMods = append(planMods, stringplanmodifier.UseNonNullStateForUnknown())
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
func fieldToResourceAttrType(f *spec.FieldSpec) attr.Type {
	if f.IsID {
		return types.StringType
	}
	return fieldToDataSourceAttrType(f)
}
