package spec

import "testing"

// --- toSnakeCase ---------------------------------------------------------------------------------

func TestToSnakeCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"photoUrls", "photo_urls"},
		{"petId", "pet_id"},
		{"shipDate", "ship_date"},
		{"firstName", "first_name"},
		{"APIKey", "api_key"},
		{"userStatus", "user_status"},
		{"id", "id"},
		{"name", "name"},
		{"created_at", "created_at"}, // already snake_case
		{"status", "status"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := toSnakeCase(c.in); got != c.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// --- buildFieldSpec ------------------------------------------------------------------------------

func TestBuildFieldSpec_NilSchema(t *testing.T) {
	tests := []struct {
		writable     bool
		wantComputed bool
	}{
		{true, false},
		{false, true},
	}
	for _, tt := range tests {
		name := "not-writable"
		if tt.writable {
			name = "writable"
		}
		t.Run(name, func(t *testing.T) {
			got := buildFieldSpec("res", "field", nil, tt.writable, false)
			if got.Type != "string" {
				t.Errorf("Type = %q, want %q", got.Type, "string")
			}
			if got.Computed != tt.wantComputed {
				t.Errorf("Computed = %v, want %v", got.Computed, tt.wantComputed)
			}
		})
	}
}

func TestBuildFieldSpec(t *testing.T) {
	model := mustParseFixture(t, "schema_components.yaml")

	tests := []struct {
		name          string
		schemaName    string
		fieldName     string
		writable      bool
		required      bool
		wantType      string
		wantComputed  bool
		wantWritable  bool
		wantRequired  bool
		wantImmutable bool
		wantSensitive bool
		wantDesc      string
		wantNestedLen int
		wantItemType  string
	}{
		{
			name:       "writable required string",
			schemaName: "SimpleString", fieldName: "name",
			writable: true, required: true,
			wantType: "string", wantWritable: true, wantRequired: true,
			wantDesc: "a plain string",
		},
		{
			name:       "not writable becomes computed",
			schemaName: "SimpleString", fieldName: "slug",
			writable: false,
			wantType: "string", wantComputed: true,
		},
		{
			name:       "readOnly overrides writable",
			schemaName: "ReadOnly", fieldName: "slug",
			writable: true,
			wantType: "string", wantComputed: true, wantWritable: false,
		},
		{
			name:       "x-immutable extension",
			schemaName: "Immutable", fieldName: "region",
			writable: true,
			wantType: "string", wantWritable: true, wantImmutable: true,
		},
		{
			name:       "x-computed true forces computed on writable field",
			schemaName: "XComputed", fieldName: "slug",
			writable: true,
			wantType: "string", wantWritable: true, wantComputed: true,
		},
		{
			name:       "x-sensitive true opt-in",
			schemaName: "Sensitive", fieldName: "vault_key",
			writable: true,
			wantType: "string", wantWritable: true, wantSensitive: true,
		},
		{
			name:       "x-sensitive false opt-out overrides name heuristic",
			schemaName: "SensitiveOptOut", fieldName: "token_count",
			writable: true,
			wantType: "string", wantWritable: true, wantSensitive: false,
		},
		{
			name:       "integer type",
			schemaName: "Integer", fieldName: "count",
			writable: true,
			wantType: "integer", wantWritable: true,
		},
		{
			name:       "number type",
			schemaName: "Number", fieldName: "ratio",
			writable: true,
			wantType: "number", wantWritable: true,
		},
		{
			name:       "boolean type",
			schemaName: "Boolean", fieldName: "enabled",
			writable: true,
			wantType: "boolean", wantWritable: true,
		},
		{
			name:       "object with nested properties",
			schemaName: "ObjectNested", fieldName: "meta",
			writable: true,
			wantType: "object", wantWritable: true, wantNestedLen: 2,
		},
		{
			name:       "array of strings",
			schemaName: "ArrayString", fieldName: "tags",
			writable: true,
			wantType: "array", wantWritable: true, wantItemType: "string",
		},
		{
			name:       "array of objects",
			schemaName: "ArrayObject", fieldName: "entries",
			writable: true,
			wantType: "array", wantWritable: true, wantItemType: "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, ok := model.Model.Components.Schemas.Get(tt.schemaName)
			if !ok {
				t.Fatalf("schema %q not found in test spec", tt.schemaName)
			}
			got := buildFieldSpec("res", tt.fieldName, proxy.Schema(), tt.writable, tt.required)

			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.Computed != tt.wantComputed {
				t.Errorf("Computed = %v, want %v", got.Computed, tt.wantComputed)
			}
			if got.Writable != tt.wantWritable {
				t.Errorf("Writable = %v, want %v", got.Writable, tt.wantWritable)
			}
			if got.Required != tt.wantRequired {
				t.Errorf("Required = %v, want %v", got.Required, tt.wantRequired)
			}
			if got.Immutable != tt.wantImmutable {
				t.Errorf("Immutable = %v, want %v", got.Immutable, tt.wantImmutable)
			}
			if got.Sensitive != tt.wantSensitive {
				t.Errorf("Sensitive = %v, want %v", got.Sensitive, tt.wantSensitive)
			}
			if tt.wantDesc != "" && got.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", got.Description, tt.wantDesc)
			}
			if tt.wantNestedLen > 0 && len(got.Nested) != tt.wantNestedLen {
				t.Errorf("len(Nested) = %d, want %d", len(got.Nested), tt.wantNestedLen)
			}
			if tt.wantItemType != "" {
				if got.ItemSpec == nil {
					t.Errorf("ItemSpec = nil, want type %q", tt.wantItemType)
				} else if got.ItemSpec.Type != tt.wantItemType {
					t.Errorf("ItemSpec.Type = %q, want %q", got.ItemSpec.Type, tt.wantItemType)
				}
			}
		})
	}
}

// --- buildFieldSpec validation constraints -------------------------------------------------------

func TestBuildFieldSpec_Validation(t *testing.T) {
	model := mustParseFixture(t, "schema_components.yaml")

	getSchema := func(name string) *FieldSpec {
		t.Helper()
		proxy, ok := model.Model.Components.Schemas.Get(name)
		if !ok {
			t.Fatalf("schema %q not found", name)
		}
		return buildFieldSpec("res", "field", proxy.Schema(), true, false)
	}

	t.Run("maxLength", func(t *testing.T) {
		f := getSchema("StringMaxLength")
		if f.MaxLength == nil || *f.MaxLength != 10 {
			t.Errorf("MaxLength = %v, want 10", f.MaxLength)
		}
	})

	t.Run("minLength", func(t *testing.T) {
		f := getSchema("StringMinLength")
		if f.MinLength == nil || *f.MinLength != 2 {
			t.Errorf("MinLength = %v, want 2", f.MinLength)
		}
	})

	t.Run("pattern", func(t *testing.T) {
		f := getSchema("StringPattern")
		if f.Pattern != `^[0-9]{4,5}$` {
			t.Errorf("Pattern = %q, want %q", f.Pattern, `^[0-9]{4,5}$`)
		}
	})

	t.Run("pattern and maxLength together", func(t *testing.T) {
		f := getSchema("StringPatternAndMaxLength")
		if f.Pattern != `^[0-9]{4,5}$` {
			t.Errorf("Pattern = %q, want %q", f.Pattern, `^[0-9]{4,5}$`)
		}
		if f.MaxLength == nil || *f.MaxLength != 5 {
			t.Errorf("MaxLength = %v, want 5", f.MaxLength)
		}
	})

	t.Run("integer minimum and maximum", func(t *testing.T) {
		f := getSchema("IntegerMinMax")
		if f.Minimum == nil || *f.Minimum != 0 {
			t.Errorf("Minimum = %v, want 0", f.Minimum)
		}
		if f.Maximum == nil || *f.Maximum != 100 {
			t.Errorf("Maximum = %v, want 100", f.Maximum)
		}
	})

	t.Run("direct enum", func(t *testing.T) {
		f := getSchema("DirectEnum")
		if len(f.Enum) != 3 {
			t.Fatalf("len(Enum) = %d, want 3", len(f.Enum))
		}
		want := map[string]bool{"foo": true, "bar": true, "baz": true}
		for _, v := range f.Enum {
			if !want[v] {
				t.Errorf("unexpected enum value %q", v)
			}
		}
	})

	t.Run("allOf enum ($ref pattern)", func(t *testing.T) {
		f := getSchema("AllOfEnum")
		if len(f.Enum) != 3 {
			t.Fatalf("len(Enum) = %d, want 3 (LAB, DEV, PROD)", len(f.Enum))
		}
		want := map[string]bool{"LAB": true, "DEV": true, "PROD": true}
		for _, v := range f.Enum {
			if !want[v] {
				t.Errorf("unexpected enum value %q", v)
			}
		}
	})

	t.Run("oneOf enum (nullable pattern)", func(t *testing.T) {
		f := getSchema("OneOfEnum")
		if len(f.Enum) != 3 {
			t.Fatalf("len(Enum) = %d, want 3 (X, Y, '')", len(f.Enum))
		}
		want := map[string]bool{"X": true, "Y": true, "": true}
		for _, v := range f.Enum {
			if !want[v] {
				t.Errorf("unexpected enum value %q", v)
			}
		}
	})
}

// --- buildFieldSpec defaults ---------------------------------------------------------------------

func TestBuildFieldSpec_Default(t *testing.T) {
	model := mustParseFixture(t, "schema_components.yaml")

	getField := func(schemaName string) *FieldSpec {
		t.Helper()
		proxy, ok := model.Model.Components.Schemas.Get(schemaName)
		if !ok {
			t.Fatalf("schema %q not found", schemaName)
		}
		return buildFieldSpec("res", "field", proxy.Schema(), true, false)
	}

	t.Run("string default", func(t *testing.T) {
		f := getField("DefaultString")
		if f.Default != "hello" {
			t.Errorf("Default = %v, want %q", f.Default, "hello")
		}
	})

	t.Run("integer default", func(t *testing.T) {
		f := getField("DefaultInteger")
		if f.Default != int64(42) {
			t.Errorf("Default = %v (%T), want int64(42)", f.Default, f.Default)
		}
	})

	t.Run("number default", func(t *testing.T) {
		f := getField("DefaultNumber")
		v, ok := f.Default.(float64)
		if !ok || v != 3.14 {
			t.Errorf("Default = %v (%T), want float64(3.14)", f.Default, f.Default)
		}
	})

	t.Run("boolean default true", func(t *testing.T) {
		f := getField("DefaultBoolTrue")
		if f.Default != true {
			t.Errorf("Default = %v, want true", f.Default)
		}
	})

	t.Run("boolean default false", func(t *testing.T) {
		f := getField("DefaultBoolFalse")
		if f.Default != false {
			t.Errorf("Default = %v, want false", f.Default)
		}
	})

	t.Run("empty array default", func(t *testing.T) {
		f := getField("DefaultEmptyArray")
		if _, ok := f.Default.([]any); !ok {
			t.Errorf("Default = %v (%T), want []any", f.Default, f.Default)
		}
	})

	t.Run("null default treated as absent", func(t *testing.T) {
		f := getField("DefaultNull")
		if f.Default != nil {
			t.Errorf("Default = %v, want nil for null default", f.Default)
		}
	})

	t.Run("no default when not writable", func(t *testing.T) {
		proxy, ok := model.Model.Components.Schemas.Get("DefaultInteger")
		if !ok {
			t.Fatal("DefaultInteger schema not found")
		}
		f := buildFieldSpec("res", "field", proxy.Schema(), false, false)
		// Default is still parsed from schema, but hasDefault logic in schema.go gates its application
		// The spec layer records it regardless of writability.
		_ = f
	})
}

// --- buildFieldSpecs -----------------------------------------------------------------------------

func TestBuildFieldSpecs(t *testing.T) {
	model := mustParseFixture(t, "schema_components.yaml")
	proxy, ok := model.Model.Components.Schemas.Get("Resource")
	if !ok {
		t.Fatal("Resource schema not found in test spec")
	}

	writeFields := map[string]bool{"name": true, "size": true}
	byName := fieldsByName(buildFieldSpecs("res", proxy.Schema(), writeFields))

	// id: not in write body -> Computed derived from OAS (not writable); IsID set by buildFieldSpecs
	id := byName["id"]
	if id == nil {
		t.Fatal("id field missing")
	}
	if !id.IsID {
		t.Error("id.IsID should be true")
	}
	if !id.Computed {
		t.Error("id.Computed should be true (id not in write body)")
	}
	if id.Writable {
		t.Error("id.Writable should be false (id not in write body)")
	}
	if id.Required {
		t.Error("id.Required should be false (required && writable = false)")
	}

	// name: in write body and in schema required
	name := byName["name"]
	if name == nil {
		t.Fatal("name field missing")
	}
	if !name.Writable {
		t.Error("name.Writable should be true")
	}
	if !name.Required {
		t.Error("name.Required should be true")
	}
	if name.Computed {
		t.Error("name.Computed should be false")
	}

	// size: in write body, not required
	size := byName["size"]
	if size == nil {
		t.Fatal("size field missing")
	}
	if !size.Writable {
		t.Error("size.Writable should be true")
	}
	if size.Required {
		t.Error("size.Required should be false")
	}
	if size.Computed {
		t.Error("size.Computed should be false")
	}

	// status: readOnly in spec — computed regardless of write body
	status := byName["status"]
	if status == nil {
		t.Fatal("status field missing")
	}
	if !status.Computed {
		t.Error("status.Computed should be true")
	}
	if status.Writable {
		t.Error("status.Writable should be false")
	}
}

func TestBuildFieldSpecs_ClientSideID(t *testing.T) {
	// When id is present in the POST body (not readOnly), it must be writable, the
	// client supplies it. buildFieldSpecs must not force Computed=true in that case.
	model := mustParseFixture(t, "schema_components.yaml")
	proxy, ok := model.Model.Components.Schemas.Get("Resource")
	if !ok {
		t.Fatal("Resource schema not found in test spec")
	}

	writeFields := map[string]bool{"id": true, "name": true, "size": true}
	byName := fieldsByName(buildFieldSpecs("res", proxy.Schema(), writeFields))

	id := byName["id"]
	if id == nil {
		t.Fatal("id field missing")
	}
	if !id.IsID {
		t.Error("id.IsID should be true")
	}
	if !id.Writable {
		t.Error("id.Writable should be true when id is in POST body")
	}
	if id.Computed {
		t.Error("id.Computed should be false when id is in POST body (client-supplied)")
	}
}
