package config

import (
	"testing"

	"github.com/diogo/perplexity-go/pkg/models"
)

func TestParseBoolean(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		defaultValue bool
		want         bool
	}{
		{"true lowercase", "true", false, true},
		{"TRUE uppercase", "TRUE", false, true},
		{"True mixed", "True", false, true},
		{"1", "1", false, true},
		{"yes", "yes", false, true},
		{"YES", "YES", false, true},
		{"on", "on", false, true},
		{"ON", "ON", false, true},
		{"false lowercase", "false", true, false},
		{"FALSE uppercase", "FALSE", true, false},
		{"0", "0", true, false},
		{"no", "no", true, false},
		{"off", "off", true, false},
		{"invalid with default true", "invalid", true, true},
		{"invalid with default false", "invalid", false, false},
		{"empty with default true", "", true, true},
		{"empty with default false", "", false, false},
		{"whitespace", "  true  ", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseBoolean(tt.value, tt.defaultValue)
			if got != tt.want {
				t.Errorf("ParseBoolean(%q, %v) = %v, want %v", tt.value, tt.defaultValue, got, tt.want)
			}
		})
	}
}

func TestIsValidLanguage(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want bool
	}{
		{"valid en-US", "en-US", true},
		{"valid pt-BR", "pt-BR", true},
		{"valid fr-FR", "fr-FR", true},
		{"valid de-DE", "de-DE", true},
		{"valid es-ES", "es-ES", true},
		{"invalid underscore", "en_US", false},
		{"invalid lowercase region", "en-us", false},
		{"invalid uppercase lang", "EN-US", false},
		{"invalid no region", "en", false},
		{"invalid too short", "e-US", false},
		{"invalid too long", "eng-USA", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidLanguage(tt.lang)
			if got != tt.want {
				t.Errorf("isValidLanguage(%q) = %v, want %v", tt.lang, got, tt.want)
			}
		})
	}
}

func TestParseSources(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
		want []models.Source
	}{
		{
			name: "single web",
			raw:  []string{"web"},
			want: []models.Source{models.SourceWeb},
		},
		{
			name: "multiple sources",
			raw:  []string{"web", "scholar", "social"},
			want: []models.Source{models.SourceWeb, models.SourceScholar, models.SourceSocial},
		},
		{
			name: "with whitespace",
			raw:  []string{"  web  ", " scholar "},
			want: []models.Source{models.SourceWeb, models.SourceScholar},
		},
		{
			name: "with duplicates",
			raw:  []string{"web", "web", "scholar"},
			want: []models.Source{models.SourceWeb, models.SourceScholar},
		},
		{
			name: "empty returns default",
			raw:  []string{},
			want: []models.Source{models.SourceWeb},
		},
		{
			name: "invalid sources filtered",
			raw:  []string{"web", "invalid", "scholar"},
			want: []models.Source{models.SourceWeb, models.SourceScholar},
		},
		{
			name: "all invalid returns default",
			raw:  []string{"invalid1", "invalid2"},
			want: []models.Source{models.SourceWeb},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSources(tt.raw)
			if len(got) != len(tt.want) {
				t.Errorf("parseSources() returned %d sources, want %d", len(got), len(tt.want))
				return
			}
			for i, s := range got {
				if s != tt.want[i] {
					t.Errorf("parseSources()[%d] = %q, want %q", i, s, tt.want[i])
				}
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if mgr.cfgDir == "" {
		t.Error("cfgDir should not be empty")
	}
	if mgr.cfgFile == "" {
		t.Error("cfgFile should not be empty")
	}
	if mgr.v == nil {
		t.Error("viper instance should not be nil")
	}
}

func TestManagerGetPaths(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	dir := mgr.GetConfigDir()
	if dir == "" {
		t.Error("GetConfigDir() returned empty string")
	}

	file := mgr.GetConfigFile()
	if file == "" {
		t.Error("GetConfigFile() returned empty string")
	}
}

func TestManagerLoad(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Load should work even without config file (uses defaults)
	cfg, err := mgr.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DefaultModel == "" {
		t.Error("DefaultModel should have a default value")
	}
	if cfg.DefaultMode == "" {
		t.Error("DefaultMode should have a default value")
	}
	if cfg.DefaultLanguage == "" {
		t.Error("DefaultLanguage should have a default value")
	}
}

func TestManagerValidate(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				DefaultModel:    models.ModelPplxPro,
				DefaultMode:     models.ModeDefault,
				DefaultLanguage: "en-US",
				DefaultSources:  []models.Source{models.SourceWeb},
			},
			wantErr: false,
		},
		{
			name: "invalid model",
			cfg: &Config{
				DefaultModel: models.Model("invalid_model"),
			},
			wantErr: true,
		},
		{
			name: "invalid mode",
			cfg: &Config{
				DefaultMode: models.Mode("invalid_mode"),
			},
			wantErr: true,
		},
		{
			name: "invalid language format",
			cfg: &Config{
				DefaultLanguage: "en_US", // underscore instead of dash
			},
			wantErr: true,
		},
		{
			name: "invalid source",
			cfg: &Config{
				DefaultSources: []models.Source{models.Source("invalid_source")},
			},
			wantErr: true,
		},
		{
			name: "empty config (uses defaults)",
			cfg: &Config{
				DefaultModel:   "",
				DefaultMode:    "",
				DefaultSources: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.validate(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
