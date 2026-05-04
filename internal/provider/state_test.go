package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// --- extractID -----------------------------------------------------------------------------------

func TestExtractID_string(t *testing.T) {
	obj, _ := types.ObjectValue(
		map[string]attr.Type{"id": types.StringType},
		map[string]attr.Value{"id": types.StringValue("abc-123")},
	)
	if got := extractID(obj, "id"); got != "abc-123" {
		t.Fatalf("got %q, want %q", got, "abc-123")
	}
}

func TestExtractID_int64(t *testing.T) {
	obj, _ := types.ObjectValue(
		map[string]attr.Type{"id": types.Int64Type},
		map[string]attr.Value{"id": types.Int64Value(42)},
	)
	if got := extractID(obj, "id"); got != "42" {
		t.Fatalf("got %q, want %q", got, "42")
	}
}

func TestExtractID_missing_field(t *testing.T) {
	obj, _ := types.ObjectValue(
		map[string]attr.Type{"name": types.StringType},
		map[string]attr.Value{"name": types.StringValue("foo")},
	)
	if got := extractID(obj, "id"); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestExtractID_unsupported_type(t *testing.T) {
	obj, _ := types.ObjectValue(
		map[string]attr.Type{"id": types.BoolType},
		map[string]attr.Value{"id": types.BoolValue(true)},
	)
	if got := extractID(obj, "id"); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

// --- attrToJSON ----------------------------------------------------------------------------------

func TestAttrToJSON_nil(t *testing.T) {
	if got := attrToJSON(nil); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestAttrToJSON_null(t *testing.T) {
	if got := attrToJSON(types.StringNull()); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestAttrToJSON_string(t *testing.T) {
	got := attrToJSON(types.StringValue("hello"))
	if got != "hello" {
		t.Fatalf("got %v, want %q", got, "hello")
	}
}

func TestAttrToJSON_int64(t *testing.T) {
	got := attrToJSON(types.Int64Value(7))
	if got != int64(7) {
		t.Fatalf("got %v, want 7", got)
	}
}

func TestAttrToJSON_float64(t *testing.T) {
	got := attrToJSON(types.Float64Value(3.14))
	if got != 3.14 {
		t.Fatalf("got %v, want 3.14", got)
	}
}

func TestAttrToJSON_bool(t *testing.T) {
	got := attrToJSON(types.BoolValue(true))
	if got != true {
		t.Fatalf("got %v, want true", got)
	}
}

func TestAttrToJSON_object(t *testing.T) {
	obj, _ := types.ObjectValue(
		map[string]attr.Type{"x": types.StringType},
		map[string]attr.Value{"x": types.StringValue("y")},
	)
	got, ok := attrToJSON(obj).(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any")
	}
	if got["x"] != "y" {
		t.Fatalf("got %v, want y", got["x"])
	}
}

func TestAttrToJSON_list(t *testing.T) {
	list, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("a"),
		types.StringValue("b"),
	})
	got, ok := attrToJSON(list).([]any)
	if !ok {
		t.Fatal("expected []any")
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("got %v", got)
	}
}

func TestAttrToJSON_list_null_elem_omitted(t *testing.T) {
	list, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("keep"),
		types.StringNull(),
	})
	result := attrMapToJSON(map[string]attr.Value{"tags": list}, nil)
	tags, ok := result["tags"].([]any)
	if !ok {
		t.Fatal("expected []any")
	}
	// null element converts to nil inside the slice, but the list key is present
	if len(tags) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(tags))
	}
	if tags[0] != "keep" {
		t.Fatalf("got %v, want keep", tags[0])
	}
}

// --- jsonToAttr ----------------------------------------------------------------------------------

func TestJsonToAttr_string_plain(t *testing.T) {
	got := jsonToAttr("hello", types.StringType)
	if got != types.StringValue("hello") {
		t.Fatalf("got %v", got)
	}
}

func TestJsonToAttr_string_nil(t *testing.T) {
	got := jsonToAttr(nil, types.StringType)
	if !got.IsNull() {
		t.Fatal("expected null string")
	}
}

func TestJsonToAttr_string_integer_float(t *testing.T) {
	// JSON numbers arrive as float64; integer floats must render without ".0".
	got := jsonToAttr(float64(42), types.StringType)
	if got != types.StringValue("42") {
		t.Fatalf("got %v, want 42", got)
	}
}

func TestJsonToAttr_string_fractional_float(t *testing.T) {
	got := jsonToAttr(float64(3.5), types.StringType)
	if got != types.StringValue("3.5") {
		t.Fatalf("got %v, want 3.5", got)
	}
}

func TestJsonToAttr_int64(t *testing.T) {
	got := jsonToAttr(float64(99), types.Int64Type)
	if got != types.Int64Value(99) {
		t.Fatalf("got %v", got)
	}
}

func TestJsonToAttr_int64_nil(t *testing.T) {
	got := jsonToAttr(nil, types.Int64Type)
	if !got.IsNull() {
		t.Fatal("expected null int64")
	}
}

func TestJsonToAttr_int64_wrong_type(t *testing.T) {
	got := jsonToAttr("not-a-number", types.Int64Type)
	if !got.IsNull() {
		t.Fatal("expected null int64 for wrong input type")
	}
}

func TestJsonToAttr_float64(t *testing.T) {
	got := jsonToAttr(float64(1.5), types.Float64Type)
	f, ok := got.(types.Float64)
	if !ok || f.ValueFloat64() != 1.5 {
		t.Fatalf("got %v", got)
	}
}

func TestJsonToAttr_float64_nil(t *testing.T) {
	got := jsonToAttr(nil, types.Float64Type)
	if !got.IsNull() {
		t.Fatal("expected null float64")
	}
}

func TestJsonToAttr_bool_true(t *testing.T) {
	got := jsonToAttr(true, types.BoolType)
	if got != types.BoolValue(true) {
		t.Fatalf("got %v", got)
	}
}

func TestJsonToAttr_bool_nil(t *testing.T) {
	got := jsonToAttr(nil, types.BoolType)
	if !got.IsNull() {
		t.Fatal("expected null bool")
	}
}

func TestJsonToAttr_object(t *testing.T) {
	attrTypes := map[string]attr.Type{"name": types.StringType, "count": types.Int64Type}
	objType := types.ObjectType{AttrTypes: attrTypes}
	raw := map[string]any{"name": "foo", "count": float64(3)}
	got := jsonToAttr(raw, objType)
	obj, ok := got.(types.Object)
	if !ok || obj.IsNull() {
		t.Fatal("expected non-null object")
	}
	if obj.Attributes()["name"] != types.StringValue("foo") {
		t.Fatalf("name: got %v", obj.Attributes()["name"])
	}
	if obj.Attributes()["count"] != types.Int64Value(3) {
		t.Fatalf("count: got %v", obj.Attributes()["count"])
	}
}

func TestJsonToAttr_object_nil(t *testing.T) {
	attrTypes := map[string]attr.Type{"x": types.StringType}
	objType := types.ObjectType{AttrTypes: attrTypes}
	got := jsonToAttr(nil, objType)
	if !got.IsNull() {
		t.Fatal("expected null object")
	}
}

func TestJsonToAttr_object_wrong_type(t *testing.T) {
	attrTypes := map[string]attr.Type{"x": types.StringType}
	objType := types.ObjectType{AttrTypes: attrTypes}
	got := jsonToAttr("not-an-object", objType)
	if !got.IsNull() {
		t.Fatal("expected null object for wrong input type")
	}
}

func TestJsonToAttr_list_strings(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	got := jsonToAttr([]any{"x", "y"}, listType)
	list, ok := got.(types.List)
	if !ok || list.IsNull() {
		t.Fatal("expected non-null list")
	}
	elems := list.Elements()
	if len(elems) != 2 || elems[0] != types.StringValue("x") {
		t.Fatalf("got %v", elems)
	}
}

func TestJsonToAttr_list_nil(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	got := jsonToAttr(nil, listType)
	list, ok := got.(types.List)
	if !ok || list.IsNull() {
		t.Fatal("expected empty (not null) list for nil input")
	}
	if len(list.Elements()) != 0 {
		t.Fatalf("expected empty list, got %v", list.Elements())
	}
}

func TestJsonToAttr_list_wrong_type(t *testing.T) {
	listType := types.ListType{ElemType: types.StringType}
	got := jsonToAttr("not-a-list", listType)
	list, ok := got.(types.List)
	if !ok || list.IsNull() {
		t.Fatal("expected empty list for wrong input type")
	}
}

// --- jsonToObject --------------------------------------------------------------------------------

func TestJsonToObject_roundtrip(t *testing.T) {
	attrTypes := map[string]attr.Type{
		"id":      types.StringType,
		"count":   types.Int64Type,
		"enabled": types.BoolType,
	}
	raw := map[string]any{
		"id":      "xyz",
		"count":   float64(5),
		"enabled": true,
	}
	obj, diags := jsonToObject(raw, nil, attrTypes)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	attrs := obj.Attributes()
	if attrs["id"] != types.StringValue("xyz") {
		t.Fatalf("id: got %v", attrs["id"])
	}
	if attrs["count"] != types.Int64Value(5) {
		t.Fatalf("count: got %v", attrs["count"])
	}
	if attrs["enabled"] != types.BoolValue(true) {
		t.Fatalf("enabled: got %v", attrs["enabled"])
	}
}

func TestJsonToObject_missing_keys_become_null(t *testing.T) {
	attrTypes := map[string]attr.Type{"name": types.StringType}
	obj, diags := jsonToObject(map[string]any{}, nil, attrTypes)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !obj.Attributes()["name"].IsNull() {
		t.Fatal("expected null for missing key")
	}
}

func TestJsonToObject_camelCase_name_mapping(t *testing.T) {
	// API returns camelCase; Terraform state must use snake_case.
	fields := []*spec.FieldSpec{
		{Name: "id", OASName: "id", IsID: true},
		{Name: "photo_urls", OASName: "photoUrls", Type: "array",
			ItemSpec: &spec.FieldSpec{Name: "item", OASName: "item", Type: "string"}},
		{Name: "pet_id", OASName: "petId", Type: "integer"},
	}
	attrTypes := map[string]attr.Type{
		"id":         types.StringType,
		"photo_urls": types.ListType{ElemType: types.StringType},
		"pet_id":     types.Int64Type,
	}
	raw := map[string]any{
		"id":        "abc",
		"photoUrls": []any{"https://example.com/a.jpg"},
		"petId":     float64(7),
	}
	obj, diags := jsonToObject(raw, fields, attrTypes)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	attrs := obj.Attributes()
	if attrs["id"] != types.StringValue("abc") {
		t.Fatalf("id: got %v", attrs["id"])
	}
	if attrs["pet_id"] != types.Int64Value(7) {
		t.Fatalf("pet_id: got %v", attrs["pet_id"])
	}
	list, ok := attrs["photo_urls"].(types.List)
	if !ok || list.IsNull() || len(list.Elements()) != 1 {
		t.Fatalf("photo_urls: got %v", attrs["photo_urls"])
	}
}

func TestAttrMapToJSON_camelCase_name_mapping(t *testing.T) {
	// Terraform plan uses snake_case; JSON body sent to API must use camelCase.
	fields := []*spec.FieldSpec{
		{Name: "photo_urls", OASName: "photoUrls", Type: "array",
			ItemSpec: &spec.FieldSpec{Name: "item", OASName: "item", Type: "string"}},
		{Name: "pet_id", OASName: "petId", Type: "integer"},
	}
	list, _ := types.ListValue(types.StringType, []attr.Value{types.StringValue("https://example.com/a.jpg")})
	attrs := map[string]attr.Value{
		"photo_urls": list,
		"pet_id":     types.Int64Value(7),
	}
	result := attrMapToJSON(attrs, fields)
	if _, ok := result["photo_urls"]; ok {
		t.Error("snake_case key 'photo_urls' must not appear in JSON output")
	}
	if _, ok := result["pet_id"]; ok {
		t.Error("snake_case key 'pet_id' must not appear in JSON output")
	}
	if result["photoUrls"] == nil {
		t.Error("camelCase key 'photoUrls' must appear in JSON output")
	}
	if result["petId"] != int64(7) {
		t.Fatalf("petId: got %v", result["petId"])
	}
}
