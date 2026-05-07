package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// extractID reads the ID field from a state object, returning it as a string.
func extractID(obj types.Object, idField string) string {
	attrs := obj.Attributes()
	v, ok := attrs[idField]
	if !ok {
		return ""
	}
	switch id := v.(type) {
	case types.String:
		return id.ValueString()
	case types.Int64:
		return fmt.Sprintf("%d", id.ValueInt64())
	}
	return ""
}

// attrMapToJSON converts a Terraform attribute map (snake_case keys) to a JSON-serialisable map
// using OASName for API keys. Pass nil fields to use attribute names as-is (no name translation).
func attrMapToJSON(attrs map[string]attr.Value, fields []*spec.FieldSpec) map[string]any {
	byName := make(map[string]*spec.FieldSpec, len(fields))
	for _, f := range fields {
		byName[f.Name] = f
	}
	result := make(map[string]any, len(attrs))
	for k, v := range attrs {
		f := byName[k]
		oasKey := k
		if f != nil {
			oasKey = f.OASName
		}
		if val := attrToJSONField(v, f); val != nil {
			result[oasKey] = val
		}
	}
	return result
}

// attrToJSON converts a single Terraform attr.Value to its JSON-native equivalent without any
// field-name translation. Kept for backward compat with simple cases.
func attrToJSON(v attr.Value) any {
	return attrToJSONField(v, nil)
}

// attrToJSONField is like attrToJSON but threads field spec for nested name mapping.
func attrToJSONField(v attr.Value, f *spec.FieldSpec) any {
	if v == nil || v.IsNull() || v.IsUnknown() {
		return nil
	}
	switch t := v.(type) {
	case types.String:
		return t.ValueString()
	case types.Int64:
		return t.ValueInt64()
	case types.Float64:
		return t.ValueFloat64()
	case types.Bool:
		return t.ValueBool()
	case types.Dynamic:
		if t.IsNull() || t.IsUnknown() {
			return nil
		}
		return attrToJSONField(t.UnderlyingValue(), nil)
	case types.Object:
		var nested []*spec.FieldSpec
		if f != nil {
			nested = f.Nested
		}
		return attrMapToJSON(t.Attributes(), nested)
	case types.List:
		elems := t.Elements()
		result := make([]any, 0, len(elems))
		var itemSpec *spec.FieldSpec
		if f != nil {
			itemSpec = f.ItemSpec
		}
		for _, e := range elems {
			result = append(result, attrToJSONField(e, itemSpec))
		}
		return result
	}
	return fmt.Sprint(v)
}

// jsonToObject builds a types.Object from an API JSON response.
// fields drives the OASName→Name translation; pass nil to use attribute names as-is.
func jsonToObject(
	raw map[string]any,
	fields []*spec.FieldSpec,
	attrTypes map[string]attr.Type,
) (types.Object, diag.Diagnostics) {
	attrs := make(map[string]attr.Value, len(attrTypes))
	if len(fields) > 0 {
		for _, f := range fields {
			attrs[f.Name] = jsonToAttrField(raw[f.OASName], attrTypes[f.Name], f)
		}
	} else {
		for name, attrType := range attrTypes {
			attrs[name] = jsonToAttr(raw[name], attrType)
		}
	}
	return types.ObjectValue(attrTypes, attrs)
}

// jsonToAttrField is like jsonToAttr but uses field spec for nested name mapping.
func jsonToAttrField(v any, t attr.Type, f *spec.FieldSpec) attr.Value {
	switch at := t.(type) {
	case basetypes.DynamicType:
		return jsonToDynamic(v)
	case basetypes.ObjectType:
		if v == nil {
			return types.ObjectNull(at.AttrTypes)
		}
		if m, ok := v.(map[string]any); ok {
			nested := make(map[string]attr.Value, len(at.AttrTypes))
			if f != nil && len(f.Nested) > 0 {
				for _, nf := range f.Nested {
					nested[nf.Name] = jsonToAttrField(m[nf.OASName], at.AttrTypes[nf.Name], nf)
				}
			} else {
				for name, nestedType := range at.AttrTypes {
					nested[name] = jsonToAttr(m[name], nestedType)
				}
			}
			obj, _ := types.ObjectValue(at.AttrTypes, nested)
			return obj
		}
		return types.ObjectNull(at.AttrTypes)
	case basetypes.ListType:
		if v == nil {
			list, _ := types.ListValue(at.ElemType, []attr.Value{})
			return list
		}
		if arr, ok := v.([]any); ok {
			elems := make([]attr.Value, len(arr))
			var itemSpec *spec.FieldSpec
			if f != nil {
				itemSpec = f.ItemSpec
			}
			for i, item := range arr {
				elems[i] = jsonToAttrField(item, at.ElemType, itemSpec)
			}
			list, _ := types.ListValue(at.ElemType, elems)
			return list
		}
		list, _ := types.ListValue(at.ElemType, []attr.Value{})
		return list
	default:
		return jsonToAttr(v, t)
	}
}

// jsonToAttr converts a JSON-decoded value to the Terraform attr.Value matching type t.
func jsonToAttr(v any, t attr.Type) attr.Value {
	switch at := t.(type) {
	case basetypes.StringType:
		if v == nil {
			return types.StringNull()
		}
		// Convert integer floats to clean integer strings ("42" not "42.0").
		if n, ok := v.(float64); ok {
			if n == float64(int64(n)) {
				return types.StringValue(fmt.Sprintf("%d", int64(n)))
			}
			return types.StringValue(fmt.Sprintf("%g", n))
		}
		return types.StringValue(fmt.Sprint(v))
	case basetypes.Int64Type:
		if v == nil {
			return types.Int64Null()
		}
		if n, ok := v.(float64); ok {
			return types.Int64Value(int64(n))
		}
		return types.Int64Null()
	case basetypes.Float64Type:
		if v == nil {
			return types.Float64Null()
		}
		if n, ok := v.(float64); ok {
			return types.Float64Value(n)
		}
		return types.Float64Null()
	case basetypes.BoolType:
		if v == nil {
			return types.BoolNull()
		}
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
		return types.BoolNull()
	case basetypes.ObjectType:
		if v == nil {
			return types.ObjectNull(at.AttrTypes)
		}
		if m, ok := v.(map[string]any); ok {
			nested := make(map[string]attr.Value, len(at.AttrTypes))
			for name, nestedType := range at.AttrTypes {
				nested[name] = jsonToAttr(m[name], nestedType)
			}
			obj, _ := types.ObjectValue(at.AttrTypes, nested)
			return obj
		}
		return types.ObjectNull(at.AttrTypes)
	case basetypes.ListType:
		if v == nil {
			list, _ := types.ListValue(at.ElemType, []attr.Value{})
			return list
		}
		if arr, ok := v.([]any); ok {
			elems := make([]attr.Value, len(arr))
			for i, item := range arr {
				elems[i] = jsonToAttr(item, at.ElemType)
			}
			list, _ := types.ListValue(at.ElemType, elems)
			return list
		}
		list, _ := types.ListValue(at.ElemType, []attr.Value{})
		return list
	}
	// Fallback: string representation
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(fmt.Sprint(v))
}

// jsonToDynamic converts any JSON value to a types.Dynamic Terraform value.
func jsonToDynamic(v any) types.Dynamic {
	if v == nil {
		return types.DynamicNull()
	}
	return types.DynamicValue(jsonToUntypedValue(v))
}

// jsonToUntypedValue converts a JSON-decoded value to the best-fit Terraform attr.Value
// without schema context. Used for dynamic attributes where no type is declared in OAS.
func jsonToUntypedValue(v any) attr.Value {
	if v == nil {
		return types.StringNull()
	}
	switch t := v.(type) {
	case string:
		return types.StringValue(t)
	case float64:
		if t == float64(int64(t)) {
			return types.Int64Value(int64(t))
		}
		return types.Float64Value(t)
	case bool:
		return types.BoolValue(t)
	case []any:
		if len(t) == 0 {
			list, _ := types.ListValue(types.StringType, []attr.Value{})
			return list
		}
		elems := make([]attr.Value, len(t))
		for i, item := range t {
			elems[i] = jsonToUntypedValue(item)
		}
		elemType := elems[0].Type(context.Background())
		list, err := types.ListValue(elemType, elems)
		if err != nil {
			// Mixed element types: fall back to string list.
			strElems := make([]attr.Value, len(t))
			for i, item := range t {
				strElems[i] = types.StringValue(fmt.Sprint(item))
			}
			list, _ = types.ListValue(types.StringType, strElems)
		}
		return list
	case map[string]any:
		attrTypes := make(map[string]attr.Type, len(t))
		attrs := make(map[string]attr.Value, len(t))
		for k, val := range t {
			av := jsonToUntypedValue(val)
			attrTypes[k] = av.Type(context.Background())
			attrs[k] = av
		}
		obj, _ := types.ObjectValue(attrTypes, attrs)
		return obj
	}
	return types.StringValue(fmt.Sprint(v))
}
