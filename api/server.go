package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/benzjeremy/ollama-fleet-manager/controller"
	"github.com/benzjeremy/ollama-fleet-manager/engine"
	"github.com/benzjeremy/ollama-fleet-manager/hardware"
	"github.com/benzjeremy/ollama-fleet-manager/models"
	"github.com/benzjeremy/ollama-fleet-manager/storage"
)

// Server coordinates HTTP API routing and security guards.
type Server struct {
	port       int
	apiToken   string
	monitor    *hardware.Monitor
	registry   *models.Registry
	engine     *engine.RuleEngine
	client     *controller.OllamaClient
	store      *storage.Store
	startTime  time.Time
	httpServer *http.Server
}

// Config defines initialization parameters for Server.
type ServerConfig struct {
	Port       int
	APIToken   string
	Monitor    *hardware.Monitor
	Registry   *models.Registry
	Engine     *engine.RuleEngine
	Client     *controller.OllamaClient
	Store      *storage.Store
}

// NewServer builds an API server instance.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		port:      cfg.Port,
		apiToken:  cfg.APIToken,
		monitor:   cfg.Monitor,
		registry:  cfg.Registry,
		engine:    cfg.Engine,
		client:    cfg.Client,
		store:     cfg.Store,
		startTime: time.Now().UTC(),
	}
}

// Handler returns the secured http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Public health check
	mux.HandleFunc("/health", s.handleHealth)

	// Protected endpoints
	mux.HandleFunc("/api/v1/metrics", s.secure(s.handleMetrics))
	mux.HandleFunc("/api/v1/models", s.secure(s.handleModels))
	mux.HandleFunc("/api/v1/models/active", s.secure(s.handleActiveModels))
	mux.HandleFunc("/api/v1/models/select", s.secure(s.handleSelectModel))
	mux.HandleFunc("/api/v1/models/evict", s.secure(s.handleEvictModel))
	mux.HandleFunc("/api/v1/events", s.secure(s.handleEvents))
	mux.HandleFunc("/api/v1/config", s.secure(s.handleConfig))

	return s.applySecurityHeaders(mux)
}

// Start listens strictly on 127.0.0.1.
func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown cleanly stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Security Middleware: Anti-DNS-Rebinding, Anti-CSRF, Token Authentication
func (s *Server) secure(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Anti-DNS-Rebinding: Validate Host header
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "Forbidden: Anti-DNS-Rebinding violation", http.StatusForbidden)
			return
		}

		// 2. Anti-CSRF: Validate Origin header if present
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !strings.HasPrefix(origin, "http://127.0.0.1") && !strings.HasPrefix(origin, "http://localhost") {
				http.Error(w, "Forbidden: Cross-Origin Request Blocked", http.StatusForbidden)
				return
			}
		}

		// 3. Token Authentication
		token := r.Header.Get("X-API-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.apiToken)) != 1 {
			http.Error(w, "Unauthorized: Invalid or missing API token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// applySecurityHeaders injects OWASP hardening headers
func (s *Server) applySecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ollamaStatus := "online"
	ver, err := s.client.Ping(r.Context())
	if err != nil {
		ollamaStatus = "offline (" + err.Error() + ")"
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"service":        "ollama-fleet-manager",
		"version":        "v1.0",
		"uptime_seconds": time.Since(s.startTime).Seconds(),
		"ollama_status":  ollamaStatus,
		"ollama_version": ver,
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snap := s.monitor.Snapshot()
	writeJSON(w, http.StatusOK, snap)
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	snap := s.monitor.Snapshot()
	allModels := s.registry.All()

	type ModelStatus struct {
		models.ModelSpec
		FitsGPU bool `json:"fits_gpu"`
		FitsRAM bool `json:"fits_ram"`
	}

	result := make([]ModelStatus, 0, len(allModels))
	for _, m := range allModels {
		fitsGPU := snap.HasGPU && snap.VRAMFreeMB >= (m.MinVRAMMB+1024)
		fitsRAM := snap.RAMFreeMB >= (m.MinRAMMB + 2048)
		result = append(result, ModelStatus{
			ModelSpec: m,
			FitsGPU:   fitsGPU,
			FitsRAM:   fitsRAM,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"models":            result,
		"resource_snapshot": snap,
	})
}

func (s *Server) handleActiveModels(w http.ResponseWriter, r *http.Request) {
	modelsList, err := s.client.ListRunningModels(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Failed to contact Ollama daemon: " + err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active_models": modelsList,
		"count":         len(modelsList),
	})
}

type SelectRequest struct {
	Category string `json:"category"`
	Purpose  string `json:"purpose"`
}

func (s *Server) handleSelectModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SelectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Category = "general"
	}

	cat := models.TaskCategory(strings.ToLower(req.Category))
	snap := s.monitor.Snapshot()

	decision, err := s.engine.Evaluate(cat, snap)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	s.store.RecordEvent(storage.AuditEvent{
		Action:  "MODEL_SELECT",
		ModelID: decision.SelectedModel.ID,
		Reason:  decision.Reason,
		Mode:    decision.ExecutionMode,
	})

	writeJSON(w, http.StatusOK, decision)
}

type EvictRequest struct {
	ModelID string `json:"model_id"`
}

func (s *Server) handleEvictModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EvictRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.ModelID == "" {
		// Evict all loaded models
		active, err := s.client.ListRunningModels(r.Context())
		if err == nil {
			for _, m := range active {
				_ = s.client.UnloadModel(r.Context(), m.Name)
			}
		}
	} else {
		if err := s.client.UnloadModel(r.Context(), req.ModelID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Failed to unload model: " + err.Error(),
			})
			return
		}
	}

	s.store.RecordEvent(storage.AuditEvent{
		Action:  "MODEL_EVICT",
		ModelID: req.ModelID,
		Reason:  "Manual or automated eviction to reclaim system memory",
		Mode:    "UNLOAD",
	})

	writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"message":  "Eviction command executed",
		"model_id": req.ModelID,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	events := s.store.GetEvents(50)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var cfg storage.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if err := s.store.UpdateConfig(cfg); err != nil {
			http.Error(w, "Failed to persist config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}

	cfg := s.store.GetConfig()
	writeJSON(w, http.StatusOK, cfg)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
