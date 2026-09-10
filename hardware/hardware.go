package hardware

import (
	"fmt"
	"time"
)

// ResourceSnapshot contains the complete telemetry of system CPU, RAM, and GPU.
type ResourceSnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	CPULoadPct   float64   `json:"cpu_load_pct"`
	RAMTotalMB   uint64    `json:"ram_total_mb"`
	RAMUsedMB    uint64    `json:"ram_used_mb"`
	RAMFreeMB    uint64    `json:"ram_free_mb"`
	RAMUsagePct  float64   `json:"ram_usage_pct"`
	HasGPU       bool      `json:"has_gpu"`
	GPUModel     string    `json:"gpu_model,omitempty"`
	VRAMTotalMB  uint64    `json:"vram_total_mb,omitempty"`
	VRAMUsedMB   uint64    `json:"vram_used_mb,omitempty"`
	VRAMFreeMB   uint64    `json:"vram_free_mb,omitempty"`
	VRAMUsagePct float64   `json:"vram_usage_pct,omitempty"`
}

// String provides a human-readable summary of the resource state.
func (r ResourceSnapshot) String() string {
	gpuInfo := "None"
	if r.HasGPU {
		gpuInfo = fmt.Sprintf("%s (VRAM: %d/%d MB, %.1f%%)", r.GPUModel, r.VRAMUsedMB, r.VRAMTotalMB, r.VRAMUsagePct)
	}
	return fmt.Sprintf("CPU: %.1f%% | RAM: %d/%d MB (%.1f%%) | GPU: %s",
		r.CPULoadPct, r.RAMUsedMB, r.RAMTotalMB, r.RAMUsagePct, gpuInfo)
}

// IsConstrained returns true if system RAM or GPU VRAM is above safety thresholds (85%).
func (r ResourceSnapshot) IsConstrained() bool {
	if r.RAMUsagePct > 85.0 {
		return true
	}
	if r.HasGPU && r.VRAMUsagePct > 88.0 {
		return true
	}
	return false
}
