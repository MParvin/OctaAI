package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mparvin/octaai/pkg/agent"
	"github.com/mparvin/octaai/pkg/browser"
	"github.com/mparvin/octaai/pkg/config"
	"github.com/mparvin/octaai/pkg/health"
	"github.com/mparvin/octaai/pkg/llm"
	"github.com/mparvin/octaai/pkg/plugin"
	"github.com/mparvin/octaai/pkg/storage"
	"github.com/mparvin/octaai/pkg/tools"
)

func isProcessableState(state storage.State) bool {
	switch state {
	case storage.StateIdle, storage.StatePlanning, storage.StateRetrying,
		storage.StateExecuting, storage.StateEvaluating:
		return true
	default:
		return false
	}
}

func main() {
	browserPort := flag.Int("browser-port", 0, "Override browser WebSocket port (default: from config, usually 8765)")
	healthAddr := flag.String("health-addr", "127.0.0.1:8766", "Daemon health/readiness listen address (empty to disable)")
	flag.Parse()

	fmt.Println("OctaAI Agent Daemon - Starting...")

	cfg, err := config.LoadConfig(config.ConfigPath())
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *browserPort > 0 {
		cfg.Browser.Port = *browserPort
		cfg.Browser.Enabled = true
	}

	fmt.Printf("Using LLM: %s (%s)\n", cfg.LLM.Provider, cfg.LLM.Model)
	fmt.Printf("Projects root: %s\n", cfg.ProjectsRoot)
	fmt.Printf("Storage: %s\n", cfg.Storage.Path)
	if cfg.Features.UseHTNPlanner || cfg.Features.UseDAGExecutor || cfg.Features.UseCapabilities {
		fmt.Printf("Features: htn=%v dag=%v capabilities=%v\n",
			cfg.Features.UseHTNPlanner, cfg.Features.UseDAGExecutor, cfg.Features.UseCapabilities)
	}

	llmProvider, err := llm.NewProvider(&cfg.LLM)
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}

	store, err := storage.NewSQLiteStorage(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	var browserServer *browser.Server
	if cfg.Browser.Enabled {
		if _, err := config.EnsureBrowserToken(cfg); err != nil {
			log.Fatalf("Failed to configure browser token: %v", err)
		}
		addr := fmt.Sprintf("localhost:%d", cfg.Browser.Port)
		browserServer = browser.NewServer(addr, cfg.Browser.Token)
		go func() {
			if err := browserServer.Start(); err != nil {
				log.Printf("Browser WebSocket server error: %v", err)
			}
		}()
		fmt.Printf("Browser automation enabled on port %d\n", cfg.Browser.Port)
	}

	pluginRegistry := plugin.NewRegistry()
	for _, p := range plugin.DefaultPlugins(browserServer) {
		pluginRegistry.Register(p)
	}

	toolRegistry := tools.NewRegistry()
	if err := pluginRegistry.LoadAll(toolRegistry, cfg); err != nil {
		log.Fatalf("Failed to load plugins: %v", err)
	}

	fmt.Printf("Registered %d tools via %d plugins\n", len(toolRegistry.List()), len(pluginRegistry.List()))

	ag := agent.NewAgent(cfg, llmProvider, toolRegistry, store)

	var healthServer *health.Server
	if *healthAddr != "" {
		healthServer = health.New(*healthAddr, func() map[string]interface{} {
			out := map[string]interface{}{
				"tools": len(toolRegistry.List()),
				"feature_flags": map[string]bool{
					"htn":          cfg.Features.UseHTNPlanner,
					"dag":          cfg.Features.UseDAGExecutor,
					"capabilities": cfg.Features.UseCapabilities,
				},
			}
			if browserServer != nil {
				out["browsers"] = len(browserServer.GetConnectedBrowsers())
			}
			if caps := ag.Engine().Capabilities(); caps != nil {
				out["capability_count"] = len(caps.List())
			}
			return out
		})
		go func() {
			if err := healthServer.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("Health server error: %v", err)
			}
		}()
		fmt.Printf("Health endpoints on http://%s/healthz and /readyz\n", *healthAddr)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Agent daemon is running. Waiting for goals...")
	fmt.Println("Press Ctrl+C to stop")

	if healthServer != nil {
		healthServer.SetReady(true)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var active sync.WaitGroup
	processing := make(map[string]struct{})
	var processingMu sync.Mutex

	processGoal := func(goalID string) {
		processingMu.Lock()
		if _, exists := processing[goalID]; exists {
			processingMu.Unlock()
			return
		}
		processing[goalID] = struct{}{}
		processingMu.Unlock()

		active.Add(1)
		go func() {
			defer active.Done()
			defer func() {
				processingMu.Lock()
				delete(processing, goalID)
				processingMu.Unlock()
			}()

			fmt.Printf("\n=== Processing Goal: %s ===\n", goalID)
			if err := ag.ProcessGoal(ctx, goalID); err != nil {
				if ctx.Err() != nil {
					log.Printf("Goal %s interrupted: %v", goalID, err)
					return
				}
				log.Printf("Error processing goal %s: %v", goalID, err)
			} else {
				fmt.Printf("=== Goal %s completed ===\n", goalID)
			}
		}()
	}

	pollGoals := func() {
		goals, err := store.ListGoals()
		if err != nil {
			log.Printf("Error listing goals: %v", err)
			return
		}
		for _, goal := range goals {
			if isProcessableState(goal.State) {
				processGoal(goal.ID)
			}
		}
	}

	pollGoals()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nShutting down...")
			if healthServer != nil {
				healthServer.SetReady(false)
			}
			cancel()
			done := make(chan struct{})
			go func() {
				active.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				log.Println("Shutdown timeout; exiting with goals still running")
			}
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			_ = pluginRegistry.Shutdown(shutdownCtx)
			if browserServer != nil {
				_ = browserServer.Stop(shutdownCtx)
			}
			if healthServer != nil {
				_ = healthServer.Shutdown(shutdownCtx)
			}
			return

		case <-ticker.C:
			pollGoals()
		}
	}
}
