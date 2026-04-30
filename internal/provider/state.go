package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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

// attrMapToJSON converts a Terraform attribute map to a JSON-serialisable map.
func attrMapToJSON(attrs map[string]attr.Value) map[string]any {
	result := make(map[string]any, len(attrs))
	for k, v := range attrs {
		if val := attrToJSON(v); val != nil {
			result[k] = val
		}
	}
	return result
}

// attrToJSON converts a single Terraform attr.Value to its JSON-native equivalent.
func attrToJSON(v attr.Value) any {
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
	case types.Object:
		return attrMapToJSON(t.Attributes())
	case types.List:
		elems := t.Elements()
		result := make([]any, 0, len(elems))
		for _, e := range elems {
			result = append(result, attrToJSON(e))
		}
		return result
	}
	return fmt.Sprint(v)
}

// jsonToObject builds a types.Object from an API JSON response using the known attrTypes map.
func jsonToObject(
	raw map[string]any,
	attrTypes map[string]attr.Type,
) (types.Object, diag.Diagnostics) {
	attrs := make(map[string]attr.Value, len(attrTypes))
	for name, attrType := range attrTypes {
		attrs[name] = jsonToAttr(raw[name], attrType)
	}
	return types.ObjectValue(attrTypes, attrs)
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
