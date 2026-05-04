package provider

import (
	"testing"

	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// --- fieldToAttrType -----------------------------------------------------------------------------

func TestFieldToAttrType_primitives(t *testing.T) {
	cases := []struct {
		typ  string
		want any
	}{
		{"string", types.StringType},
		{"integer", types.Int64Type},
		{"number", types.Float64Type},
		{"boolean", types.BoolType},
		{"unknown", types.StringType}, // fallback
	}
	for _, c := range cases {
		t.Run(c.typ, func(t *testing.T) {
			got := fieldToAttrType(&spec.FieldSpec{Name: "f", Type: c.typ})
			if got != c.want {
				t.Fatalf("type %q: got %v, want %v", c.typ, got, c.want)
			}
		})
	}
}

func TestFieldToAttrType_id_always_string(t *testing.T) {
	// Even an integer ID field must map to StringType for terraform import.
	got := fieldToAttrType(&spec.FieldSpec{Name: "id", Type: "integer", IsID: true})
	if got != types.StringType {
		t.Fatalf("got %v, want StringType", got)
	}
}

func TestFieldToAttrType_object(t *testing.T) {
	f := &spec.FieldSpec{
		Name:   "meta",
		Type:   "object",
		Nested: []*spec.FieldSpec{{Name: "key", Type: "string"}},
	}
	got := fieldToAttrType(f)
	objType, ok := got.(types.ObjectType)
	if !ok {
		t.Fatalf("expected ObjectType, got %T", got)
	}
	if objType.AttrTypes["key"] != types.StringType {
		t.Fatalf("nested key: got %v", objType.AttrTypes["key"])
	}
}

func TestFieldToAttrType_array_with_itemspec(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "tags",
		Type:     "array",
		ItemSpec: &spec.FieldSpec{Name: "", Type: "string"},
	}
	got := fieldToAttrType(f)
	listType, ok := got.(types.ListType)
	if !ok {
		t.Fatalf("expected ListType, got %T", got)
	}
	if listType.ElemType != types.StringType {
		t.Fatalf("elem type: got %v", listType.ElemType)
	}
}

func TestFieldToAttrType_array_no_itemspec(t *testing.T) {
	f := &spec.FieldSpec{Name: "tags", Type: "array"}
	got := fieldToAttrType(f)
	listType, ok := got.(types.ListType)
	if !ok {
		t.Fatalf("expected ListType, got %T", got)
	}
	if listType.ElemType != types.StringType {
		t.Fatalf("default elem type should be StringType, got %v", listType.ElemType)
	}
}

// --- fieldToSchemaAttr ---------------------------------------------------------------------------

func TestFieldToSchemaAttr_id(t *testing.T) {
	f := &spec.FieldSpec{Name: "id", Type: "string", IsID: true}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Computed || attr.Required || attr.Optional {
		t.Fatal("ID field must be Computed only")
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("ID field must have UseNonNullStateForUnknown plan modifier")
	}
}

func TestFieldToSchemaAttr_required_writable_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "name", Type: "string", Required: true, Writable: true}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Required || attr.Optional || attr.Computed {
		t.Fatalf("required writable: Required=%v Optional=%v Computed=%v",
			attr.Required, attr.Optional, attr.Computed)
	}
}

func TestFieldToSchemaAttr_optional_writable_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "desc", Type: "string", Required: false, Writable: true}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if attr.Required || !attr.Optional {
		t.Fatalf("optional: Required=%v Optional=%v", attr.Required, attr.Optional)
	}
}

func TestFieldToSchemaAttr_computed_readonly(t *testing.T) {
	f := &spec.FieldSpec{Name: "created_at", Type: "string", Writable: false}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Computed || attr.Required || attr.Optional {
		t.Fatalf("readonly: Computed=%v Required=%v Optional=%v",
			attr.Computed, attr.Required, attr.Optional)
	}
}

func TestFieldToSchemaAttr_immutable_string(t *testing.T) {
	f := &spec.FieldSpec{
		Name:      "region",
		Type:      "string",
		Writable:  true,
		Required:  true,
		Immutable: true,
	}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("immutable field must have RequiresReplace plan modifier")
	}
}

func TestFieldToSchemaAttr_sensitive_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "password", Type: "string", Writable: true, Sensitive: true}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Sensitive {
		t.Fatal("expected Sensitive=true")
	}
}

func TestFieldToSchemaAttr_integer(t *testing.T) {
	f := &spec.FieldSpec{Name: "port", Type: "integer", Writable: true, Required: true}
	got := fieldToSchemaAttr(f)
	if _, ok := got.(schema.Int64Attribute); !ok {
		t.Fatalf("expected Int64Attribute, got %T", got)
	}
}

func TestFieldToSchemaAttr_integer_immutable(t *testing.T) {
	f := &spec.FieldSpec{
		Name:      "port",
		Type:      "integer",
		Writable:  true,
		Required:  true,
		Immutable: true,
	}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected Int64Attribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("immutable integer must have RequiresReplace plan modifier")
	}
}

func TestFieldToSchemaAttr_number(t *testing.T) {
	f := &spec.FieldSpec{Name: "weight", Type: "number", Writable: true}
	got := fieldToSchemaAttr(f)
	if _, ok := got.(schema.Float64Attribute); !ok {
		t.Fatalf("expected Float64Attribute, got %T", got)
	}
}

func TestFieldToSchemaAttr_boolean(t *testing.T) {
	f := &spec.FieldSpec{Name: "enabled", Type: "boolean", Writable: true}
	got := fieldToSchemaAttr(f)
	if _, ok := got.(schema.BoolAttribute); !ok {
		t.Fatalf("expected BoolAttribute, got %T", got)
	}
}

func TestFieldToSchemaAttr_boolean_immutable(t *testing.T) {
	f := &spec.FieldSpec{Name: "enabled", Type: "boolean", Writable: true, Immutable: true}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected BoolAttribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("immutable bool must have RequiresReplace plan modifier")
	}
}

func TestFieldToSchemaAttr_object(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "meta",
		Type:     "object",
		Writable: true,
		Nested: []*spec.FieldSpec{
			{
				Name:     "key",
				Type:     "string",
				Writable: true,
			},
		},
	}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected SingleNestedAttribute, got %T", got)
	}
	if _, ok := attr.Attributes["key"]; !ok {
		t.Fatal("expected nested key attribute")
	}
}

func TestFieldToSchemaAttr_array_of_strings(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "tags",
		Type:     "array",
		Writable: true,
		ItemSpec: &spec.FieldSpec{Name: "", Type: "string"},
	}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.ListAttribute)
	if !ok {
		t.Fatalf("expected ListAttribute, got %T", got)
	}
	if attr.ElementType != types.StringType {
		t.Fatalf("elem type: got %v", attr.ElementType)
	}
}

func TestFieldToSchemaAttr_array_of_objects(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "items",
		Type:     "array",
		Writable: true,
		ItemSpec: &spec.FieldSpec{
			Name: "",
			Type: "object",
			Nested: []*spec.FieldSpec{
				{
					Name:     "name",
					Type:     "string",
					Writable: true,
				},
			},
		},
	}
	got := fieldToSchemaAttr(f)
	attr, ok := got.(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected ListNestedAttribute, got %T", got)
	}
	if _, ok := attr.NestedObject.Attributes["name"]; !ok {
		t.Fatal("expected nested name attribute")
	}
}

// --- fieldToDSAttr -------------------------------------------------------------------------------

func TestFieldToDSAttr_primitives(t *testing.T) {
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
			got := fieldToDSAttr(&spec.FieldSpec{Name: "f", Type: c.typ})
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

func TestFieldToDSAttr_sensitive_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "token", Type: "string", Sensitive: true}
	got := fieldToDSAttr(f)
	attr, ok := got.(dsschema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Sensitive {
		t.Fatal("expected Sensitive=true")
	}
}

func TestFieldToDSAttr_object(t *testing.T) {
	f := &spec.FieldSpec{
		Name:   "meta",
		Type:   "object",
		Nested: []*spec.FieldSpec{{Name: "k", Type: "string"}},
	}
	got := fieldToDSAttr(f)
	attr, ok := got.(dsschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected SingleNestedAttribute, got %T", got)
	}
	if _, ok := attr.Attributes["k"]; !ok {
		t.Fatal("expected nested k attribute")
	}
}

func TestFieldToDSAttr_array_of_strings(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "tags",
		Type:     "array",
		ItemSpec: &spec.FieldSpec{Name: "", Type: "string"},
	}
	got := fieldToDSAttr(f)
	attr, ok := got.(dsschema.ListAttribute)
	if !ok {
		t.Fatalf("expected ListAttribute, got %T", got)
	}
	if attr.ElementType != types.StringType {
		t.Fatalf("elem type: got %v", attr.ElementType)
	}
}

func TestFieldToDSAttr_array_no_itemspec(t *testing.T) {
	f := &spec.FieldSpec{Name: "tags", Type: "array"}
	got := fieldToDSAttr(f)
	attr, ok := got.(dsschema.ListAttribute)
	if !ok {
		t.Fatalf("expected ListAttribute, got %T", got)
	}
	if attr.ElementType != types.StringType {
		t.Fatalf("default elem type should be StringType, got %v", attr.ElementType)
	}
}

func TestFieldToDSAttr_array_of_objects(t *testing.T) {
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
	got := fieldToDSAttr(f)
	attr, ok := got.(dsschema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected ListNestedAttribute, got %T", got)
	}
	if _, ok := attr.NestedObject.Attributes["name"]; !ok {
		t.Fatal("expected nested name attribute")
	}
}

// --- buildSchema / buildDataSourceSchema ---------------------------------------------------------

func TestBuildSchema_produces_correct_keys(t *testing.T) {
	fields := []*spec.FieldSpec{
		{Name: "id", Type: "string", IsID: true},
		{Name: "name", Type: "string", Writable: true, Required: true},
	}
	s, attrTypes := buildSchema(fields)
	if _, ok := s.Attributes["id"]; !ok {
		t.Fatal("expected id in schema")
	}
	if _, ok := s.Attributes["name"]; !ok {
		t.Fatal("expected name in schema")
	}
	if attrTypes["id"] != types.StringType {
		t.Fatalf("id attrType: got %v", attrTypes["id"])
	}
}

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
