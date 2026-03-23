package spawner

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/claude"
	"github.com/vasis/singugen/internal/comms"
	"github.com/vasis/singugen/internal/memory"
)

// AgentConfig describes an agent to spawn.
type AgentConfig struct {
	Name        string
	Description string
	Model       string
}

// AgentEntry holds a running agent and its dependencies.
type AgentEntry struct {
	Agent   *agent.Agent
	Session *claude.Session
	Memory  *memory.Store
	Cancel  context.CancelFunc
	Config  AgentConfig
}

// Pool manages multiple named agents.
type Pool struct {
	agents      map[string]*AgentEntry
	bus         *comms.Bus
	launcher    claude.ProcessLauncher
	defaultName string
	baseDir     string
	mu          sync.RWMutex
	logger      *slog.Logger
}

// NewPool creates an agent pool.
func NewPool(launcher claude.ProcessLauncher, bus *comms.Bus, baseDir, defaultName string, logger *slog.Logger) *Pool {
	return &Pool{
		agents:      make(map[string]*AgentEntry),
		bus:         bus,
		launcher:    launcher,
		defaultName: defaultName,
		baseDir:     baseDir,
		logger:      logger,
	}
}

// Spawn creates and starts a new agent.
func (p *Pool) Spawn(ctx context.Context, cfg AgentConfig) error {
	p.mu.Lock()
	if _, exists := p.agents[cfg.Name]; exists {
		p.mu.Unlock()
		return fmt.Errorf("spawner: agent %q already exists", cfg.Name)
	}
	p.mu.Unlock()

	memDir := fmt.Sprintf("%s/%s/memory", p.baseDir, cfg.Name)
	memStore := memory.New(memDir, p.logger)
	if err := memStore.Init(); err != nil {
		return fmt.Errorf("spawner: init memory for %s: %w", cfg.Name, err)
	}

	prompt, _ := memStore.FormatForPrompt()

	sess := claude.NewSession(claude.SessionConfig{
		Model:        cfg.Model,
		SystemPrompt: prompt,
	}, p.launcher, p.logger)

	if err := sess.Start(ctx); err != nil {
		return fmt.Errorf("spawner: start session for %s: %w", cfg.Name, err)
	}

	childCtx, cancel := context.WithCancel(ctx)

	a := agent.New(agent.Config{
		Name:      cfg.Name,
		QueueSize: 64,
	}, sess, p.logger)

	p.bus.Subscribe(cfg.Name)

	entry := &AgentEntry{
		Agent:   a,
		Session: sess,
		Memory:  memStore,
		Cancel:  cancel,
		Config:  cfg,
	}

	p.mu.Lock()
	p.agents[cfg.Name] = entry
	p.mu.Unlock()

	go func() {
		if err := a.Run(childCtx); err != nil {
			p.logger.Error("agent exited", "name", cfg.Name, "error", err)
		}
	}()

	p.logger.Info("agent spawned", "name", cfg.Name, "description", cfg.Description)
	return nil
}

// Get returns an agent by name.
func (p *Pool) Get(name string) (*agent.Agent, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	entry, ok := p.agents[name]
	if !ok {
		return nil, false
	}
	return entry.Agent, true
}

// Stop stops and removes an agent.
func (p *Pool) Stop(name string) error {
	p.mu.Lock()
	entry, ok := p.agents[name]
	if !ok {
		p.mu.Unlock()
		return fmt.Errorf("spawner: agent %q not found", name)
	}
	delete(p.agents, name)
	p.mu.Unlock()

	p.bus.Unsubscribe(name)
	entry.Cancel()
	entry.Session.Close()

	p.logger.Info("agent stopped", "name", name)
	return nil
}

// List returns configs of all running agents.
func (p *Pool) List() []AgentConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	configs := make([]AgentConfig, 0, len(p.agents))
	for _, entry := range p.agents {
		configs = append(configs, entry.Config)
	}
	return configs
}

// SubmitTo submits a request to a named agent.
func (p *Pool) SubmitTo(name string, req agent.Request) error {
	a, ok := p.Get(name)
	if !ok {
		return fmt.Errorf("spawner: agent %q not found", name)
	}
	return a.Submit(req)
}

// Default returns the default agent.
func (p *Pool) Default() (*agent.Agent, bool) {
	return p.Get(p.defaultName)
}

// DefaultName returns the name of the default agent.
func (p *Pool) DefaultName() string {
	return p.defaultName
}

// ShutdownAll stops all agents gracefully.
func (p *Pool) ShutdownAll() {
	p.mu.Lock()
	names := make([]string, 0, len(p.agents))
	for name := range p.agents {
		names = append(names, name)
	}
	p.mu.Unlock()

	for _, name := range names {
		p.Stop(name)
	}
	p.logger.Info("all agents stopped")
}
