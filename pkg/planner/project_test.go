package planner

import "testing"

func TestSanitizeProjectName(t *testing.T) {
	tests := map[string]string{
		"github-list":  "github-list",
		"GitHub List":  "github-list",
		"  my_app  ":   "my_app",
		"foo@bar!":     "foobar",
	}
	for input, want := range tests {
		if got := SanitizeProjectName(input); got != want {
			t.Errorf("SanitizeProjectName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGoalNeedsProject(t *testing.T) {
	if !goalNeedsProject("Develop a python script for GitHub", "") {
		t.Fatal("expected script goal to need a project")
	}
	if goalNeedsProject("fix typo in README", "") {
		t.Fatal("expected simple edit to skip project")
	}
	if !goalNeedsProject("anything", "my-app") {
		t.Fatal("explicit project name should require project")
	}
}

func TestResolveProjectNameExplicit(t *testing.T) {
	name, needs := resolveProjectName("write code", "GitHub-List", "", false)
	if !needs || name != "github-list" {
		t.Fatalf("resolveProjectName() = (%q, %v)", name, needs)
	}
}
