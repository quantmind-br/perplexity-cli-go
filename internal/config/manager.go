// Package config handles configuration management.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/diogo/perplexity-go/pkg/models"
	"github.com/spf13/viper"
)

const (
	configDirName  = ".perplexity-cli"
	configFileName = "config"
	configFileType = "json"
)

// Config holds all configuration options.
type Config struct {
	DefaultModel    models.Model    `mapstructure:"default_model"`
	DefaultMode     models.Mode     `mapstructure:"default_mode"`
	DefaultLanguage string          `mapstructure:"default_language"`
	DefaultSources  []models.Source `mapstructure:"default_sources"`
	Streaming       bool            `mapstructure:"streaming"`
	Incognito       bool            `mapstructure:"incognito"`
	CookieFile      string          `mapstructure:"cookie_file"`
	HistoryFile     string          `mapstructure:"history_file"`
}

// Manager handles configuration loading and saving.
type Manager struct {
	v       *viper.Viper
	cfgDir  string
	cfgFile string
}

// NewManager creates a new configuration manager.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cfgDir := filepath.Join(home, configDirName)
	cfgFile := filepath.Join(cfgDir, configFileName+"."+configFileType)

	m := &Manager{
		v:       viper.New(),
		cfgDir:  cfgDir,
		cfgFile: cfgFile,
	}

	// Set defaults
	m.setDefaults()

	// Setup viper
	m.v.SetConfigName(configFileName)
	m.v.SetConfigType(configFileType)
	m.v.AddConfigPath(cfgDir)
	m.v.AddConfigPath(".")

	// Environment variable support
	m.v.SetEnvPrefix("PERPLEXITY")
	m.v.AutomaticEnv()
	m.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return m, nil
}

// setDefaults sets default configuration values.
func (m *Manager) setDefaults() {
	m.v.SetDefault("default_model", string(models.ModelPplxPro))
	m.v.SetDefault("default_mode", string(models.ModeDefault))
	m.v.SetDefault("default_language", "en-US")
	m.v.SetDefault("default_sources", []string{string(models.SourceWeb)})
	m.v.SetDefault("streaming", true)
	m.v.SetDefault("incognito", false)
	m.v.SetDefault("cookie_file", filepath.Join(m.cfgDir, "cookies.json"))
	m.v.SetDefault("history_file", filepath.Join(m.cfgDir, "history.jsonl"))
}

// Load reads configuration from file and environment.
func (m *Manager) Load() (*Config, error) {
	// Create config directory if not exists
	if err := os.MkdirAll(m.cfgDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Try to read config file (ignore if not exists)
	if err := m.v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only return error if it's not "file not found"
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	cfg := &Config{}

	// Manual parsing to handle type conversions
	cfg.DefaultModel = models.Model(m.v.GetString("default_model"))
	cfg.DefaultMode = models.Mode(m.v.GetString("default_mode"))
	cfg.DefaultLanguage = m.v.GetString("default_language")
	cfg.Streaming = m.v.GetBool("streaming")
	cfg.Incognito = m.v.GetBool("incognito")
	cfg.CookieFile = m.v.GetString("cookie_file")
	cfg.HistoryFile = m.v.GetString("history_file")

	// Parse sources
	sourcesRaw := m.v.GetStringSlice("default_sources")
	if len(sourcesRaw) == 0 {
		// Try as comma-separated string
		sourcesStr := m.v.GetString("default_sources")
		if sourcesStr != "" {
			sourcesRaw = strings.Split(sourcesStr, ",")
		}
	}
	cfg.DefaultSources = parseSources(sourcesRaw)

	// Validate configuration
	if err := m.validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to file.
func (m *Manager) Save(cfg *Config) error {
	// Create config directory if not exists
	if err := os.MkdirAll(m.cfgDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	m.v.Set("default_model", string(cfg.DefaultModel))
	m.v.Set("default_mode", string(cfg.DefaultMode))
	m.v.Set("default_language", cfg.DefaultLanguage)
	m.v.Set("streaming", cfg.Streaming)
	m.v.Set("incognito", cfg.Incognito)
	m.v.Set("cookie_file", cfg.CookieFile)
	m.v.Set("history_file", cfg.HistoryFile)

	sources := make([]string, len(cfg.DefaultSources))
	for i, s := range cfg.DefaultSources {
		sources[i] = string(s)
	}
	m.v.Set("default_sources", sources)

	return m.v.WriteConfigAs(m.cfgFile)
}

// validate checks configuration values.
func (m *Manager) validate(cfg *Config) error {
	// Validate model
	if cfg.DefaultModel != "" && !models.IsValidModel(cfg.DefaultModel) {
		return fmt.Errorf("invalid model: %s", cfg.DefaultModel)
	}

	// Validate mode
	if cfg.DefaultMode != "" && !models.IsValidMode(cfg.DefaultMode) {
		return fmt.Errorf("invalid mode: %s", cfg.DefaultMode)
	}

	// Validate language format (xx-XX)
	if cfg.DefaultLanguage != "" && !isValidLanguage(cfg.DefaultLanguage) {
		return fmt.Errorf("invalid language format: %s (expected xx-XX)", cfg.DefaultLanguage)
	}

	// Validate sources
	for _, s := range cfg.DefaultSources {
		if !models.IsValidSource(s) {
			return fmt.Errorf("invalid source: %s", s)
		}
	}

	return nil
}

// GetConfigDir returns the configuration directory path.
func (m *Manager) GetConfigDir() string {
	return m.cfgDir
}

// GetConfigFile returns the configuration file path.
func (m *Manager) GetConfigFile() string {
	return m.cfgFile
}

// parseSources converts string slice to Source slice.
func parseSources(raw []string) []models.Source {
	sources := make([]models.Source, 0, len(raw))
	seen := make(map[models.Source]bool)

	for _, s := range raw {
		s = strings.TrimSpace(s)
		source := models.Source(s)
		if models.IsValidSource(source) && !seen[source] {
			sources = append(sources, source)
			seen[source] = true
		}
	}

	if len(sources) == 0 {
		return []models.Source{models.SourceWeb}
	}

	return sources
}

// isValidLanguage checks if the language format is valid (xx-XX).
var languageRegex = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)

func isValidLanguage(lang string) bool {
	return languageRegex.MatchString(lang)
}

// ParseBoolean parses boolean strings (true, false, 1, 0, yes, no, on, off).
func ParseBoolean(value string, defaultValue bool) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}
