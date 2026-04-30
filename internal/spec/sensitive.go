package spec

import (
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
)

// sensitiveKeywords are matched against the lowercase field name. A field is
// considered sensitive if its name contains any of these substrings, or if the
// spec explicitly sets x-sensitive: true.
var sensitiveKeywords = []string{
	"password",
	"passwd",
	"secret",
	"private_key",
	"privatekey",
	"api_key",
	"apikey",
	"token",
	"credential",
	"passphrase",
}

// isSensitiveField returns true if the field should be treated as sensitive.
// x-sensitive: true forces sensitive regardless of the field name.
// x-sensitive: false suppresses the name-heuristic (opt-out).
// When absent, the lowercase field name is checked against sensitiveKeywords.
func isSensitiveField(name string, schema *base.Schema) bool {
	if schema != nil && schema.Extensions != nil {
		if node, ok := schema.Extensions.Get("x-sensitive"); ok && node != nil {
			if node.Value == "true" {
				return true
			}
			if node.Value == "false" {
				return false
			}
		}
	}
	lower := strings.ToLower(name)
	for _, kw := range sensitiveKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
