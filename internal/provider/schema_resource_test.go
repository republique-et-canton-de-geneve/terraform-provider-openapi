package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/republique-et-canton-de-geneve/terraform-provider-openapi/internal/spec"
)

// --- buildResourceSchema -------------------------------------------------------------------------

func TestBuildResourceSchema_produces_correct_keys(t *testing.T) {
	fields := []*spec.FieldSpec{
		{Name: "id", Type: "string", IsID: true},
		{Name: "name", Type: "string", Writable: true, Required: true},
	}
	s, attrTypes := buildResourceSchema(fields)
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

// --- fieldToResourceSchemaAttr -------------------------------------------------------------------

func TestFieldToResourceSchemaAttr_id(t *testing.T) {
	f := &spec.FieldSpec{Name: "id", Type: "string", IsID: true}
	got := fieldToResourceSchemaAttr(f)
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

func TestFieldToResourceSchemaAttr_required_writable_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "name", Type: "string", Required: true, Writable: true}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Required || attr.Optional || attr.Computed {
		t.Fatalf("required writable: Required=%v Optional=%v Computed=%v",
			attr.Required, attr.Optional, attr.Computed)
	}
}

func TestFieldToResourceSchemaAttr_optional_writable_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "desc", Type: "string", Required: false, Writable: true}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if attr.Required || !attr.Optional {
		t.Fatalf("optional: Required=%v Optional=%v", attr.Required, attr.Optional)
	}
}

func TestFieldToResourceSchemaAttr_computed_readonly(t *testing.T) {
	f := &spec.FieldSpec{Name: "created_at", Type: "string", Writable: false}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Computed || attr.Required || attr.Optional {
		t.Fatalf("readonly: Computed=%v Required=%v Optional=%v",
			attr.Computed, attr.Required, attr.Optional)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("computed field must have UseNonNullStateForUnknown plan modifier")
	}
}

func TestFieldToResourceSchemaAttr_immutable_string(t *testing.T) {
	f := &spec.FieldSpec{
		Name:      "region",
		Type:      "string",
		Writable:  true,
		Required:  true,
		Immutable: true,
	}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("immutable field must have RequiresReplace plan modifier")
	}
}

func TestFieldToResourceSchemaAttr_sensitive_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "password", Type: "string", Writable: true, Sensitive: true}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if !attr.Sensitive {
		t.Fatal("expected Sensitive=true")
	}
}

func TestFieldToResourceSchemaAttr_integer(t *testing.T) {
	f := &spec.FieldSpec{Name: "port", Type: "integer", Writable: true, Required: true}
	got := fieldToResourceSchemaAttr(f)
	if _, ok := got.(schema.Int64Attribute); !ok {
		t.Fatalf("expected Int64Attribute, got %T", got)
	}
}

func TestFieldToResourceSchemaAttr_integer_immutable(t *testing.T) {
	f := &spec.FieldSpec{
		Name:      "port",
		Type:      "integer",
		Writable:  true,
		Required:  true,
		Immutable: true,
	}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected Int64Attribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("immutable integer must have RequiresReplace plan modifier")
	}
}

func TestFieldToResourceSchemaAttr_number(t *testing.T) {
	f := &spec.FieldSpec{Name: "weight", Type: "number", Writable: true}
	got := fieldToResourceSchemaAttr(f)
	if _, ok := got.(schema.Float64Attribute); !ok {
		t.Fatalf("expected Float64Attribute, got %T", got)
	}
}

func TestFieldToResourceSchemaAttr_number_immutable(t *testing.T) {
	f := &spec.FieldSpec{
		Name:      "ratio",
		Type:      "number",
		Writable:  true,
		Required:  true,
		Immutable: true,
	}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.Float64Attribute)
	if !ok {
		t.Fatalf("expected Float64Attribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("immutable number must have RequiresReplace plan modifier")
	}
}

func TestFieldToResourceSchemaAttr_immutable_computed_string(t *testing.T) {
	// x-immutable + server-computed: both UseNonNullStateForUnknown and RequiresReplace.
	f := &spec.FieldSpec{
		Name:      "region",
		Type:      "string",
		Writable:  true,
		Computed:  true,
		Immutable: true,
	}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 2 {
		t.Fatalf(
			"immutable computed field must have 2 plan modifiers, got %d",
			len(attr.PlanModifiers))
	}
}

func TestFieldToResourceSchemaAttr_boolean(t *testing.T) {
	f := &spec.FieldSpec{Name: "enabled", Type: "boolean", Writable: true}
	got := fieldToResourceSchemaAttr(f)
	if _, ok := got.(schema.BoolAttribute); !ok {
		t.Fatalf("expected BoolAttribute, got %T", got)
	}
}

func TestFieldToResourceSchemaAttr_boolean_immutable(t *testing.T) {
	f := &spec.FieldSpec{Name: "enabled", Type: "boolean", Writable: true, Immutable: true}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected BoolAttribute, got %T", got)
	}
	if len(attr.PlanModifiers) != 1 {
		t.Fatal("immutable bool must have RequiresReplace plan modifier")
	}
}

func TestFieldToResourceSchemaAttr_untyped(t *testing.T) {
	f := &spec.FieldSpec{Name: "payload", Type: "untyped", Writable: true}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if _, ok := attr.CustomType.(jsontypes.NormalizedType); !ok {
		t.Fatalf("expected NormalizedType CustomType, got %T", attr.CustomType)
	}
}

func TestFieldToResourceSchemaAttr_object(t *testing.T) {
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
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("expected SingleNestedAttribute, got %T", got)
	}
	if _, ok := attr.Attributes["key"]; !ok {
		t.Fatal("expected nested key attribute")
	}
}

func TestFieldToResourceSchemaAttr_array_of_strings(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "tags",
		Type:     "array",
		Writable: true,
		ItemSpec: &spec.FieldSpec{Name: "", Type: "string"},
	}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.ListAttribute)
	if !ok {
		t.Fatalf("expected ListAttribute, got %T", got)
	}
	if attr.ElementType != types.StringType {
		t.Fatalf("elem type: got %v", attr.ElementType)
	}
}

func TestFieldToResourceSchemaAttr_array_of_objects(t *testing.T) {
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
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("expected ListNestedAttribute, got %T", got)
	}
	if _, ok := attr.NestedObject.Attributes["name"]; !ok {
		t.Fatal("expected nested name attribute")
	}
}

// --- fieldToResourceSchemaAttr defaults ----------------------------------------------------------

func TestFieldToResourceSchemaAttr_default_string(t *testing.T) {
	f := &spec.FieldSpec{Name: "status", Type: "string", Writable: true, Default: "active"}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if attr.Default == nil {
		t.Fatal("expected Default to be set")
	}
	if !attr.Computed {
		t.Fatal("field with default must be Computed")
	}
	if !attr.Optional {
		t.Fatal("field with default must be Optional")
	}
}

func TestFieldToResourceSchemaAttr_default_integer(t *testing.T) {
	f := &spec.FieldSpec{Name: "size", Type: "integer", Writable: true, Default: int64(30)}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.Int64Attribute)
	if !ok {
		t.Fatalf("expected Int64Attribute, got %T", got)
	}
	if attr.Default == nil {
		t.Fatal("expected Default to be set")
	}
	if !attr.Computed || !attr.Optional {
		t.Fatal("field with default must be Optional+Computed")
	}
}

func TestFieldToResourceSchemaAttr_default_boolean(t *testing.T) {
	f := &spec.FieldSpec{Name: "enabled", Type: "boolean", Writable: true, Default: false}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.BoolAttribute)
	if !ok {
		t.Fatalf("expected BoolAttribute, got %T", got)
	}
	if attr.Default == nil {
		t.Fatal("expected Default to be set")
	}
	if !attr.Computed || !attr.Optional {
		t.Fatal("field with default must be Optional+Computed")
	}
}

func TestFieldToResourceSchemaAttr_default_empty_array(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "emails",
		Type:     "array",
		Writable: true,
		Default:  []any{},
		ItemSpec: &spec.FieldSpec{Name: "", Type: "string"},
	}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.ListAttribute)
	if !ok {
		t.Fatalf("expected ListAttribute, got %T", got)
	}
	if attr.Default == nil {
		t.Fatal("expected Default to be set")
	}
	if !attr.Computed || !attr.Optional {
		t.Fatal("field with default must be Optional+Computed")
	}
}

func TestFieldToResourceSchemaAttr_default_not_applied_to_readonly(t *testing.T) {
	// Default in spec on a non-writable field (e.g. readOnly) must not be applied to schema.
	f := &spec.FieldSpec{Name: "created_at", Type: "string", Writable: false, Default: "now"}
	got := fieldToResourceSchemaAttr(f)
	attr, ok := got.(schema.StringAttribute)
	if !ok {
		t.Fatalf("expected StringAttribute, got %T", got)
	}
	if attr.Default != nil {
		t.Fatal("Default must not be set on non-writable (readonly) fields")
	}
}

// --- fieldToResourceAttrType ---------------------------------------------------------------------

func TestFieldToResourceAttrType_primitives(t *testing.T) {
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
			got := fieldToResourceAttrType(&spec.FieldSpec{Name: "f", Type: c.typ})
			if got != c.want {
				t.Fatalf("type %q: got %v, want %v", c.typ, got, c.want)
			}
		})
	}
}

func TestFieldToResourceAttrType_untyped_is_normalized(t *testing.T) {
	got := fieldToResourceAttrType(&spec.FieldSpec{Name: "payload", Type: "untyped"})
	if _, ok := got.(jsontypes.NormalizedType); !ok {
		t.Fatalf("expected NormalizedType, got %T", got)
	}
}

func TestFieldToResourceAttrType_id_always_string(t *testing.T) {
	// Even an integer ID field must map to StringType for terraform import.
	got := fieldToResourceAttrType(&spec.FieldSpec{Name: "id", Type: "integer", IsID: true})
	if got != types.StringType {
		t.Fatalf("got %v, want StringType", got)
	}
}

func TestFieldToResourceAttrType_object(t *testing.T) {
	f := &spec.FieldSpec{
		Name:   "meta",
		Type:   "object",
		Nested: []*spec.FieldSpec{{Name: "key", Type: "string"}},
	}
	got := fieldToResourceAttrType(f)
	objType, ok := got.(types.ObjectType)
	if !ok {
		t.Fatalf("expected ObjectType, got %T", got)
	}
	if objType.AttrTypes["key"] != types.StringType {
		t.Fatalf("nested key: got %v", objType.AttrTypes["key"])
	}
}

func TestFieldToResourceAttrType_array_with_itemspec(t *testing.T) {
	f := &spec.FieldSpec{
		Name:     "tags",
		Type:     "array",
		ItemSpec: &spec.FieldSpec{Name: "", Type: "string"},
	}
	got := fieldToResourceAttrType(f)
	listType, ok := got.(types.ListType)
	if !ok {
		t.Fatalf("expected ListType, got %T", got)
	}
	if listType.ElemType != types.StringType {
		t.Fatalf("elem type: got %v", listType.ElemType)
	}
}

func TestFieldToResourceAttrType_array_no_itemspec(t *testing.T) {
	f := &spec.FieldSpec{Name: "tags", Type: "array"}
	got := fieldToResourceAttrType(f)
	listType, ok := got.(types.ListType)
	if !ok {
		t.Fatalf("expected ListType, got %T", got)
	}
	if listType.ElemType != types.StringType {
		t.Fatalf("default elem type should be StringType, got %v", listType.ElemType)
	}
}
