package workflow

import "errors"

// ErrIncompleteExecution is returned when not all nodes could be executed.
var ErrIncompleteExecution = errors.New("workflow execution incomplete: unresolved dependencies or cycle")
