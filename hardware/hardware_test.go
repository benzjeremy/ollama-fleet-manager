package hardware

import (
	"testing"
)

func TestHardwareMonitor(t *testing.T) {
	mon := NewMonitor()
	snap := mon.Snapshot()

	if snap.RAMTotalMB == 0 {
		t.Errorf("Expected RAMTotalMB > 0, got %d", snap.RAMTotalMB)
	}

	if snap.RAMUsagePct < 0 || snap.RAMUsagePct > 100 {
		t.Errorf("Expected RAMUsagePct between 0 and 100, got %f", snap.RAMUsagePct)
	}

	str := snap.String()
	if len(str) == 0 {
		t.Errorf("Expected non-empty string representation")
	}
}

func TestMockGPUMonitor(t *testing.T) {
	mon := NewMonitor()
	mon.gpuDetector = func() (bool, string, uint64, uint64) {
		return true, "NVIDIA GeForce RTX 4090", 24576, 4096
	}

	snap := mon.Snapshot()
	if !snap.HasGPU {
		t.Errorf("Expected HasGPU=true")
	}
	if snap.VRAMTotalMB != 24576 {
		t.Errorf("Expected VRAMTotalMB=24576, got %d", snap.VRAMTotalMB)
	}
	if snap.VRAMUsedMB != 4096 {
		t.Errorf("Expected VRAMUsedMB=4096, got %d", snap.VRAMUsedMB)
	}
	if snap.VRAMFreeMB != 20480 {
		t.Errorf("Expected VRAMFreeMB=20480, got %d", snap.VRAMFreeMB)
	}
	if snap.IsConstrained() {
		t.Errorf("Expected IsConstrained=false under low VRAM usage")
	}
}
