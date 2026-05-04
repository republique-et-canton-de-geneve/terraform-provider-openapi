package spec

import "testing"

// --- normalizePath -------------------------------------------------------------------------------

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/foo", "/foo/"},
		{"/foo/", "/foo/"},
		{"/api/v1/widgets/{id}", "/api/v1/widgets/{id}/"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizePath(tt.in); got != tt.want {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- commonStrPrefix -----------------------------------------------------------------------------

func TestCommonStrPrefix(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		{"/api/v1/widgets", "/api/v1/things", "/api/v1/"},
		{"/api/v1/foo", "/api/v2/bar", "/api/v"},
		{"abc", "xyz", ""},
		{"", "abc", ""},
		{"abc", "abc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			if got := commonStrPrefix(tt.a, tt.b); got != tt.want {
				t.Errorf("commonStrPrefix(%q, %q) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// --- splitResourcePath ---------------------------------------------------------------------------

func TestSplitResourcePath(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		name   string
		hasID  bool
	}{
		{"/api/v1/widgets/", "/api/v1/", "widgets", false},
		{"/api/v1/widgets/{id}/", "/api/v1/", "widgets", true},
		{"/api/v1/network/vlans/{id}/", "/api/v1/", "network_vlans", true},
		{"/api/v1/", "/api/v1/", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotName, gotHasID := splitResourcePath(tt.path, tt.prefix)
			if gotName != tt.name || gotHasID != tt.hasID {
				t.Errorf("splitResourcePath(%q, %q) = (%q, %v), want (%q, %v)",
					tt.path, tt.prefix, gotName, gotHasID, tt.name, tt.hasID)
			}
		})
	}
}

// --- findCommonPathPrefix ------------------------------------------------------------------------

func TestFindCommonPathPrefix(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{
			"single path",
			[]string{"/api/v1/widgets/"},
			"",
		},
		{
			"two resources under shared prefix",
			[]string{
				"/api/v1/widgets/",
				"/api/v1/widgets/{id}/",
				"/api/v1/things/",
				"/api/v1/things/{id}/",
			},
			"/api/v1/",
		},
		{
			"root-level paths have no strippable prefix",
			[]string{"/vlans/", "/vlans/{id}/"},
			"",
		},
		{
			"completely different paths",
			[]string{"/api/v1/foo/", "/v2/bar/"},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findCommonPathPrefix(tt.paths); got != tt.want {
				t.Errorf("findCommonPathPrefix(%v) = %q, want %q", tt.paths, got, tt.want)
			}
		})
	}
}
