package spec

import (
	"strings"
	"unicode"
)

// toSnakeCase converts a camelCase or PascalCase string to snake_case.
// "photoUrls" -> "photo_urls", "petId" -> "pet_id", "APIKey" -> "api_key".
// Already-snake strings pass through unchanged.
func toSnakeCase(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				// Standard camelCase boundary (lower->upper): "petId" -> "pet_Id"
				// Acronym end (upper+upper->lower): "APIKey" -> "API_Key"
				if unicode.IsLower(prev) ||
					(unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
					b.WriteByte('_')
				}
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
