package engine

import (
	"testing"
)

func TestViewWithOutputIncludesRuntimeResult(t *testing.T) {
	step := NewExecutionStep("step_1", "goal_1", "write file", StepInput{ToolName: "filesystem"})
	view := step.ViewWithOutput(&StepOutput{
		Success: true,
		Output:  "Wrote 20 bytes",
	})

	if !view.Success {
		t.Fatal("expected Success=true in view")
	}
	if view.Output != "Wrote 20 bytes" {
		t.Fatalf("expected output in view, got %q", view.Output)
	}

	emptyView := step.View()
	if emptyView.Success || emptyView.Output != "" {
		t.Fatal("View() without stored output should not report success")
	}
}
