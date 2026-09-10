package models

import (
	"testing"
)

func TestModelRegistry(t *testing.T) {
	reg := NewRegistry()

	all := reg.All()
	if len(all) < 5 {
		t.Errorf("Expected at least 5 default models, got %d", len(all))
	}

	coder, ok := reg.Get("qwen2.5-coder:7b")
	if !ok {
		t.Fatalf("Expected qwen2.5-coder:7b in registry")
	}
	if coder.Category != CategoryCoding {
		t.Errorf("Expected category %s, got %s", CategoryCoding, coder.Category)
	}

	codingModels := reg.FilterByCategory(CategoryCoding)
	if len(codingModels) == 0 {
		t.Errorf("Expected at least 1 coding model")
	}

	custom := ModelSpec{
		ID:          "custom:1b",
		DisplayName: "Custom 1B",
		Category:    CategoryWriting,
		MinRAMMB:    1024,
	}
	if err := reg.Register(custom); err != nil {
		t.Fatalf("Failed to register custom model: %v", err)
	}

	c, found := reg.Get("custom:1b")
	if !found || c.DisplayName != "Custom 1B" {
		t.Errorf("Registered model not properly retrieved")
	}
}
