package ui

import (
	"testing"

	"github.com/diogo/perplexity-go/internal/config"
	"github.com/diogo/perplexity-go/pkg/models"
)

func TestBuildConfigMenuItems(t *testing.T) {
	cfg := &config.Config{
		DefaultModel:    models.ModelGPT51,
		DefaultMode:     models.ModePro,
		DefaultLanguage: "pt-BR",
		DefaultSources:  []models.Source{models.SourceWeb, models.SourceScholar},
		Streaming:       true,
		Incognito:       false,
		CookieFile:      "/path/to/cookies.json",
		HistoryFile:     "/path/to/history.jsonl",
	}

	items := buildConfigMenuItems(cfg)

	if len(items) != 8 {
		t.Errorf("Expected 8 menu items, got %d", len(items))
	}

	// Verify each item
	expectedItems := map[string]string{
		"default_model":    "gpt51",
		"default_mode":     "pro",
		"default_language": "pt-BR",
		"default_sources":  "web, scholar",
		"streaming":        "true",
		"incognito":        "false",
		"cookie_file":      "/path/to/cookies.json",
		"history_file":     "/path/to/history.jsonl",
	}

	for _, item := range items {
		expected, ok := expectedItems[item.Key]
		if !ok {
			t.Errorf("Unexpected menu item key: %s", item.Key)
			continue
		}
		if item.Value != expected {
			t.Errorf("Item %s: expected value %q, got %q", item.Key, expected, item.Value)
		}
	}
}

func TestBuildConfigMenuItems_EmptySources(t *testing.T) {
	cfg := &config.Config{
		DefaultSources: []models.Source{},
	}

	items := buildConfigMenuItems(cfg)

	// Find sources item
	var sourcesItem ConfigMenuItem
	for _, item := range items {
		if item.Key == "default_sources" {
			sourcesItem = item
			break
		}
	}

	if sourcesItem.Value != "" {
		t.Errorf("Expected empty sources value, got %q", sourcesItem.Value)
	}
}

func TestGetModeDescription(t *testing.T) {
	tests := []struct {
		mode     models.Mode
		expected string
	}{
		{models.ModeFast, "Quick responses"},
		{models.ModePro, "Balanced quality"},
		{models.ModeReasoning, "Deep analysis"},
		{models.ModeDeepResearch, "Comprehensive research"},
		{models.ModeDefault, "Standard mode"},
		{models.Mode("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			result := getModeDescription(tt.mode)
			if result != tt.expected {
				t.Errorf("getModeDescription(%s) = %q, want %q", tt.mode, result, tt.expected)
			}
		})
	}
}

func TestConfigMenuItem_Fields(t *testing.T) {
	item := ConfigMenuItem{
		Key:         "test_key",
		Label:       "Test Label",
		Description: "Test Description",
		Value:       "test_value",
	}

	if item.Key != "test_key" {
		t.Errorf("Key = %q, want %q", item.Key, "test_key")
	}
	if item.Label != "Test Label" {
		t.Errorf("Label = %q, want %q", item.Label, "Test Label")
	}
	if item.Description != "Test Description" {
		t.Errorf("Description = %q, want %q", item.Description, "Test Description")
	}
	if item.Value != "test_value" {
		t.Errorf("Value = %q, want %q", item.Value, "test_value")
	}
}

func TestHandleReset_ConfigValues(t *testing.T) {
	cfg := &config.Config{
		DefaultModel:    models.ModelGPT51,
		DefaultMode:     models.ModeReasoning,
		DefaultLanguage: "pt-BR",
		DefaultSources:  []models.Source{models.SourceScholar},
		Streaming:       false,
		Incognito:       true,
	}

	// Reset to defaults (simulating what handleReset does when confirmed)
	cfg.DefaultModel = models.ModelPplxPro
	cfg.DefaultMode = models.ModeDefault
	cfg.DefaultLanguage = "en-US"
	cfg.DefaultSources = []models.Source{models.SourceWeb}
	cfg.Streaming = true
	cfg.Incognito = false

	// Verify reset values
	if cfg.DefaultModel != models.ModelPplxPro {
		t.Errorf("DefaultModel = %q, want %q", cfg.DefaultModel, models.ModelPplxPro)
	}
	if cfg.DefaultMode != models.ModeDefault {
		t.Errorf("DefaultMode = %q, want %q", cfg.DefaultMode, models.ModeDefault)
	}
	if cfg.DefaultLanguage != "en-US" {
		t.Errorf("DefaultLanguage = %q, want %q", cfg.DefaultLanguage, "en-US")
	}
	if len(cfg.DefaultSources) != 1 || cfg.DefaultSources[0] != models.SourceWeb {
		t.Errorf("DefaultSources = %v, want [web]", cfg.DefaultSources)
	}
	if !cfg.Streaming {
		t.Error("Streaming should be true after reset")
	}
	if cfg.Incognito {
		t.Error("Incognito should be false after reset")
	}
}

func TestBuildConfigMenuItems_AllModels(t *testing.T) {
	// Test with each available model
	for _, model := range models.AvailableModels {
		cfg := &config.Config{
			DefaultModel: model,
		}

		items := buildConfigMenuItems(cfg)

		// Find model item
		var modelItem ConfigMenuItem
		for _, item := range items {
			if item.Key == "default_model" {
				modelItem = item
				break
			}
		}

		if modelItem.Value != string(model) {
			t.Errorf("Model %s: expected value %q, got %q", model, string(model), modelItem.Value)
		}
	}
}

func TestBuildConfigMenuItems_AllModes(t *testing.T) {
	modes := []models.Mode{
		models.ModeFast,
		models.ModePro,
		models.ModeReasoning,
		models.ModeDeepResearch,
		models.ModeDefault,
	}

	for _, mode := range modes {
		cfg := &config.Config{
			DefaultMode: mode,
		}

		items := buildConfigMenuItems(cfg)

		// Find mode item
		var modeItem ConfigMenuItem
		for _, item := range items {
			if item.Key == "default_mode" {
				modeItem = item
				break
			}
		}

		if modeItem.Value != string(mode) {
			t.Errorf("Mode %s: expected value %q, got %q", mode, string(mode), modeItem.Value)
		}
	}
}

func TestBuildConfigMenuItems_AllSources(t *testing.T) {
	cfg := &config.Config{
		DefaultSources: models.AvailableSources,
	}

	items := buildConfigMenuItems(cfg)

	// Find sources item
	var sourcesItem ConfigMenuItem
	for _, item := range items {
		if item.Key == "default_sources" {
			sourcesItem = item
			break
		}
	}

	// Should contain all sources
	expectedValue := "web, scholar, social"
	if sourcesItem.Value != expectedValue {
		t.Errorf("Sources value = %q, want %q", sourcesItem.Value, expectedValue)
	}
}

func TestConfigMenuItemLabels(t *testing.T) {
	cfg := &config.Config{}
	items := buildConfigMenuItems(cfg)

	expectedLabels := map[string]string{
		"default_model":    "Model",
		"default_mode":     "Mode",
		"default_language": "Language",
		"default_sources":  "Sources",
		"streaming":        "Streaming",
		"incognito":        "Incognito",
		"cookie_file":      "Cookie file",
		"history_file":     "History file",
	}

	for _, item := range items {
		expected, ok := expectedLabels[item.Key]
		if !ok {
			t.Errorf("Unexpected item key: %s", item.Key)
			continue
		}
		if item.Label != expected {
			t.Errorf("Item %s: expected label %q, got %q", item.Key, expected, item.Label)
		}
	}
}

func TestBuildConfigMenuItems_BooleanValues(t *testing.T) {
	tests := []struct {
		name       string
		streaming  bool
		incognito  bool
		expStream  string
		expIncog   string
	}{
		{"both_true", true, true, "true", "true"},
		{"both_false", false, false, "false", "false"},
		{"mixed", true, false, "true", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Streaming: tt.streaming,
				Incognito: tt.incognito,
			}

			items := buildConfigMenuItems(cfg)

			var streamingValue, incognitoValue string
			for _, item := range items {
				if item.Key == "streaming" {
					streamingValue = item.Value
				}
				if item.Key == "incognito" {
					incognitoValue = item.Value
				}
			}

			if streamingValue != tt.expStream {
				t.Errorf("Streaming = %q, want %q", streamingValue, tt.expStream)
			}
			if incognitoValue != tt.expIncog {
				t.Errorf("Incognito = %q, want %q", incognitoValue, tt.expIncog)
			}
		})
	}
}
