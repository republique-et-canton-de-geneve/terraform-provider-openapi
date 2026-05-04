package spec

import (
	"sort"
	"testing"

	"github.com/pb33f/libopenapi"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// --- Helpers -------------------------------------------------------------------------------------

func parseSpec(t *testing.T, yaml string) *libopenapi.DocumentModel[v3high.Document] {
	t.Helper()
	doc, err := libopenapi.NewDocument([]byte(yaml))
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		t.Fatalf("BuildV3Model: %v", err)
	}
	return model
}

func fieldsByName(fields []*FieldSpec) map[string]*FieldSpec {
	m := make(map[string]*FieldSpec, len(fields))
	for _, f := range fields {
		m[f.Name] = f
	}
	return m
}

// schemaComponentsYAML is a spec with one schema per feature exercised by buildFieldSpec.
const schemaComponentsYAML = `
openapi: "3.0.3"
info:
  title: Test
  version: "1"
paths: {}
components:
  schemas:
    SimpleString:
      type: string
      description: "a plain string"
    ReadOnly:
      type: string
      readOnly: true
    Immutable:
      type: string
      x-immutable: true
    Sensitive:
      type: string
      x-sensitive: true
    SensitiveOptOut:
      type: string
      x-sensitive: false
    Integer:
      type: integer
    Number:
      type: number
    Boolean:
      type: boolean
    ObjectNested:
      type: object
      required:
        - inner_name
      properties:
        inner_name:
          type: string
        inner_count:
          type: integer
    ArrayString:
      type: array
      items:
        type: string
    ArrayObject:
      type: array
      items:
        type: object
        properties:
          item_id:
            type: integer
          item_name:
            type: string
    Resource:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
        name:
          type: string
        size:
          type: integer
        status:
          type: string
          readOnly: true
`

// widgetsSpecYAML is a realistic spec with one full-CRUD resource under /api/v1/.
const widgetsSpecYAML = `
openapi: "3.0.3"
info:
  title: Widgets API
  version: "1"
paths:
  /api/v1/widgets/:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                size:
                  type: integer
      responses:
        "201":
          description: Created
  /api/v1/widgets/{id}/:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                required:
                  - id
                  - name
                properties:
                  id:
                    type: integer
                  name:
                    type: string
                  size:
                    type: integer
                  created_at:
                    type: string
                    format: date-time
                    readOnly: true
    patch:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                size:
                  type: integer
      responses:
        "200":
          description: OK
    delete:
      responses:
        "204":
          description: No Content
`

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
			got := buildFieldSpec("field", nil, tt.writable, false)
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
	model := parseSpec(t, schemaComponentsYAML)

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
			got := buildFieldSpec(tt.fieldName, proxy.Schema(), tt.writable, tt.required)

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

// --- buildFieldSpecs -----------------------------------------------------------------------------

func TestBuildFieldSpecs(t *testing.T) {
	model := parseSpec(t, schemaComponentsYAML)
	proxy, ok := model.Model.Components.Schemas.Get("Resource")
	if !ok {
		t.Fatal("Resource schema not found in test spec")
	}

	writeFields := map[string]bool{"name": true, "size": true}
	byName := fieldsByName(buildFieldSpecs(proxy.Schema(), writeFields))

	// id: not in write body → Computed derived from OAS (not writable); IsID set by buildFieldSpecs
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
	model := parseSpec(t, schemaComponentsYAML)
	proxy, ok := model.Model.Components.Schemas.Get("Resource")
	if !ok {
		t.Fatal("Resource schema not found in test spec")
	}

	writeFields := map[string]bool{"id": true, "name": true, "size": true}
	byName := fieldsByName(buildFieldSpecs(proxy.Schema(), writeFields))

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

// --- DiscoverResources ---------------------------------------------------------------------------

func TestDiscoverResources_NilModel(t *testing.T) {
	if got := DiscoverResources(nil); got != nil {
		t.Errorf("DiscoverResources(nil) = %v, want nil", got)
	}
}

func TestDiscoverResources_EmptyPaths(t *testing.T) {
	model := parseSpec(t, `
openapi: "3.0.3"
info:
  title: Empty
  version: "1"
paths: {}
`)
	if got := DiscoverResources(model); len(got) != 0 {
		t.Errorf("expected no resources, got %d", len(got))
	}
}

func TestDiscoverResources_FullCRUD(t *testing.T) {
	specs := DiscoverResources(parseSpec(t, widgetsSpecYAML))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	rs := specs[0]

	if rs.Name != "widgets" {
		t.Errorf("Name = %q, want %q", rs.Name, "widgets")
	}
	if rs.ListPath != "/api/v1/widgets/" {
		t.Errorf("ListPath = %q, want %q", rs.ListPath, "/api/v1/widgets/")
	}
	if rs.ItemPath != "/api/v1/widgets/{id}/" {
		t.Errorf("ItemPath = %q, want %q", rs.ItemPath, "/api/v1/widgets/{id}/")
	}
	if rs.IDPathParam != "id" {
		t.Errorf("IDPathParam = %q, want %q", rs.IDPathParam, "id")
	}
	if !rs.HasCreate {
		t.Error("HasCreate should be true")
	}
	if !rs.HasUpdate {
		t.Error("HasUpdate should be true")
	}
	if rs.UpdateMethod != "PATCH" {
		t.Errorf("UpdateMethod = %q, want PATCH", rs.UpdateMethod)
	}
	if !rs.HasDelete {
		t.Error("HasDelete should be true")
	}
	if len(rs.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(rs.Fields))
	}

	byName := fieldsByName(rs.Fields)

	id := byName["id"]
	if id == nil {
		t.Fatal("id field missing")
	}
	if !id.IsID || !id.Computed || id.Writable {
		t.Errorf("id: IsID=%v Computed=%v Writable=%v, want true/true/false",
			id.IsID, id.Computed, id.Writable)
	}

	name := byName["name"]
	if name == nil {
		t.Fatal("name field missing")
	}
	if !name.Writable || !name.Required || name.Computed {
		t.Errorf("name: Writable=%v Required=%v Computed=%v, want true/true/false",
			name.Writable, name.Required, name.Computed)
	}

	createdAt := byName["created_at"]
	if createdAt == nil {
		t.Fatal("created_at field missing")
	}
	if !createdAt.Computed || createdAt.Writable {
		t.Errorf("created_at: Computed=%v Writable=%v, want true/false",
			createdAt.Computed, createdAt.Writable)
	}
	if createdAt.Format != "date-time" {
		t.Errorf("created_at.Format = %q, want %q", createdAt.Format, "date-time")
	}
}

func TestDiscoverResources_PUTUpdate(t *testing.T) {
	specs := DiscoverResources(parseSpec(t, `
openapi: "3.0.3"
info:
  title: Test
  version: "1"
paths:
  /api/v1/items/:
    post:
      responses:
        "201":
          description: Created
  /api/v1/items/{id}/:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  label:
                    type: string
    put:
      responses:
        "200":
          description: OK
`))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	if specs[0].UpdateMethod != "PUT" {
		t.Errorf("UpdateMethod = %q, want PUT", specs[0].UpdateMethod)
	}
}

func TestDiscoverResources_NoGETExcludesResource(t *testing.T) {
	specs := DiscoverResources(parseSpec(t, `
openapi: "3.0.3"
info:
  title: Test
  version: "1"
paths:
  /api/v1/items/:
    post:
      responses:
        "201":
          description: Created
  /api/v1/items/{id}/:
    delete:
      responses:
        "204":
          description: No Content
`))
	if len(specs) != 0 {
		t.Errorf("expected 0 resources (no GET on item), got %d", len(specs))
	}
}

func TestDiscoverResources_ItemPathOnly(t *testing.T) {
	specs := DiscoverResources(parseSpec(t, `
openapi: "3.0.3"
info:
  title: Test
  version: "1"
paths:
  /api/v1/items/{id}/:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  label:
                    type: string
`))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	rs := specs[0]
	if rs.HasCreate {
		t.Error("HasCreate should be false (no list path)")
	}
	if rs.ListPath != "" {
		t.Errorf("ListPath = %q, want empty", rs.ListPath)
	}
}

func TestDiscoverResources_CamelCaseFieldsConvertedToSnake(t *testing.T) {
	specs := DiscoverResources(parseSpec(t, `
openapi: "3.0.3"
info:
  title: Pets
  version: "1"
paths:
  /pets/:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                photoUrls:
                  type: array
                  items:
                    type: string
                petName:
                  type: string
      responses:
        "201":
          description: Created
  /pets/{petId}/:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
                  petName:
                    type: string
                  photoUrls:
                    type: array
                    items:
                      type: string
    delete:
      responses:
        "204":
          description: No Content
`))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	byName := fieldsByName(specs[0].Fields)

	petName := byName["pet_name"]
	if petName == nil {
		t.Fatal("field 'pet_name' missing (expected snake_case conversion of 'petName')")
	}
	if petName.OASName != "petName" {
		t.Errorf("pet_name.OASName = %q, want %q", petName.OASName, "petName")
	}

	photoUrls := byName["photo_urls"]
	if photoUrls == nil {
		t.Fatal("field 'photo_urls' missing (expected snake_case conversion of 'photoUrls')")
	}
	if photoUrls.OASName != "photoUrls" {
		t.Errorf("photo_urls.OASName = %q, want %q", photoUrls.OASName, "photoUrls")
	}
}

func TestDiscoverResources_MultipleResources(t *testing.T) {
	specs := DiscoverResources(parseSpec(t, `
openapi: "3.0.3"
info:
  title: Test
  version: "1"
paths:
  /api/v1/widgets/:
    post:
      responses:
        "201":
          description: Created
  /api/v1/widgets/{id}/:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
  /api/v1/things/:
    post:
      responses:
        "201":
          description: Created
  /api/v1/things/{id}/:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
`))
	if len(specs) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(specs))
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	if specs[0].Name != "things" {
		t.Errorf("specs[0].Name = %q, want things", specs[0].Name)
	}
	if specs[1].Name != "widgets" {
		t.Errorf("specs[1].Name = %q, want widgets", specs[1].Name)
	}
}
