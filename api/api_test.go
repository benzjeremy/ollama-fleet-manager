package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/benzjeremy/ollama-fleet-manager/controller"
	"github.com/benzjeremy/ollama-fleet-manager/engine"
	"github.com/benzjeremy/ollama-fleet-manager/hardware"
	"github.com/benzjeremy/ollama-fleet-manager/models"
	"github.com/benzjeremy/ollama-fleet-manager/storage"
)

func setupTestServer(t *testing.T) (*Server, string) {
	tmpDir := filepath.Join(os.TempDir(), "fleet_api_test")
	_ = os.RemoveAll(tmpDir)

	token := "valid_32_byte_token_for_tests_123456"
	mon := hardware.NewMonitor()
	reg := models.NewRegistry()
	eng := engine.NewRuleEngine(reg)
	client := controller.NewOllamaClient("http://127.0.0.1:11434")
	store, err := storage.NewStore(tmpDir, "TestPassphrase123!")
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	srv := NewServer(ServerConfig{
		Port:     8080,
		APIToken: token,
		Monitor:  mon,
		Registry: reg,
		Engine:   eng,
		Client:   client,
		Store:    store,
	})

	return srv, token
}

func TestHealthEndpoint(t *testing.T) {
	srv, _ := setupTestServer(t)
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected HTTP 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if resp["service"] != "ollama-fleet-manager" {
		t.Errorf("Expected service ollama-fleet-manager, got %v", resp["service"])
	}
}

func TestSecurityTokenRejection(t *testing.T) {
	srv, _ := setupTestServer(t)
	handler := srv.Handler()

	// Missing token
	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	req.Host = "127.0.0.1:8080"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected HTTP 401 on missing token, got %d", w.Code)
	}

	// Invalid token
	req = httptest.NewRequest("GET", "/api/v1/metrics", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("X-API-Token", "wrong_token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected HTTP 401 on wrong token, got %d", w.Code)
	}
}

func TestAntiDNSRebindingProtection(t *testing.T) {
	srv, token := setupTestServer(t)
	handler := srv.Handler()

	// Malicious host header (e.g. attacker.com rebinding to 127.0.0.1)
	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	req.Host = "attacker-domain.evil:8080"
	req.Header.Set("X-API-Token", token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected HTTP 403 on DNS rebinding attempt, got %d", w.Code)
	}
}

func TestAntiCSRFProtection(t *testing.T) {
	srv, token := setupTestServer(t)
	handler := srv.Handler()

	// External malicious website attempting cross-origin fetch
	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Origin", "https://malicious-site.com")
	req.Header.Set("X-API-Token", token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected HTTP 403 on CSRF cross-origin attempt, got %d", w.Code)
	}
}

func TestModelSelectionWorkflow(t *testing.T) {
	srv, token := setupTestServer(t)
	handler := srv.Handler()

	payload, _ := json.Marshal(map[string]string{
		"category": "coding",
		"purpose":  "High speed code refactor",
	})
	req := httptest.NewRequest("POST", "/api/v1/models/select", bytes.NewReader(payload))
	req.Host = "127.0.0.1:8080"
	req.Header.Set("X-API-Token", token)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %d (%s)", w.Code, w.Body.String())
	}

	var dec engine.Decision
	if err := json.Unmarshal(w.Body.Bytes(), &dec); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if dec.SelectedModel.ID == "" {
		t.Errorf("Expected non-empty model ID in decision")
	}
}
