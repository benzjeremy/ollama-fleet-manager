package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAES256GCMEncryption(t *testing.T) {
	passphrase := "SecureFleetKey_987654321!"
	plaintext := []byte("Zero-Dummy-Security: Production-level AES-256-GCM verification payload.")

	ciphertext, err := Encrypt(plaintext, passphrase)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if bytes.Equal(plaintext, ciphertext) {
		t.Fatalf("Ciphertext cannot be equal to plaintext")
	}

	decrypted, err := Decrypt(ciphertext, passphrase)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Decrypted text does not match original plaintext")
	}

	// Tampering test
	tampered := make([]byte, len(ciphertext))
	copy(tampered, ciphertext)
	tampered[len(tampered)-1] ^= 0xFF // Flip last byte of GCM auth tag

	_, err = Decrypt(tampered, passphrase)
	if err == nil {
		t.Fatalf("Expected decryption failure on tampered ciphertext, but got nil")
	}

	// Wrong password test
	_, err = Decrypt(ciphertext, "WrongPassphrase123")
	if err == nil {
		t.Fatalf("Expected decryption failure on wrong passphrase, but got nil")
	}
}

func TestEncryptedStorePersistence(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "fleet_test_store")
	defer os.RemoveAll(tmpDir)

	pass := "VaultMasterPassphrase42!"
	store, err := NewStore(tmpDir, pass)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	store.RecordEvent(AuditEvent{
		Action:  "DEPLOY",
		ModelID: "qwen2.5-coder:7b",
		Reason:  "Automated task dispatch",
		Mode:    "GPU",
	})

	cfg := store.GetConfig()
	cfg.PollInterval = 15
	if err := store.UpdateConfig(cfg); err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	// Reopen store and verify persistence
	reopened, err := NewStore(tmpDir, pass)
	if err != nil {
		t.Fatalf("Reopening store failed: %v", err)
	}

	if reopened.GetConfig().PollInterval != 15 {
		t.Errorf("Expected PollInterval=15, got %d", reopened.GetConfig().PollInterval)
	}

	events := reopened.GetEvents(10)
	if len(events) != 1 || events[0].ModelID != "qwen2.5-coder:7b" {
		t.Errorf("Audit events not properly persisted and reloaded")
	}
}
