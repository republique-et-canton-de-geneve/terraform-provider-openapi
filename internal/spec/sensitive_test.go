package spec

import "testing"

// --- isSensitiveField ----------------------------------------------------------------------------

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		// x-sensitive extension tested in discover_test.go (requires a Schema object).
		// Name-based heuristics:
		{"password field", "password", true},
		{"passwd variant", "db_passwd", true},
		{"secret field", "client_secret", true},
		{"private_key snake", "private_key", true},
		{"privatekey compact", "privatekey", true},
		{"api_key snake", "api_key", true},
		{"apikey compact", "apikey", true},
		{"access_token", "access_token", true},
		{"credential", "credential", true},
		{"credentials plural", "credentials", true},
		{"passphrase", "ssh_passphrase", true},
		{"mixed case Password", "Password", true},
		{"all caps SECRET_KEY", "SECRET_KEY", true},

		// Non-sensitive:
		{"name", "name", false},
		{"id", "id", false},
		{"description", "description", false},
		{"url", "url", false},
		{"created_at", "created_at", false},
		{"public_key", "public_key", false},

		// Name heuristic matches even when keyword is a substring.
		// Use x-sensitive: false to opt out.
		{"token_count matches token keyword", "token_count", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSensitiveField(tt.fieldName, nil)
			if got != tt.want {
				t.Errorf("isSensitiveField(%q) = %v, want %v", tt.fieldName, got, tt.want)
			}
		})
	}
}
