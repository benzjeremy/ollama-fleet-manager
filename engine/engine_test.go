package engine

import (
	"testing"

	"github.com/benzjeremy/ollama-fleet-manager/hardware"
	"github.com/benzjeremy/ollama-fleet-manager/models"
)

func TestRuleEngineGPUSelection(t *testing.T) {
	reg := models.NewRegistry()
	eng := NewRuleEngine(reg)

	// High-spec GPU with 16 GB VRAM free
	snap := hardware.ResourceSnapshot{
		RAMTotalMB:   32768,
		RAMFreeMB:    20480,
		RAMUsagePct:  37.5,
		HasGPU:       true,
		GPUModel:     "NVIDIA RTX 4080",
		VRAMTotalMB:  16384,
		VRAMFreeMB:   12288,
		VRAMUsagePct: 25.0,
	}

	dec, err := eng.Evaluate(models.CategoryCoding, snap)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if dec.ExecutionMode != "GPU" {
		t.Errorf("Expected GPU execution mode, got %s", dec.ExecutionMode)
	}
	if dec.SelectedModel.ID != "qwen2.5-coder:7b" {
		t.Errorf("Expected qwen2.5-coder:7b, got %s", dec.SelectedModel.ID)
	}
}

func TestRuleEngineCPUFallback(t *testing.T) {
	reg := models.NewRegistry()
	eng := NewRuleEngine(reg)

	// No GPU, but ample host RAM (32 GB)
	snap := hardware.ResourceSnapshot{
		RAMTotalMB:   32768,
		RAMFreeMB:    16384,
		RAMUsagePct:  50.0,
		HasGPU:       false,
	}

	dec, err := eng.Evaluate(models.CategoryReasoning, snap)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if dec.ExecutionMode != "CPU" {
		t.Errorf("Expected CPU execution mode, got %s", dec.ExecutionMode)
	}
	if dec.SelectedModel.ID != "deepseek-r1:8b" {
		t.Errorf("Expected deepseek-r1:8b, got %s", dec.SelectedModel.ID)
	}
}

func TestRuleEngineConstrainedOOMProtection(t *testing.T) {
	reg := models.NewRegistry()
	eng := NewRuleEngine(reg)

	// Memory under heavy pressure (92% RAM usage)
	snap := hardware.ResourceSnapshot{
		RAMTotalMB:   16384,
		RAMFreeMB:    1024,
		RAMUsagePct:  93.75,
		HasGPU:       false,
	}

	dec, err := eng.Evaluate(models.CategoryCoding, snap)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if dec.SelectedModel.ID != "tinyllama:1.1b" {
		t.Errorf("Expected emergency fallback tinyllama:1.1b, got %s", dec.SelectedModel.ID)
	}
	if !dec.RequiresEvict {
		t.Errorf("Expected RequiresEvict=true under constrained memory")
	}
}
