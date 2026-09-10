package engine

import (
	"fmt"
	"sort"

	"github.com/benzjeremy/ollama-fleet-manager/hardware"
	"github.com/benzjeremy/ollama-fleet-manager/models"
)

// Decision represents the recommendation of the rule engine.
type Decision struct {
	SelectedModel   models.ModelSpec `json:"selected_model"`
	ExecutionMode   string           `json:"execution_mode"` // "GPU" or "CPU"
	Reason          string           `json:"reason"`
	RequiresEvict   bool             `json:"requires_evict"`
	ConfidenceScore float64          `json:"confidence_score"`
}

// RuleEngine evaluates hardware telemetry and matches the optimal model.
type RuleEngine struct {
	registry *models.Registry
}

// NewRuleEngine initializes the decision engine.
func NewRuleEngine(reg *models.Registry) *RuleEngine {
	return &RuleEngine{
		registry: reg,
	}
}

// Evaluate determines the optimal model for a requested task given system resources.
func (e *RuleEngine) Evaluate(category models.TaskCategory, snap hardware.ResourceSnapshot) (Decision, error) {
	candidates := e.registry.FilterByCategory(category)
	if len(candidates) == 0 {
		candidates = e.registry.All()
	}

	// Sort candidates by capability / minimum requirement descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].MinRAMMB > candidates[j].MinRAMMB
	})

	// 1. If system is already heavily constrained, prioritize emergency fallback
	if snap.IsConstrained() {
		fallback, ok := e.registry.Get("tinyllama:1.1b")
		if ok {
			return Decision{
				SelectedModel:   fallback,
				ExecutionMode:   "CPU",
				Reason:          "System memory constrained (>85%); defaulting to emergency lightweight model",
				RequiresEvict:   true,
				ConfidenceScore: 0.95,
			}, nil
		}
	}

	// 2. Try to fit candidate onto GPU if available
	if snap.HasGPU && snap.VRAMFreeMB > 0 {
		for _, cand := range candidates {
			// Require 1024 MB buffer in VRAM to prevent GPU driver stutter
			if snap.VRAMFreeMB >= (cand.MinVRAMMB + 1024) {
				return Decision{
					SelectedModel: cand,
					ExecutionMode: "GPU",
					Reason: fmt.Sprintf("Selected for high-performance GPU execution (%d MB VRAM available)",
						snap.VRAMFreeMB),
					RequiresEvict:   false,
					ConfidenceScore: 0.98,
				}, nil
			}
		}
	}

	// 3. Fallback to CPU execution if system RAM is sufficient
	for _, cand := range candidates {
		// Require 2048 MB buffer in host RAM for OS operations
		if snap.RAMFreeMB >= (cand.MinRAMMB + 2048) {
			return Decision{
				SelectedModel: cand,
				ExecutionMode: "CPU",
				Reason: fmt.Sprintf("Selected for CPU execution (%d MB host RAM available)",
					snap.RAMFreeMB),
				RequiresEvict:   false,
				ConfidenceScore: 0.88,
			}, nil
		}
	}

	// 4. If no standard candidate fits comfortably, find the smallest available model
	var smallest models.ModelSpec
	var found bool
	for _, m := range e.registry.All() {
		if !found || m.MinRAMMB < smallest.MinRAMMB {
			smallest = m
			found = true
		}
	}

	if found {
		return Decision{
			SelectedModel:   smallest,
			ExecutionMode:   "CPU",
			Reason:          "Low memory headroom: selected smallest registered model to avoid OOM killer",
			RequiresEvict:   true,
			ConfidenceScore: 0.75,
		}, nil
	}

	return Decision{}, fmt.Errorf("no suitable model found for category: %s", category)
}
