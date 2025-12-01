package main

import (
	"fmt"

	"github.com/diogo/perplexity-go/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration interactively",
	Long: `Open an interactive menu to view and modify configuration settings.

Use arrow keys to navigate, Enter to select an option, and Esc to cancel.

Configuration options:
  - Model:     Default AI model (pplx_pro, gpt5, claude45sonnet, etc.)
  - Mode:      Search mode (fast, pro, reasoning, deep-research, default)
  - Language:  Response language (e.g., en-US, pt-BR)
  - Sources:   Search sources (web, scholar, social)
  - Streaming: Enable/disable streaming output
  - Incognito: Enable/disable history saving`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return ui.RunInteractiveConfig(cfg, cfgMgr)
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
	configCmd.AddCommand(configPathCmd)
}
