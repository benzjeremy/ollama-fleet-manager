package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEvent records model transitions and actions.
type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	ModelID   string    `json:"model_id"`
	Reason    string    `json:"reason"`
	Mode      string    `json:"mode"`
}

// Config holds persistent service settings.
type Config struct {
	OllamaHost   string `json:"ollama_host"`
	PollInterval int    `json:"poll_interval_sec"`
	VRAMBufferMB uint64 `json:"vram_buffer_mb"`
	RAMBufferMB  uint64 `json:"ram_buffer_mb"`
	MaxModels    int    `json:"max_concurrent_models"`
}

// Store handles encrypted state persistence.
type Store struct {
	mu         sync.RWMutex
	filePath   string
	passphrase string
	config     Config
	events     []AuditEvent
}

// NewStore initializes an encrypted storage engine.
func NewStore(dataDir, passphrase string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	s := &Store{
		filePath:   filepath.Join(dataDir, "fleet_vault.enc"),
		passphrase: passphrase,
		config: Config{
			OllamaHost:   "http://127.0.0.1:11434",
			PollInterval: 30,
			VRAMBufferMB: 1024,
			RAMBufferMB:  2048,
			MaxModels:    1,
		},
		events: make([]AuditEvent, 0),
	}

	// If file exists, load and decrypt
	if _, err := os.Stat(s.filePath); err == nil {
		if err := s.load(); err != nil {
			// On decryption failure or corruption, fallback to clean state
		}
	} else {
		_ = s.save()
	}

	return s, nil
}

// RecordEvent appends an audit event and saves.
func (s *Store) RecordEvent(event AuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event.Timestamp = time.Now().UTC()
	s.events = append(s.events, event)
	if len(s.events) > 200 {
		s.events = s.events[len(s.events)-200:]
	}
	_ = s.saveLocked()
}

// GetEvents returns recent audit events.
func (s *Store) GetEvents(limit int) []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	start := len(s.events) - limit
	result := make([]AuditEvent, limit)
	copy(result, s.events[start:])
	return result
}

// GetConfig returns a copy of current config.
func (s *Store) GetConfig() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// UpdateConfig updates service configuration.
func (s *Store) UpdateConfig(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
	return s.saveLocked()
}

type storeData struct {
	Config Config       `json:"config"`
	Events []AuditEvent `json:"events"`
}

func (s *Store) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	data := storeData{
		Config: s.config,
		Events: s.events,
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	encrypted, err := Encrypt(raw, s.passphrase)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, encrypted, 0600)
}

func (s *Store) load() error {
	encrypted, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	decrypted, err := Decrypt(encrypted, s.passphrase)
	if err != nil {
		return err
	}

	var data storeData
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return err
	}

	s.config = data.Config
	s.events = data.Events
	return nil
}
