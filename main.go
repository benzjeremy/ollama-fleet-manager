package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/benzjeremy/ollama-fleet-manager/api"
	"github.com/benzjeremy/ollama-fleet-manager/controller"
	"github.com/benzjeremy/ollama-fleet-manager/engine"
	"github.com/benzjeremy/ollama-fleet-manager/hardware"
	"github.com/benzjeremy/ollama-fleet-manager/models"
	"github.com/benzjeremy/ollama-fleet-manager/storage"
)

const (
	Version = "v1.0"
	Banner  = `
  ___  _ _                        _____ _           _   ___  ___                                  
 / _ \| | |                      |  ___| |         | |  |  \/  |                                  
/ /_\ \ | | __ _ _ __ ___   __ _ | |_  | | ___  ___| |_ | .  . | __ _ _ __   __ _  __ _  ___ _ __ 
|  _  | | |/ _` + "`" + ` | '_ ` + "`" + ` _ \ / _` + "`" + ` ||  _| | |/ _ \/ _ \ __|| |\/| |/ _` + "`" + ` | '_ \ / _` + "`" + ` |/ _` + "`" + ` |/ _ \ '__|
| | | | | | (_| | | | | | | (_| || |   | |  __/  __/ |_ | |  | | (_| | | | | (_| | (_| |  __/ |   
\_| |_/_|_|\__,_|_| |_| |_|\__,_|\_|   |_|\___|\___|\__|\_|  |_/\__,_|_| |_|\__,_|\__, |\___|_|   
                                                                                   __/ |          
                                                                                  |___/           
`
)

func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "fleet_sec_token_fallback_" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func main() {
	port := flag.Int("port", 8080, "Port to listen on (binds strictly to 127.0.0.1)")
	apiToken := flag.String("token", "", "32-byte API token for authentication (auto-generated if empty)")
	ollamaURL := flag.String("ollama", "http://127.0.0.1:11434", "Ollama daemon API URL")
	pollInterval := flag.Int("poll-interval", 30, "Interval in seconds to collect hardware telemetry")
	dataDir := flag.String("data-dir", "", "Directory for encrypted state storage")
	passphrase := flag.String("passphrase", "FleetManagerDefaultMasterVault_2026!", "Passphrase for AES-256-GCM vault encryption")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ollama-fleet-manager %s (Zero-Dummy-Security, GNU GPL-3.0)\n", Version)
		os.Exit(0)
	}

	token := *apiToken
	if token == "" {
		token = generateSecureToken()
	}

	if *dataDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			*dataDir = filepath.Join(home, ".config", "ollama-fleet-manager")
		} else {
			*dataDir = "./data"
		}
	}

	// 1. Initialize Subsystems
	mon := hardware.NewMonitor()
	reg := models.NewRegistry()
	eng := engine.NewRuleEngine(reg)
	client := controller.NewOllamaClient(*ollamaURL)
	store, err := storage.NewStore(*dataDir, *passphrase)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize encrypted store: %v", err)
	}

	srv := api.NewServer(api.ServerConfig{
		Port:     *port,
		APIToken: token,
		Monitor:  mon,
		Registry: reg,
		Engine:   eng,
		Client:   client,
		Store:    store,
	})

	// Print Startup Diagnostics
	fmt.Print(Banner)
	fmt.Printf("🚀 Ollama Fleet Manager %s | Zero-Dummy-Security Edition\n", Version)
	fmt.Printf("🔒 License: GNU General Public License v3.0 (GPL-3.0)\n")
	fmt.Printf("🌐 Bound Address:    http://127.0.0.1:%d\n", *port)
	fmt.Printf("🔑 Access Token:     %s\n", token)
	fmt.Printf("🤖 Ollama Endpoint:  %s\n", *ollamaURL)
	fmt.Printf("💾 Encrypted Vault:  %s\n", *dataDir)
	fmt.Printf("🛡️ Guards:           Anti-DNS-Rebinding: ON | Anti-CSRF: ON | AES-256-GCM: ON\n")
	fmt.Println("--------------------------------------------------------------------------------")

	// Hardware Check
	snap := mon.Snapshot()
	fmt.Printf("📊 Initial Hardware: %s\n", snap.String())

	// 2. Background Hardware Telemetry Daemon
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(time.Duration(*pollInterval) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				currentSnap := mon.Snapshot()
				if currentSnap.IsConstrained() {
					log.Printf("[WARN] System memory constrained (RAM: %.1f%%, VRAM: %.1f%%). Monitoring for OOM.",
						currentSnap.RAMUsagePct, currentSnap.VRAMUsagePct)
				}
			}
		}
	}()

	// 3. Graceful Shutdown Listener
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("\n[INFO] Received termination signal. Initiating graceful shutdown...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("[ERROR] Server shutdown error: %v", err)
		}
		log.Println("[INFO] Ollama Fleet Manager stopped successfully.")
		os.Exit(0)
	}()

	// 4. Start HTTP Server
	if err := srv.Start(); err != nil && err != context.Canceled {
		log.Fatalf("[FATAL] Server terminated: %v", err)
	}
}
