package hardware

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Monitor collects system performance telemetry.
type Monitor struct {
	mu           sync.RWMutex
	lastCPUStats []uint64
	lastCPUTime  time.Time
	gpuDetector  func() (bool, string, uint64, uint64)
}

// NewMonitor initializes a hardware telemetry monitor.
func NewMonitor() *Monitor {
	return &Monitor{
		gpuDetector: DetectGPU,
	}
}

// Snapshot returns the current ResourceSnapshot.
func (m *Monitor) Snapshot() ResourceSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	ramTotal, ramUsed, ramFree, ramPct := m.getMemoryInfo()
	cpuPct := m.getCPULoad()

	hasGPU, gpuModel, vramTotal, vramUsed := m.gpuDetector()
	var vramFree uint64
	var vramPct float64
	if hasGPU && vramTotal > 0 {
		if vramTotal > vramUsed {
			vramFree = vramTotal - vramUsed
		}
		vramPct = (float64(vramUsed) / float64(vramTotal)) * 100.0
	}

	return ResourceSnapshot{
		Timestamp:    time.Now().UTC(),
		CPULoadPct:   cpuPct,
		RAMTotalMB:   ramTotal,
		RAMUsedMB:    ramUsed,
		RAMFreeMB:    ramFree,
		RAMUsagePct:  ramPct,
		HasGPU:       hasGPU,
		GPUModel:     gpuModel,
		VRAMTotalMB:  vramTotal,
		VRAMUsedMB:   vramUsed,
		VRAMFreeMB:   vramFree,
		VRAMUsagePct: vramPct,
	}
}

// getMemoryInfo parses /proc/meminfo on Linux or falls back to runtime statistics.
func (m *Monitor) getMemoryInfo() (totalMB, usedMB, freeMB uint64, usagePct float64) {
	if runtime.GOOS == "linux" {
		f, err := os.Open("/proc/meminfo")
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			var memTotal, memAvailable uint64
			for scanner.Scan() {
				line := scanner.Text()
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if fields[0] == "MemTotal:" {
						memTotal, _ = strconv.ParseUint(fields[1], 10, 64)
					} else if fields[0] == "MemAvailable:" {
						memAvailable, _ = strconv.ParseUint(fields[1], 10, 64)
					}
				}
			}
			if memTotal > 0 {
				totalMB = memTotal / 1024
				freeMB = memAvailable / 1024
				if totalMB >= freeMB {
					usedMB = totalMB - freeMB
				}
				usagePct = (float64(usedMB) / float64(totalMB)) * 100.0
				return totalMB, usedMB, freeMB, usagePct
			}
		}
	}

	// Cross-platform fallback estimation based on runtime memory stats
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	totalMB = 16384 // Default estimated 16 GB baseline for testing/Windows
	usedMB = rtm.Alloc / (1024 * 1024)
	if usedMB > totalMB {
		usedMB = totalMB / 2
	}
	freeMB = totalMB - usedMB
	usagePct = (float64(usedMB) / float64(totalMB)) * 100.0
	return totalMB, usedMB, freeMB, usagePct
}

// getCPULoad computes instantaneous CPU utilization percentage.
func (m *Monitor) getCPULoad() float64 {
	if runtime.GOOS == "linux" {
		f, err := os.Open("/proc/stat")
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			if scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				if len(fields) > 4 && fields[0] == "cpu" {
					var currentStats []uint64
					for _, fld := range fields[1:] {
						val, _ := strconv.ParseUint(fld, 10, 64)
						currentStats = append(currentStats, val)
					}

					now := time.Now()
					if len(m.lastCPUStats) > 3 && !m.lastCPUTime.IsZero() {
						prevIdle := m.lastCPUStats[3]
						currIdle := currentStats[3]

						var prevTotal, currTotal uint64
						for _, v := range m.lastCPUStats {
							prevTotal += v
						}
						for _, v := range currentStats {
							currTotal += v
						}

						totalDelta := float64(currTotal - prevTotal)
						idleDelta := float64(currIdle - prevIdle)

						m.lastCPUStats = currentStats
						m.lastCPUTime = now

						if totalDelta > 0 {
							cpuLoad := ((totalDelta - idleDelta) / totalDelta) * 100.0
							if cpuLoad < 0 {
								cpuLoad = 0
							}
							if cpuLoad > 100 {
								cpuLoad = 100
							}
							return cpuLoad
						}
					}
					m.lastCPUStats = currentStats
					m.lastCPUTime = now
				}
			}
		}
	}
	return 2.5 // Fallback baseline idle load
}
