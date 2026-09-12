# 🚀 ollama-fleet-manager

[![CI](https://github.com/benzjeremy/ollama-fleet-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/benzjeremy/ollama-fleet-manager/actions)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://github.com/benzjeremy/ollama-fleet-manager/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Release](https://img.shields.io/badge/Release-v1.0-f43f5e.svg)](https://github.com/benzjeremy/ollama-fleet-manager/releases/tag/v1.0)

Intelligent, resource-efficient fleet manager and dynamic model router for [Ollama](https://ollama.com). Built in pure Go with hardware telemetry sensors (CPU, RAM, GPU VRAM), automated OOM prevention, and uncompromising **Zero-Dummy-Security**.

---

## 💡 Why ollama-fleet-manager?

Monolithic local AI deployments frequently run into unpredictable memory bottlenecks:
1. **OOM Crashes (Out of Memory):** When large models like `llama3.1:8b` or `deepseek-r1:8b` are loaded concurrently, the Linux kernel invokes the `OOM-Killer`.
2. **Resource Waste:** Simple tasks (summaries, proofreading) do not require 8B parameters and unnecessarily consume precious GPU VRAM.
3. **Insecure Local Endpoints:** Many local AI servers bind exposed to `0.0.0.0` or lack basic CSRF and DNS-rebinding protections.

**ollama-fleet-manager** resolves these issues through continuous hardware telemetry, a rules-based model selection engine, and cryptographically secured local APIs.

---

## 🛡️ Zero-Dummy-Security (Production-Grade Defaults)

- 🔒 **Local Binding:** The daemon strictly binds to `127.0.0.1:<port>` (never `0.0.0.0`).
- 🛡️ **Anti-DNS-Rebinding:** Strict HTTP `Host` header validation rejects foreign hostnames with `HTTP 403 Forbidden`.
- 🚫 **Anti-CSRF:** Strict `Origin` and `Referer` validation prevents cross-site request forgery attacks from browser tabs.
- 🔑 **Cryptographic Token Authentication:** Every API request requires a random 32-byte CSPRNG token (`X-API-Token`).
- 🔐 **AES-256-GCM & PBKDF2:** All persistent configurations and audit logs are stored encrypted with AES-256-GCM (key derivation via PBKDF2 with **100,000 iterations**).

---

## 📦 Installation

### 1. Download Precompiled Binaries
Download the matching archive from [GitHub Releases](https://github.com/benzjeremy/ollama-fleet-manager/releases):

#### Linux (x86_64):
```bash
tar -xzf ollama-fleet-manager-v1.0-linux.tar.gz
sudo cp ollama-fleet-manager /usr/local/bin/
ollama-fleet-manager --version
```

#### Windows (x86_64):
Extract `ollama-fleet-manager-v1.0-windows.zip` and run `ollama-fleet-manager.exe`.

### 2. Install via Go:
```bash
go install github.com/benzjeremy/ollama-fleet-manager@latest
```

---

## 🚀 Quick Start

Start the daemon in your terminal:
```bash
ollama-fleet-manager --port=8080 --poll-interval=30
```

Console output upon launch:
```text
  ___  _ _                        _____ _           _   ___  ___                                  
 / _ \| | |                      |  ___| |         | |  |  \/  |                                  
/ /_\ \ | | __ _ _ __ ___   __ _ | |_  | | ___  ___| |_ | .  . | __ _ _ __   __ _  __ _  ___ _ __ 
|  _  | | |/ _` | '_ ` _ \ / _` ||  _| | |/ _ \/ _ \ __|| |\/| |/ _` | '_ \ / _` |/ _` |/ _ \ '__|
| | | | | | (_| | | | | | | (_| || |   | |  __/  __/ |_ | |  | | (_| | | | | (_| | (_| |  __/ |   
\_| |_/_|_|\__,_|_| |_| |_|\__,_|\_|   |_|\___|\___|\__|\_|  |_/\__,_|_| |_|\__,_|\__, |\___|_|   
                                                                                   __/ |          
                                                                                  |___/           

🚀 Ollama Fleet Manager v1.0 | Zero-Dummy-Security Edition
🔒 License: GNU General Public License v3.0 (GPL-3.0)
🌐 Bound Address:    http://127.0.0.1:8080
🔑 Access Token:     a9f738b9c...
🤖 Ollama Endpoint:  http://127.0.0.1:11434
💾 Encrypted Vault:  ~/.config/ollama-fleet-manager
🛡️ Guards:           Anti-DNS-Rebinding: ON | Anti-CSRF: ON | AES-256-GCM: ON
--------------------------------------------------------------------------------
📊 Initial Hardware: CPU: 1.8% | RAM: 5812/32014 MB (18.2%) | GPU: NVIDIA RTX (VRAM: 1240/16384 MB, 7.6%)
```

---

## 📡 REST API Documentation

All requests (except `/health`) require the authentication header:
```http
X-API-Token: <your-32-byte-token>
```

### 1. `GET /health`
Returns service status, uptime, and Ollama connectivity:
```json
{
  "status": "ok",
  "service": "ollama-fleet-manager",
  "version": "v1.0",
  "uptime_seconds": 42.1,
  "ollama_status": "online",
  "ollama_version": "0.5.7"
}
```

### 2. `GET /api/v1/metrics`
Returns current real-time hardware telemetry:
```json
{
  "timestamp": "2026-09-10T20:45:00Z",
  "cpu_load_pct": 3.4,
  "ram_total_mb": 32014,
  "ram_used_mb": 6120,
  "ram_free_mb": 25894,
  "ram_usage_pct": 19.1,
  "has_gpu": true,
  "gpu_model": "NVIDIA GeForce RTX 4080",
  "vram_total_mb": 16384,
  "vram_used_mb": 2048,
  "vram_free_mb": 14336,
  "vram_usage_pct": 12.5
}
```

### 3. `POST /api/v1/models/select`
Selects optimal model candidate based on active hardware thresholds and request intent:
```bash
curl -X POST http://127.0.0.1:8080/api/v1/models/select \
  -H "X-API-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{"category": "coding", "purpose": "High-speed code refactoring"}'
```
Response:
```json
{
  "selected_model": {
    "id": "qwen2.5-coder:7b",
    "display_name": "Qwen 2.5 Coder (7B)",
    "category": "coding",
    "min_ram_mb": 8192,
    "min_vram_mb": 6144,
    "context_window": 16384,
    "parameters": "7.6B",
    "quantization": "Q4_K_M"
  },
  "execution_mode": "GPU",
  "reason": "Selected for high-performance GPU execution (14336 MB VRAM available)",
  "requires_evict": false,
  "confidence_score": 0.98
}
```

### 4. `POST /api/v1/models/evict`
Unloads models from memory to release VRAM/RAM for higher-priority system tasks:
```bash
curl -X POST http://127.0.0.1:8080/api/v1/models/evict \
  -H "X-API-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{"model_id": "llama3.1:8b"}'
```

---

## 👥 Contributors & Credits

- Jeremy Benz ([@benzjeremy](https://github.com/benzjeremy)) – Project Founder & Lead Architect
- AI Development Partner: Google Antigravity
- © 2026 Jeremy Benz

---

## 📄 License

This project is licensed under the **GNU General Public License, Version 3.0 (GPL-3.0)**.  
See [LICENSE](LICENSE) for details.
