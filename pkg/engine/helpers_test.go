package engine

import "testing"

func TestParseToolCall(t *testing.T) {
	tc, err := parseToolCall(`{"tool":"filesystem","args":{"action":"read_file","path":"a.txt"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if tc.ToolName != "filesystem" {
		t.Fatalf("tool=%s", tc.ToolName)
	}
	if tc.Args["path"] != "a.txt" {
		t.Fatalf("args=%v", tc.Args)
	}

	tc, err = parseToolCall("```json\n{\"tool_name\":\"command\",\"arguments\":{\"command\":\"ls\",\"cwd\":\".\"}}\n```")
	if err != nil {
		t.Fatal(err)
	}
	if tc.ToolName != "command" {
		t.Fatalf("tool=%s", tc.ToolName)
	}

	if _, err := parseToolCall("no json here"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := parseToolCall(`{"args":{}}`); err == nil {
		t.Fatal("expected missing tool error")
	}
}
