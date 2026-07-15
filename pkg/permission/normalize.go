package permission

import (
	"regexp"
	"strings"
)

var multiSpace = regexp.MustCompile(`\s+`)

// NormalizeCommand collapses whitespace and lowercases a shell command for policy matching.
func NormalizeCommand(cmd string) string {
	return strings.ToLower(multiSpace.ReplaceAllString(strings.TrimSpace(cmd), " "))
}
