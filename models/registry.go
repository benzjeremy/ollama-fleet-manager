package models

import (
	"fmt"
	"strings"
	"sync"
)

// Registry manages the catalog of available models.
type Registry struct {
	mu     sync.RWMutex
	models map[string]ModelSpec
}

// NewRegistry initializes the registry populated with standard verified Ollama models.
func NewRegistry() *Registry {
	r := &Registry{
		models: make(map[string]ModelSpec),
	}

	defaults := []ModelSpec{
		{
			ID:              "llama3.2:3b",
			DisplayName:     "Llama 3.2 (3B)",
			Category:        CategoryGeneral,
			MinRAMMB:        4096,
			MinVRAMMB:       2800,
			ContextWindow:   8192,
			Parameters:      "3.2B",
			Quantization:    "Q4_K_M",
			RecommendedTask: "Low-overhead daily chat, quick summaries, CPU execution",
		},
		{
			ID:              "llama3.1:8b",
			DisplayName:     "Llama 3.1 (8B)",
			Category:        CategoryWriting,
			MinRAMMB:        8192,
			MinVRAMMB:       6144,
			ContextWindow:   16384,
			Parameters:      "8.0B",
			Quantization:    "Q4_K_M",
			RecommendedTask: "Complex text synthesis, copywriting, multi-step dialogue",
		},
		{
			ID:              "qwen2.5-coder:7b",
			DisplayName:     "Qwen 2.5 Coder (7B)",
			Category:        CategoryCoding,
			MinRAMMB:        8192,
			MinVRAMMB:       6144,
			ContextWindow:   16384,
			Parameters:      "7.6B",
			Quantization:    "Q4_K_M",
			RecommendedTask: "High-precision code completion, refactoring, AST debugging",
		},
		{
			ID:              "deepseek-r1:8b",
			DisplayName:     "DeepSeek R1 (8B)",
			Category:        CategoryReasoning,
			MinRAMMB:        10240,
			MinVRAMMB:       7168,
			ContextWindow:   16384,
			Parameters:      "8.2B",
			Quantization:    "Q4_K_M",
			RecommendedTask: "Formal logic, causal graph reasoning, architectural analysis",
		},
		{
			ID:              "llava:7b",
			DisplayName:     "LLaVA Vision (7B)",
			Category:        CategoryVision,
			MinRAMMB:        10240,
			MinVRAMMB:       7168,
			ContextWindow:   4096,
			Parameters:      "7.0B",
			Quantization:    "Q4_K_M",
			RecommendedTask: "Screen inspection, diagram OCR, CAD/UI layout analysis",
		},
		{
			ID:              "tinyllama:1.1b",
			DisplayName:     "TinyLlama (1.1B)",
			Category:        CategoryGeneral,
			MinRAMMB:        2048,
			MinVRAMMB:       1200,
			ContextWindow:   2048,
			Parameters:      "1.1B",
			Quantization:    "Q4_K_M",
			RecommendedTask: "Emergency constrained fallback mode under severe OOM pressure",
		},
	}

	for _, m := range defaults {
		r.models[m.ID] = m
	}
	return r
}

// Get returns the ModelSpec for a given ID.
func (r *Registry) Get(id string) (ModelSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.models[id]
	return m, ok
}

// All returns a copy of all registered models.
func (r *Registry) All() []ModelSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]ModelSpec, 0, len(r.models))
	for _, m := range r.models {
		res = append(res, m)
	}
	return res
}

// FilterByCategory returns models matching the specified task category.
func (r *Registry) FilterByCategory(cat TaskCategory) []ModelSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []ModelSpec
	for _, m := range r.models {
		if m.Category == cat || (cat == CategoryGeneral) {
			res = append(res, m)
		}
	}
	return res
}

// Register adds or updates a custom model specification.
func (r *Registry) Register(spec ModelSpec) error {
	if strings.TrimSpace(spec.ID) == "" {
		return fmt.Errorf("model id cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[spec.ID] = spec
	return nil
}
