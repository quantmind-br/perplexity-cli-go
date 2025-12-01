package models

import "testing"

func TestIsValidModel(t *testing.T) {
	tests := []struct {
		name  string
		model Model
		want  bool
	}{
		{"valid pplx_pro", ModelPplxPro, true},
		{"valid gpt5", ModelGPT5, true},
		{"valid claude45sonnet", ModelClaude45Sonnet, true},
		{"valid experimental", ModelExperimental, true},
		{"valid sonar", ModelSonar, true},
		{"valid grok4", ModelGrok4, true},
		{"valid gemini2flash", ModelGemini2Flash, true},
		{"valid gpt5_thinking", ModelGPT5Thinking, true},
		{"valid claude45sonnetthinking", ModelClaude45SonnetThink, true},
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
	if len(AvailableModels) != 9 {
		t.Errorf("Expected 9 available models, got %d", len(AvailableModels))
	}

	// Verify all models in the list are valid
	for _, m := range AvailableModels {
		if !IsValidModel(m) {
			t.Errorf("Model %q in AvailableModels is not valid", m)
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
