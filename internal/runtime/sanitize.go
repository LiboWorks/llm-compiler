package runtime

import (
	"regexp"
	"strings"
)

// SanitizeForShell prepares free-form text (like LLM output) to be
// safely placed inside a double-quoted shell argument. It performs a
// light sanitization: trims whitespace, collapses internal whitespace
// to single spaces, escapes double quotes, and removes NULs.
// This is intentionally conservative but avoids executing arbitrary
// multi-line commands when workflows embed LLM output into `sh -c`.
func SanitizeForShell(s string) string {
	if s == "" {
		return s
	}
	// remove NULs
	s = strings.ReplaceAll(s, "\x00", "")
	// collapse whitespace (space/newline/tab) to single spaces
	ws := regexp.MustCompile(`\s+`)
	s = ws.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	// escape double quotes so it can be inserted into a "..." string
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
