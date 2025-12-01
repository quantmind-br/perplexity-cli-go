package models

import "testing"

func TestIsValidModel(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		want  bool
	}{
		// Pro models
		{"valid pplx_pro", ModelPplxPro, true},
		{"valid gpt51", ModelGPT51, true},
		{"valid grok41nonreasoning", ModelGrok41NonReasoning, true},
		{"valid experimental", ModelExperimental, true},
		{"valid claude45sonnet", ModelClaude45Sonnet, true},
		// Reasoning models
		{"valid gemini30pro", ModelGemini30Pro, true},
		{"valid gpt51_thinking", ModelGPT51Thinking, true},
		{"valid grok41reasoning", ModelGrok41Reasoning, true},
		{"valid kimik2thinking", ModelKimiK2Thinking, true},
		{"valid claude45sonnetthinking", ModelClaude45SonnetThink, true},
		// Invalid
		{"invalid model", Model("invalid_model"), false},
		{"empty model", Model(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidModel(tt.model)
			if got != tt.want {
				t.Errorf("IsValidModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestIsValidSource(t *testing.T) {
	tests := []struct {
		name   string
		source Source
		want   bool
	}{
		{"valid web", SourceWeb, true},
		{"valid scholar", SourceScholar, true},
		{"valid social", SourceSocial, true},
		{"invalid source", Source("invalid"), false},
		{"empty source", Source(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidSource(tt.source)
			if got != tt.want {
				t.Errorf("IsValidSource(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

func TestIsValidMode(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
		want bool
	}{
		{"valid fast", ModeFast, true},
		{"valid pro", ModePro, true},
		{"valid reasoning", ModeReasoning, true},
		{"valid deep-research", ModeDeepResearch, true},
		{"valid default", ModeDefault, true},
		{"invalid mode", Mode("invalid"), false},
		{"empty mode", Mode(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidMode(tt.mode)
			if got != tt.want {
				t.Errorf("IsValidMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestAvailableModels(t *testing.T) {
	expectedTotal := len(AvailableProModels) + len(AvailableReasoningModels)
	if len(AvailableModels) != expectedTotal {
		t.Errorf("Expected %d available models, got %d", expectedTotal, len(AvailableModels))
	}

	// Verify all models in the list are valid
	for _, m := range AvailableModels {
		if !IsValidModel(m) {
			t.Errorf("Model %q in AvailableModels is not valid", m)
		}
	}
}

func TestAvailableProModels(t *testing.T) {
	if len(AvailableProModels) != 5 {
		t.Errorf("Expected 5 pro models, got %d", len(AvailableProModels))
	}

	expectedModels := []Model{
		ModelPplxPro,
		ModelGPT51,
		ModelGrok41NonReasoning,
		ModelExperimental,
		ModelClaude45Sonnet,
	}

	for i, m := range AvailableProModels {
		if m != expectedModels[i] {
			t.Errorf("AvailableProModels[%d] = %q, want %q", i, m, expectedModels[i])
		}
	}
}

func TestAvailableReasoningModels(t *testing.T) {
	if len(AvailableReasoningModels) != 5 {
		t.Errorf("Expected 5 reasoning models, got %d", len(AvailableReasoningModels))
	}

	expectedModels := []Model{
		ModelGemini30Pro,
		ModelGPT51Thinking,
		ModelGrok41Reasoning,
		ModelKimiK2Thinking,
		ModelClaude45SonnetThink,
	}

	for i, m := range AvailableReasoningModels {
		if m != expectedModels[i] {
			t.Errorf("AvailableReasoningModels[%d] = %q, want %q", i, m, expectedModels[i])
		}
	}
}

func TestAvailableSources(t *testing.T) {
	if len(AvailableSources) != 3 {
		t.Errorf("Expected 3 available sources, got %d", len(AvailableSources))
	}

	expectedSources := []Source{SourceWeb, SourceScholar, SourceSocial}
	for i, s := range AvailableSources {
		if s != expectedSources[i] {
			t.Errorf("AvailableSources[%d] = %q, want %q", i, s, expectedSources[i])
		}
	}
}
