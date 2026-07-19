package engine

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/llm"
	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

type featureStubLLM struct{}

func (s *featureStubLLM) Name() string  { return "stub" }
func (s *featureStubLLM) Model() string { return "stub-model" }
func (s *featureStubLLM) Complete(ctx context.Context, prompt string, opts *llm.Options) (*llm.Response, error) {
	return &llm.Response{
		Content: `{"tasks":[{"id":"task_1","description":"Setup","type":"sequential","complexity":2}]}`,
	}, nil
}

func TestNewEngineWiresHTNAndCapabilities(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Features.UseHTNPlanner = true
	cfg.Features.UseCapabilities = true
	cfg.Features.UseDAGExecutor = true
	cfg.Storage.Path = filepath.Join(t.TempDir(), "state.db")

	store, err := storage.NewSQLiteStorage(cfg.Storage.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	eng := NewEngine(cfg, &featureStubLLM{}, tools.NewRegistry(), store)
	if eng.Capabilities() == nil {
		t.Fatal("expected capability registry when flags enabled")
	}
	if len(eng.Capabilities().List()) == 0 {
		t.Fatal("expected builtin capabilities")
	}
	if !eng.useDAG {
		t.Fatal("expected useDAG true")
	}
	_ = llm.Provider(&featureStubLLM{})
}

func TestNewEngineDefaultsNoCapabilities(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.Path = filepath.Join(t.TempDir(), "state.db")
	store, err := storage.NewSQLiteStorage(cfg.Storage.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	eng := NewEngine(cfg, &featureStubLLM{}, tools.NewRegistry(), store)
	if eng.Capabilities() != nil {
		t.Fatal("expected nil capabilities when flags off")
	}
	if eng.useDAG {
		t.Fatal("expected useDAG false by default")
	}
}
