package webapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/vasis/singugen/internal/kanban"
	"github.com/vasis/singugen/internal/memory"
	"github.com/vasis/singugen/internal/spawner"
)

// Config configures the WebApp server.
type Config struct {
	Port int
}

// Server provides HTTP API for the Telegram WebApp.
type Server struct {
	httpServer *http.Server
	pool       *spawner.Pool
	board      *kanban.Board
	botToken   string
	allowFrom  map[int64]bool
	sessions   sync.Map // token → UserInfo
	logger     *slog.Logger
	addrCh     chan string // signals actual listen address after Start
}

// NewServer creates a WebApp server.
func NewServer(cfg Config, pool *spawner.Pool, board *kanban.Board, botToken string, allowFrom map[int64]bool, logger *slog.Logger) *Server {
	s := &Server{
		pool:      pool,
		board:     board,
		botToken:  botToken,
		allowFrom: allowFrom,
		logger:    logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth", s.handleAuth)
	mux.HandleFunc("GET /api/kanban", s.withAuth(s.handleKanbanList))
	mux.HandleFunc("POST /api/kanban", s.withAuth(s.handleKanbanCreate))
	mux.HandleFunc("POST /api/kanban/{id}/move", s.withAuth(s.handleKanbanMove))
	mux.HandleFunc("DELETE /api/kanban/{id}", s.withAuth(s.handleKanbanDelete))
	mux.HandleFunc("GET /api/memory/{agent}", s.withAuth(s.handleMemoryList))
	mux.HandleFunc("GET /api/memory/{agent}/{name}", s.withAuth(s.handleMemoryGet))
	mux.HandleFunc("PUT /api/memory/{agent}/{name}", s.withAuth(s.handleMemorySave))
	mux.HandleFunc("GET /api/agents", s.withAuth(s.handleAgentsList))

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}
	s.addrCh = make(chan string, 1)

	return s
}

// Start begins listening. Blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("webapp: listen: %w", err)
	}

	addr := ln.Addr().String()
	s.addrCh <- addr

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutdownCtx)
	}()

	s.logger.Info("webapp server started", "addr", addr)
	if err := s.httpServer.Serve(ln); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// --- Auth ---

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	var body struct {
		InitData string `json:"init_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := ValidateInitData(body.InitData, s.botToken)
	if err != nil {
		jsonError(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	if len(s.allowFrom) > 0 && !s.allowFrom[user.ID] {
		jsonError(w, "user not authorized", http.StatusForbidden)
		return
	}

	token := generateToken()
	s.sessions.Store(token, user)

	jsonOK(w, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			jsonError(w, "missing authorization", http.StatusUnauthorized)
			return
		}

		if _, ok := s.sessions.Load(token); !ok {
			jsonError(w, "invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// --- Kanban ---

func (s *Server) handleKanbanList(w http.ResponseWriter, _ *http.Request) {
	all, err := s.board.ListAll()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, all)
}

func (s *Server) handleKanbanCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Assignee    string `json:"assignee"`
		Priority    string `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	task, err := s.board.Add(body.Title, body.Description, body.Assignee, body.Priority)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, task)
}

func (s *Server) handleKanbanMove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Column string `json:"column"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.board.Move(id, body.Column); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]string{"status": "moved"})
}

func (s *Server) handleKanbanDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.board.Delete(id); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]string{"status": "deleted"})
}

// --- Memory ---

func (s *Server) handleMemoryList(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("agent")
	store := s.getMemory(agentName)
	if store == nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}
	entries, err := store.LoadAll()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, entries)
}

func (s *Server) handleMemoryGet(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("agent")
	name := r.PathValue("name")
	store := s.getMemory(agentName)
	if store == nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}
	entry, err := store.Load(name)
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, entry)
}

func (s *Server) handleMemorySave(w http.ResponseWriter, r *http.Request) {
	agentName := r.PathValue("agent")
	name := r.PathValue("name")
	store := s.getMemory(agentName)
	if store == nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := store.Save(name, body.Content); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "saved"})
}

func (s *Server) getMemory(agentName string) *memory.Store {
	if s.pool == nil {
		return nil
	}
	store, ok := s.pool.GetMemory(agentName)
	if !ok {
		return nil
	}
	return store
}

// --- Agents ---

func (s *Server) handleAgentsList(w http.ResponseWriter, _ *http.Request) {
	configs := s.pool.List()

	type agentInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		State       string `json:"state"`
	}

	var result []agentInfo
	for _, cfg := range configs {
		state := "unknown"
		if a, ok := s.pool.Get(cfg.Name); ok {
			state = a.State().String()
		}
		result = append(result, agentInfo{
			Name:        cfg.Name,
			Description: cfg.Description,
			State:       state,
		})
	}

	jsonOK(w, result)
}

// --- Helpers ---

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
