package memory

import "testing"

func TestSemanticSearch(t *testing.T) {
	m := NewManager(nil)
	m.Remember("g1", "file_created", "Created app.py with Flask routes", "execution")
	m.Remember("g1", "test_run", "pytest failed on test_health endpoint", "execution")
	m.Remember("g1", "dependency", "Added flask to requirements.txt", "execution")

	results := m.Search("g1", "flask application routes", 2)
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	if results[0].Entry.Key != "file_created" {
		t.Fatalf("expected file_created as top result, got %s", results[0].Entry.Key)
	}
}
