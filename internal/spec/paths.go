package spec

import (
	"regexp"
	"strings"
)

var pathParamRE = regexp.MustCompile(`\{([^}]+)\}`)

// normalizePath ensures a path ends with a trailing slash.
func normalizePath(p string) string {
	if !strings.HasSuffix(p, "/") {
		return p + "/"
	}
	return p
}

// splitResourcePath strips the common prefix, then returns the underscore-joined non-param segments
// as the resource name and whether a path parameter follows.
func splitResourcePath(path, prefix string) (string, bool) {
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return "", false
	}
	var parts []string
	hasID := false
	for _, p := range strings.Split(trimmed, "/") {
		if pathParamRE.MatchString(p) {
			hasID = true
			break
		}
		parts = append(parts, p)
	}
	return strings.Join(parts, "_"), hasID
}

// findCommonPathPrefix returns the longest slash-delimited prefix shared by all paths,
// provided stripping it leaves every path with at least one non-param segment.
func findCommonPathPrefix(paths []string) string {
	if len(paths) < 2 {
		return ""
	}
	common := paths[0]
	for _, p := range paths[1:] {
		common = commonStrPrefix(common, p)
		if common == "" {
			return ""
		}
	}
	for {
		i := strings.LastIndex(common, "/")
		if i <= 0 {
			return ""
		}
		candidate := common[:i+1]
		if isValidPathPrefix(candidate, paths) {
			return candidate
		}
		common = common[:i]
	}
}

// isValidPathPrefix returns true if stripping prefix leaves every path with a non-empty,
// non-param first segment.
func isValidPathPrefix(prefix string, paths []string) bool {
	for _, p := range paths {
		rest := strings.Trim(strings.TrimPrefix(p, prefix), "/")
		if rest == "" {
			return false
		}
		firstSeg := strings.SplitN(rest, "/", 2)[0]
		if pathParamRE.MatchString(firstSeg) {
			return false
		}
	}
	return true
}

// commonStrPrefix returns the longest common leading substring of a and b.
func commonStrPrefix(a, b string) string {
	n := min(len(a), len(b))
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}
