package main

import (
	"fmt"

	"github.com/diogo/perplexity-go/pkg/models"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `View and modify perplexity CLI configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		render.RenderTitle("Configuration")

		fmt.Printf("Config file: %s\n\n", cfgMgr.GetConfigFile())
		fmt.Printf("default_model:    %s\n", cfg.DefaultModel)
		fmt.Printf("default_mode:     %s\n", cfg.DefaultMode)
		fmt.Printf("default_language: %s\n", cfg.DefaultLanguage)
		fmt.Printf("default_sources:  %v\n", cfg.DefaultSources)
		fmt.Printf("streaming:        %v\n", cfg.Streaming)
		fmt.Printf("incognito:        %v\n", cfg.Incognito)
		fmt.Printf("cookie_file:      %s\n", cfg.CookieFile)
		fmt.Printf("history_file:     %s\n", cfg.HistoryFile)

		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		switch key {
		case "default_model":
			model := models.Model(value)
			if !models.IsValidModel(model) {
				return fmt.Errorf("invalid model: %s", value)
			}
			cfg.DefaultModel = model

		case "default_mode":
			mode := models.Mode(value)
			if !models.IsValidMode(mode) {
				return fmt.Errorf("invalid mode: %s", value)
			}
			cfg.DefaultMode = mode

		case "default_language":
			cfg.DefaultLanguage = value

		case "streaming":
			cfg.Streaming = value == "true" || value == "1" || value == "yes"

		case "incognito":
			cfg.Incognito = value == "true" || value == "1" || value == "yes"

		case "cookie_file":
			cfg.CookieFile = value

		case "history_file":
			cfg.HistoryFile = value

		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := cfgMgr.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %v", err)
		}

		render.RenderSuccess(fmt.Sprintf("Set %s = %s", key, value))
		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.DefaultModel = models.ModelPplxPro
		cfg.DefaultMode = models.ModeDefault
		cfg.DefaultLanguage = "en-US"
		cfg.DefaultSources = []models.Source{models.SourceWeb}
		cfg.Streaming = true
		cfg.Incognito = false

		if err := cfgMgr.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %v", err)
		}

		render.RenderSuccess("Configuration reset to defaults")
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(cfgMgr.GetConfigFile())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configPathCmd)
}
