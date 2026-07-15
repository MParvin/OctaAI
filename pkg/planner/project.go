package planner

import (
	"regexp"
	"strings"
)

var projectNamePattern = regexp.MustCompile(`[^a-z0-9_-]+`)

// SanitizeProjectName normalizes a user- or LLM-provided directory name.
func SanitizeProjectName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = projectNamePattern.ReplaceAllString(name, "")
	name = strings.Trim(name, "-_")
	return name
}

func goalNeedsProject(goalDescription, explicitName string) bool {
	if explicitName != "" {
		return true
	}
	lower := strings.ToLower(goalDescription)
	keywords := []string{
		"script", "application", "program", "project", "service",
		"tool", "cli", "app ", " app", "library", "package", "module",
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func resolveProjectName(goalDescription, explicitName, llmName string, needsProject bool) (string, bool) {
	if explicitName != "" {
		name := SanitizeProjectName(explicitName)
		if name != "" {
			return name, true
		}
	}
	if !needsProject {
		return "", false
	}
	if llmName != "" {
		return llmName, true
	}
	return "", false
}
