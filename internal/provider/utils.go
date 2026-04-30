package provider

import "encoding/json"

// codeIn reports whether code is in the codes slice.
func codeIn(code int, codes []int) bool {
	for _, c := range codes {
		if code == c {
			return true
		}
	}
	return false
}

// sensitiveJSONKeys lists JSON field names whose values must be redacted in logs.
// Copied from aria provider internal/provider/utils_client_core.go, extended with
// common secret field names from the spec package's sensitiveKeywords list.
var sensitiveJSONKeys = map[string]bool{
	"password":          true,
	"passwd":            true,
	"secret":            true,
	"client_secret":     true,
	"clientSecret":      true,
	"private_key":       true,
	"privateKey":        true,
	"api_key":           true,
	"apiKey":            true,
	"token":             true,
	"access_token":      true,
	"accessToken":       true,
	"refreshToken":      true,
	"credential":        true,
	"credentials":       true,
	"systemCredentials": true,
	"passphrase":        true,
}

// redactSensitiveKeys walks a JSON structure and replaces sensitive values with "<REDACTED>".
// Copied verbatim from aria provider internal/provider/utils_client_core.go redactSensitiveKeys().
func redactSensitiveKeys(data any) any {
	switch v := data.(type) {
	case map[string]any:
		for key, val := range v {
			if sensitiveJSONKeys[key] {
				v[key] = "<REDACTED>"
			} else {
				v[key] = redactSensitiveKeys(val)
			}
		}
		return v
	case []any:
		for i, val := range v {
			v[i] = redactSensitiveKeys(val)
		}
		return v
	default:
		return data
	}
}

// redactJSON unmarshals raw JSON, redacts sensitive keys, and re-marshals.
// Copied verbatim from aria provider internal/provider/utils_client_core.go redactJSON().
func redactJSON(raw []byte) []byte {
	var data any
	if json.Unmarshal(raw, &data) != nil {
		return raw
	}
	redactSensitiveKeys(data)
	result, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return raw
	}
	return result
}
