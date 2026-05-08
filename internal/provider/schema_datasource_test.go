package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// --- buildDataSourceSchema -----------------------------------------------------------------------

func TestBuildDataSourceSchema_wraps_in_items(t *testing.T) {
	fields := []*spec.FieldSpec{
		{Name: "id", Type: "string"},
		{Name: "name", Type: "string"},
	}
	s := buildDataSourceSchema(fields)
	items, ok := s.Attributes["items"]
	if !ok {
		t.Fatal("expected items attribute")
	}
	nested, ok := items.(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected ListNestedAttribute, got %T", items)
	}
	if _, ok := nested.NestedObject.Attributes["id"]; !ok {
		t.Fatal("expected id in nested items schema")
	}
}

// --- buildDataSourceAttrTypes --------------------------------------------------------------------

func TestBuildDataSourceAttrTypes_id_is_int64(t *testing.T) {
	fields := []*spec.FieldSpec{
		{Name: "id", Type: "integer", IsID: true},
		{Name: "name", Type: "string"},
	}
	m := buildDataSourceAttrTypes(fields)
	if m["id"] != types.Int64Type {
		t.Fatalf("id: got %v, want Int64Type", m["id"])
	}
	if m["name"] != types.StringType {
		t.Fatalf("name: got %v, want StringType", m["name"])
	}
}

// --- fieldToDataSourceAttr -----------------------------------------------------------------------

func TestFieldToDataSourceAttr_primitives(t *testing.T) {
	cases := []struct {
		typ  string
		want any
	}{
		{"string", dsschema.StringAttribute{}},
		{"integer", dsschema.Int64Attribute{}},
		{"number", dsschema.Float64Attribute{}},
		{"boolean", dsschema.BoolAttribute{}},
	}
	for _, c := range cases {
		t.Run(c.typ, func(t *testing.T) {
			got := fieldToDataSourceAttr(&spec.FieldSpec{Name: "f", Type: c.typ})
			switch c.typ {
			case "string":
				if _, ok := got.(dsschema.StringAttribute); !ok {
					t.Fatalf("got %T", got)
				}
			case "integer":
				if _, ok := got.(dsschema.Int64Attribute); !ok {
					t.Fatalf("got %T", got)
				}
			case "number":
				if _, ok := got.(dsschema.Float64Attribute); !ok {
					t.Fatalf("got %T", got)
				}
			case "boolean":
				if _, ok := got.(dsschema.BoolAttribute); !ok {
					t.Fatalf("got %T", got)
				}
			}
		})
	}
}

func TestFieldToDataSourceAttr_untyped(t *testing.T) {
	got := fieldToDataSourceAttr(&spec.FieldSpec{Name: "payload", Type: "untyped"})
	attr, ok := got.(dsschema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if _, ok := attr.CustomType.(jsontypes.NormalizedType); !ok {
		t.Fatalf("expected NormalizedType CustomType, got %T", attr.CustomType)
	}
}

func TestFieldToDataSourceAttr_sensitive_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "token", Type: "string", Sensitive: true}
	got := fieldToDataSourceAttr(f)
	attr, ok := got.(dsschema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Sensitive {
		t.Fatal("expected Sensitive=true")
	}
}

func TestFieldToDataSourceAttr_object(t *testing.T) {
	f := &spec.FieldSpec{
		Name:   "meta",
		Type:   "object",
		Nested: []*spec.FieldSpec{{Name: "k", Type: "string"}},
	}
	got := fieldToDataSourceAttr(f)
	attr, ok := got.(dsschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected SingleNestedAttribute, got %T", got)
	}
	if _, ok := attr.Attributes["k"]; !ok {
		t.Fatal("expected nested k attribute")
	}
}

func TestFieldToDataSourceAttr_array_of_strings(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "tags",
		Type:     "array",
		ItemSpec: &spec.FieldSpec{Name: "", Type: "string"},
	}
	got := fieldToDataSourceAttr(f)
	attr, ok := got.(dsschema.ListAttribute)
	if !ok {
		t.Fatalf("expected ListAttribute, got %T", got)
	}
	if attr.ElementType != types.StringType {
		t.Fatalf("elem type: got %v", attr.ElementType)
	}
}

func TestFieldToDataSourceAttr_array_no_itemspec(t *testing.T) {
	f := &spec.FieldSpec{Name: "tags", Type: "array"}
	got := fieldToDataSourceAttr(f)
	attr, ok := got.(dsschema.ListAttribute)
	if !ok {
		t.Fatalf("expected ListAttribute, got %T", got)
	}
	if attr.ElementType != types.StringType {
		t.Fatalf("default elem type should be StringType, got %v", attr.ElementType)
	}
}

func TestFieldToDataSourceAttr_array_of_objects(t *testing.T) {
	f := &spec.FieldSpec{
		Name: "items",
		Type: "array",
		ItemSpec: &spec.FieldSpec{
			Name: "",
			Type: "object",
			Nested: []*spec.FieldSpec{
				{Name: "name", Type: "string"},
			},
		},
	}
	got := fieldToDataSourceAttr(f)
	attr, ok := got.(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected ListNestedAttribute, got %T", got)
	}
	if _, ok := attr.NestedObject.Attributes["name"]; !ok {
		t.Fatal("expected nested name attribute")
	}
}

// --- fieldToDataSourceAttrType -------------------------------------------------------------------

func TestFieldToDataSourceAttrType_id_keeps_natural_type(t *testing.T) {
	// Data sources do not support terraform import, so the ID field must keep its API type.
	got := fieldToDataSourceAttrType(&spec.FieldSpec{Name: "id", Type: "integer", IsID: true})
	if got != types.Int64Type {
		t.Fatalf("got %v, want Int64Type", got)
	}
}

func TestFieldToDataSourceAttrType_primitives(t *testing.T) {
	cases := []struct {
		typ  string
		want any
	}{
		{"string", types.StringType},
		{"integer", types.Int64Type},
		{"number", types.Float64Type},
		{"boolean", types.BoolType},
		{"unknown", types.StringType},
	}
	for _, c := range cases {
		t.Run(c.typ, func(t *testing.T) {
			got := fieldToDataSourceAttrType(&spec.FieldSpec{Name: "f", Type: c.typ})
			if got != c.want {
				t.Fatalf("type %q: got %v, want %v", c.typ, got, c.want)
			}
		})
	}
}

func TestFieldToDataSourceAttrType_array_with_integer_id_item(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "items",
		Type:     "array",
		ItemSpec: &spec.FieldSpec{Name: "id", Type: "integer", IsID: true},
	}
	got := fieldToDataSourceAttrType(f)
	listType, ok := got.(types.ListType)
	if !ok {
		t.Fatalf("expected ListType, got %T", got)
	}
	if listType.ElemType != types.Int64Type {
		t.Fatalf("elem type: got %v, want Int64Type", listType.ElemType)
	}
}
