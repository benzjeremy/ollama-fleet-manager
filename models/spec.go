package models

// TaskCategory represents the primary workload intended for an Ollama model.
type TaskCategory string

const (
	CategoryCoding    TaskCategory = "coding"
	CategoryWriting   TaskCategory = "writing"
	CategoryReasoning TaskCategory = "reasoning"
	CategoryVision    TaskCategory = "vision"
	CategoryGeneral   TaskCategory = "general"
)

// ModelSpec defines hardware requirements and characteristics for an AI model.
type ModelSpec struct {
	ID              string       `json:"id"`
	DisplayName     string       `json:"display_name"`
	Category        TaskCategory `json:"category"`
	MinRAMMB        uint64       `json:"min_ram_mb"`
	MinVRAMMB       uint64       `json:"min_vram_mb"`
	ContextWindow   int          `json:"context_window"`
	Parameters      string       `json:"parameters"`
	Quantization    string       `json:"quantization"`
	RecommendedTask string       `json:"recommended_task"`
}
