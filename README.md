# 🚀 ollama-fleet-manager

[![CI](https://github.com/benzjeremy/ollama-fleet-manager/actions/workflows/ci.yml/badge.svg)](https://github.com/benzjeremy/ollama-fleet-manager/actions)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](https://github.com/benzjeremy/ollama-fleet-manager/blob/main/LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![Release](https://img.shields.io/badge/Release-v1.0-f43f5e.svg)](https://github.com/benzjeremy/ollama-fleet-manager/releases/tag/v1.0)

Intelligenter, ressourcenschonender Fleet Manager und dynamischer Modell-Router für [Ollama](https://ollama.com). Entwickelt in purem Go mit Hardware-Telemetrie-Sensoren (CPU, RAM, GPU-VRAM), automatischer OOM-Prävention und kompromissloser **Zero-Dummy-Security**.

---

## 💡 Warum ollama-fleet-manager?

Monolithische KI-Deployments scheitern in der Praxis oft an unvorhersehbaren Speicherengpässen:
1. **OOM-Crashes (Out of Memory):** Werden große Modelle wie `llama3.1:8b` oder `deepseek-r1:8b` parallel gestartet, killt der Linux-Kernel den Prozess (`OOM-Killer`).
2. **Ressourcenverschwendung:** Einfache Aufgaben (Textzusammenfassung, Rechtschreibkorrektur) benötigen keine 8B-Parameter und belegen unnötig wertvollen GPU-VRAM.
3. **Mangelnde Sicherheit lokaler Schnittstellen:** Viele KI-Server binden ungeschützt an `0.0.0.0` oder besitzen keine CSRF- / DNS-Rebinding-Schutzmechanismen.

**ollama-fleet-manager** löst diese Probleme durch eine kontinuierliche Hardware-Überwachung, eine regelbasierte Matching-Engine und kryptografisch abgesicherte lokale Schnittstellen.

---

## 🛡️ Zero-Dummy-Security (Militärischer Standard ab Werk)

- 🔒 **Lokale Bindung:** Der Server lauscht ausnahmslos an `127.0.0.1:<port>` (niemals `0.0.0.0`).
- 🛡️ **Anti-DNS-Rebinding:** Strikte Validierung des HTTP-`Host`-Headers. Fremde Domain-Anfragen werden sofort mit `HTTP 403 Forbidden` abgewiesen.
- 🚫 **Anti-CSRF:** Prüfung des `Origin`-Headers. Verhindert böswillige API-Trigger aus fremden Browser-Tabs.
- 🔑 **Kryptografische Token-Authentifizierung:** Jeder API-Aufruf erfordert ein zufälliges 32-Byte CSPRNG-Token (`X-API-Token`).
- 🔐 **AES-256-GCM & PBKDF2:** Alle persistenten Konfigurationen und Audit-Logs werden im lokalen Tresor mit AES-256-GCM verschlüsselt gespeichert (Schlüsselableitung via PBKDF2 mit **100.000 Iterationen**).

---

## 📦 Installation

### 1. Vorab kompilierte Binaries herunterladen
Lade das passende Archiv aus den [GitHub Releases](https://github.com/benzjeremy/ollama-fleet-manager/releases) herunter:

#### Linux (x86_64):
```bash
tar -xzf ollama-fleet-manager-v1.0-linux.tar.gz
sudo cp ollama-fleet-manager /usr/local/bin/
ollama-fleet-manager --version
```

#### Windows (x86_64):
Entpacke `ollama-fleet-manager-v1.0-windows.zip` und starte `ollama-fleet-manager.exe`.

### 2. Installation via Go:
```bash
go install github.com/benzjeremy/ollama-fleet-manager@latest
```

---

## 🚀 Schnellstart

Starte den Dienst im Terminal:
```bash
ollama-fleet-manager --port=8080 --poll-interval=30
```

Ausgabe beim Start:
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

## 📡 REST-API Dokumentation

Alle Anfragen (außer `/health`) erfordern den Header:
```http
X-API-Token: <dein-32-byte-token>
```

### 1. `GET /health`
Liefert Status, Uptime und Konnektivität zur lokalen Ollama-Instanz.
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
Liefert die aktuellen Hardware-Telemetriedaten:
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
Wählt das beste Modell basierend auf der aktuellen Hardware und dem Einsatzzweck:
```bash
curl -X POST http://127.0.0.1:8080/api/v1/models/select \
  -H "X-API-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{"category": "coding", "purpose": "High-speed code refactoring"}'
```
Antwort:
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
Entlädt geladene Modelle aus dem VRAM/RAM, um Speicher für dringende Systemprozesse freizugeben:
```bash
curl -X POST http://127.0.0.1:8080/api/v1/models/evict \
  -H "X-API-Token: <token>" \
  -H "Content-Type: application/json" \
  -d '{"model_id": "llama3.1:8b"}'
```

---

## 👥 Mitwirkende & Credits

- Jeremy Benz ([@benzjeremy](https://github.com/benzjeremy) & [@jbenz1706](https://github.com/jbenz1706)) – Projektgründer & Lead Architect
- KI-Entwicklungspartner: Google Antigravity
- © 2026 Jeremy Benz

---

## 📄 Lizenz

Dieses Projekt steht unter der **GNU General Public License, Version 3.0 (GPL-3.0)**.  
Details siehe [LICENSE](https://github.com/benzjeremy/ollama-fleet-manager/blob/main/LICENSE).
