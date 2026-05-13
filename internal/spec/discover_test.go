package spec

import (
	"sort"
	"testing"
)

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
	specs := DiscoverResources(mustParseFixture(t, "widgets.yaml"))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	rs := specs[0]

	if rs.SingularName != "widget" {
		t.Errorf("SingularName = %q, want %q", rs.SingularName, "widget")
	}
	if rs.PluralName != "widgets" {
		t.Errorf("PluralName = %q, want %q", rs.PluralName, "widgets")
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
		return
	}
	if !id.IsID || !id.Computed || id.Writable {
		t.Errorf("id: IsID=%v Computed=%v Writable=%v, want true/true/false",
			id.IsID, id.Computed, id.Writable)
	}

	name := byName["name"]
	if name == nil {
		t.Fatal("name field missing")
		return
	}
	if !name.Writable || !name.Required || name.Computed {
		t.Errorf("name: Writable=%v Required=%v Computed=%v, want true/true/false",
			name.Writable, name.Required, name.Computed)
	}

	createdAt := byName["created_at"]
	if createdAt == nil {
		t.Fatal("created_at field missing")
		return
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
		return
	}
	if petName.OASName != "petName" {
		t.Errorf("pet_name.OASName = %q, want %q", petName.OASName, "petName")
	}

	photoUrls := byName["photo_urls"]
	if photoUrls == nil {
		t.Fatal("field 'photo_urls' missing (expected snake_case conversion of 'photoUrls')")
		return
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
	sort.Slice(specs, func(i, j int) bool { return specs[i].SingularName < specs[j].SingularName })
	if specs[0].SingularName != "thing" {
		t.Errorf("specs[0].SingularName = %q, want thing", specs[0].SingularName)
	}
	if specs[1].SingularName != "widget" {
		t.Errorf("specs[1].SingularName = %q, want widget", specs[1].SingularName)
	}
}

// --- DiscoverResources x-timeout -----------------------------------------------------------------

func TestDiscoverResources_Timeouts(t *testing.T) {
	specs := DiscoverResources(mustParseFixture(t, "timeouts.yaml"))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	rs := specs[0]

	if rs.Timeouts.List != "2m" {
		t.Errorf("Timeouts.List = %q, want %q", rs.Timeouts.List, "2m")
	}
	if rs.Timeouts.Create != "30m" {
		t.Errorf("Timeouts.Create = %q, want %q", rs.Timeouts.Create, "30m")
	}
	if rs.Timeouts.Read != "10s" {
		t.Errorf("Timeouts.Read = %q, want %q", rs.Timeouts.Read, "10s")
	}
	if rs.Timeouts.Update != "15m" {
		t.Errorf("Timeouts.Update = %q, want %q", rs.Timeouts.Update, "15m")
	}
	if rs.Timeouts.Delete != "10m" {
		t.Errorf("Timeouts.Delete = %q, want %q", rs.Timeouts.Delete, "10m")
	}
}

func TestDiscoverResources_TimeoutsAbsent(t *testing.T) {
	specs := DiscoverResources(mustParseFixture(t, "widgets.yaml"))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	rs := specs[0]

	if rs.Timeouts.List != "" {
		t.Errorf("Timeouts.List = %q, want empty", rs.Timeouts.List)
	}
	if rs.Timeouts.Create != "" {
		t.Errorf("Timeouts.Create = %q, want empty", rs.Timeouts.Create)
	}
	if rs.Timeouts.Read != "" {
		t.Errorf("Timeouts.Read = %q, want empty", rs.Timeouts.Read)
	}
	if rs.Timeouts.Update != "" {
		t.Errorf("Timeouts.Update = %q, want empty", rs.Timeouts.Update)
	}
	if rs.Timeouts.Delete != "" {
		t.Errorf("Timeouts.Delete = %q, want empty", rs.Timeouts.Delete)
	}
}

// --- DiscoverResources validation ----------------------------------------------------------------

func TestDiscoverResources_Validation(t *testing.T) {
	specs := DiscoverResources(mustParseFixture(t, "validation.yaml"))
	if len(specs) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(specs))
	}
	rs := specs[0]

	if rs.SingularName != "item" {
		t.Errorf("SingularName = %q, want item", rs.SingularName)
	}
	if rs.PluralName != "items" {
		t.Errorf("PluralName = %q, want items", rs.PluralName)
	}
	if !rs.HasCreate || !rs.HasUpdate || !rs.HasDelete {
		t.Errorf("HasCreate=%v HasUpdate=%v HasDelete=%v, want all true",
			rs.HasCreate, rs.HasUpdate, rs.HasDelete)
	}
	if rs.UpdateMethod != "PATCH" {
		t.Errorf("UpdateMethod = %q, want PATCH", rs.UpdateMethod)
	}

	byName := fieldsByName(rs.Fields)

	t.Run("maxLength on name", func(t *testing.T) {
		f := byName["name"]
		if f == nil {
			t.Fatal("field name missing")
			return
		}
		if f.MaxLength == nil || *f.MaxLength != 255 {
			t.Errorf("MaxLength = %v, want 255", f.MaxLength)
		}
	})

	t.Run("pattern and maxLength on code", func(t *testing.T) {
		f := byName["code"]
		if f == nil {
			t.Fatal("field code missing")
			return
		}
		if f.Pattern != `^[0-9]{4,5}$` {
			t.Errorf("Pattern = %q, want ^[0-9]{4,5}$", f.Pattern)
		}
		if f.MaxLength == nil || *f.MaxLength != 5 {
			t.Errorf("MaxLength = %v, want 5", f.MaxLength)
		}
	})

	t.Run("integer min/max on disk_size", func(t *testing.T) {
		f := byName["disk_size"]
		if f == nil {
			t.Fatal("field disk_size missing")
			return
		}
		if f.Minimum == nil || *f.Minimum != 0 {
			t.Errorf("Minimum = %v, want 0", f.Minimum)
		}
		if f.Maximum == nil || *f.Maximum != 4096 {
			t.Errorf("Maximum = %v, want 4096", f.Maximum)
		}
	})

	t.Run("allOf enum on choice", func(t *testing.T) {
		f := byName["choice"]
		if f == nil {
			t.Fatal("field choice missing")
			return
		}
		want := map[string]bool{"A": true, "B": true, "C": true, "D": true}
		if len(f.Enum) != len(want) {
			t.Fatalf("len(Enum) = %d, want %d", len(f.Enum), len(want))
		}
		for _, v := range f.Enum {
			if !want[v] {
				t.Errorf("unexpected enum value %q", v)
			}
		}
	})

	t.Run("direct $ref enum on choice_direct", func(t *testing.T) {
		f := byName["choice_direct"]
		if f == nil {
			t.Fatal("field choice_direct missing")
			return
		}
		want := map[string]bool{"A": true, "B": true, "C": true, "D": true}
		if len(f.Enum) != len(want) {
			t.Fatalf("len(Enum) = %d, want %d", len(f.Enum), len(want))
		}
		for _, v := range f.Enum {
			if !want[v] {
				t.Errorf("unexpected enum value %q", v)
			}
		}
	})

	t.Run("oneOf enum on domain (nullable)", func(t *testing.T) {
		f := byName["domain"]
		if f == nil {
			t.Fatal("field domain missing")
			return
		}
		want := map[string]bool{"A": true, "B": true, "C": true, "D": true, "": true}
		if len(f.Enum) != len(want) {
			t.Fatalf("len(Enum) = %d, want %d", len(f.Enum), len(want))
		}
		for _, v := range f.Enum {
			if !want[v] {
				t.Errorf("unexpected enum value %q", v)
			}
		}
	})

	t.Run("readOnly state has no validation constraints", func(t *testing.T) {
		f := byName["state"]
		if f == nil {
			t.Fatal("field state missing")
			return
		}
		if !f.Computed || f.Writable {
			t.Errorf("state: Computed=%v Writable=%v, want true/false", f.Computed, f.Writable)
		}
	})
}
