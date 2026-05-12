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
