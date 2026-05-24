package agent

import (
	"context"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/engine"
	"github.com/mparvin/octaai/pkg/llm"
	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

// Agent wraps the execution engine for backward compatibility.
type Agent struct {
	engine *engine.Engine
}

// NewAgent creates a new agent backed by the execution engine.
func NewAgent(cfg *config.Config, llmProvider llm.Provider, toolRegistry *tools.Registry, store storage.Storage) *Agent {
	return &Agent{
		engine: engine.NewEngine(cfg, llmProvider, toolRegistry, store),
	}
}

// ProcessGoal processes a goal from start to completion.
func (a *Agent) ProcessGoal(ctx context.Context, goalID string) error {
	return a.engine.ProcessGoal(ctx, goalID)
}

// Engine exposes the underlying execution engine for advanced operations.
func (a *Agent) Engine() *engine.Engine {
	return a.engine
}
