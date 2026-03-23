package webapp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/vasis/singugen/internal/comms"
	"github.com/vasis/singugen/internal/kanban"
	"github.com/vasis/singugen/internal/spawner"
)

func testServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()

	board := kanban.NewBoard(dir+"/kanban", testLogger())
	board.Init()

	bus := comms.New()
	pool := spawner.NewPool(fakeLauncher{}, bus, dir+"/agents", "main", testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	pool.Spawn(ctx, spawner.AgentConfig{Name: "main", Description: "Main"})
	time.Sleep(50 * time.Millisecond)

	srv := NewServer(Config{Port: 0}, pool, board, "test-token", nil, testLogger())

	// Start on random port.
	go srv.Start(ctx)

	// Wait for server to be ready.
	for range 20 {
		time.Sleep(50 * time.Millisecond)
		addr := srv.httpServer.Addr
		if addr != "" && addr != ":0" {
			break
		}
	}

	t.Cleanup(func() {
		cancel()
		pool.ShutdownAll()
	})

	return srv, "http://" + srv.httpServer.Addr
}

func TestServer_AuthAndKanban(t *testing.T) {
	_, baseURL := testServer(t)

	// Create auth token by calling /api/auth with valid initData.
	params := map[string]string{
		"user":      `{"id":12345,"first_name":"Test","username":"test"}`,
		"auth_date": "1700000000",
	}
	initData := buildTestInitData("test-token", params)

	authBody, _ := json.Marshal(map[string]string{"init_data": initData})
	resp, err := http.Post(baseURL+"/api/auth", "application/json", bytes.NewReader(authBody))
	if err != nil {
		t.Fatalf("auth request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("auth status = %d, body = %s", resp.StatusCode, body)
	}

	var authResp struct {
		Token string   `json:"token"`
		User  UserInfo `json:"user"`
	}
	json.NewDecoder(resp.Body).Decode(&authResp)

	if authResp.Token == "" {
		t.Fatal("empty auth token")
	}

	// Use token to access kanban.
	req, _ := http.NewRequest("GET", baseURL+"/api/kanban", nil)
	req.Header.Set("Authorization", authResp.Token)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Errorf("kanban status = %d, want 200", resp2.StatusCode)
	}
}

func TestServer_Unauthorized(t *testing.T) {
	_, baseURL := testServer(t)

	// No auth token.
	resp, err := http.Get(baseURL + "/api/kanban")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestServer_Agents(t *testing.T) {
	_, baseURL := testServer(t)

	// Get auth.
	params := map[string]string{
		"user":      `{"id":1,"first_name":"T","username":"t"}`,
		"auth_date": "1700000000",
	}
	initData := buildTestInitData("test-token", params)
	authBody, _ := json.Marshal(map[string]string{"init_data": initData})
	resp, err := http.Post(baseURL+"/api/auth", "application/json", bytes.NewReader(authBody))
	if err != nil {
		t.Fatalf("auth request failed: %v", err)
	}
	var authResp struct{ Token string }
	json.NewDecoder(resp.Body).Decode(&authResp)
	resp.Body.Close()

	// List agents.
	req, _ := http.NewRequest("GET", baseURL+"/api/agents", nil)
	req.Header.Set("Authorization", authResp.Token)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("agents request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("agents status = %d, body = %s", resp2.StatusCode, body)
	}

	var agents []map[string]string
	json.NewDecoder(resp2.Body).Decode(&agents)

	if len(agents) == 0 {
		t.Error("no agents returned")
	}
}
