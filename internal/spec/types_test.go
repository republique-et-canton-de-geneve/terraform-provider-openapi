package spec

import "testing"

// --- ResolvedItemPath ----------------------------------------------------------------------------

func TestResolvedItemPath(t *testing.T) {
	tests := []struct {
		name        string
		itemPath    string
		idPathParam string
		id          string
		want        string
	}{
		{
			"standard single-param path",
			"/api/v1/widgets/{id}/", "id", "42",
			"/api/v1/widgets/42/",
		},
		{
			"no IDPathParam returns ItemPath unchanged",
			"/api/v1/widgets/", "", "42",
			"/api/v1/widgets/",
		},
		{
			"UUID id",
			"/api/v1/items/{id}/", "id", "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			"/api/v1/items/a1b2c3d4-e5f6-7890-abcd-ef1234567890/",
		},
		{
			"non-standard param name",
			"/api/v1/projects/{project_id}/", "project_id", "99",
			"/api/v1/projects/99/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &ResourceSpec{ItemPath: tt.itemPath, IDPathParam: tt.idPathParam}
			if got := rs.ResolvedItemPath(tt.id); got != tt.want {
				t.Errorf("ResolvedItemPath(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
