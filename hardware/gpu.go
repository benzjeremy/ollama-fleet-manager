package hardware

import (
	"os/exec"
	"strconv"
	"strings"
)

// DetectGPU checks for available GPU hardware (NVIDIA SMI or system drivers).
func DetectGPU() (hasGPU bool, model string, vramTotalMB uint64, vramUsedMB uint64) {
	// 1. Try querying nvidia-smi with CSV output
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,memory.used", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			parts := strings.Split(lines[0], ",")
			if len(parts) >= 3 {
				name := strings.TrimSpace(parts[0])
				total, err1 := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
				used, err2 := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
				if err1 == nil && err2 == nil && total > 0 {
					return true, name, total, used
				}
			}
		}
	}

	// Default fallback: No discrete NVIDIA GPU detected, CPU-only execution
	return false, "CPU Only", 0, 0
}
