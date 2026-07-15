package permission

import "strings"

// ShellQuote wraps a string in single quotes for safe remote shell use.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
