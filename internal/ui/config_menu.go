package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/diogo/perplexity-go/internal/config"
	"github.com/diogo/perplexity-go/pkg/models"
)

// customKeyMap returns a keymap that includes ESC as a quit key.
func customKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "back"),
	)
	return km
}

// ConfigMenuItem represents a configuration option in the menu.
type ConfigMenuItem struct {
	Key         string
	Label       string
	Description string
	Value       string
}

// RunInteractiveConfig displays an interactive configuration menu.
func RunInteractiveConfig(cfg *config.Config, cfgMgr *config.Manager) error {
	for {
		// Build menu items with current values
		items := buildConfigMenuItems(cfg)

		// Create options for the select menu
		options := make([]huh.Option[string], len(items)+2)
		for i, item := range items {
			label := fmt.Sprintf("%-18s %s", item.Label, DimStyle.Render(item.Value))
			options[i] = huh.NewOption(label, item.Key)
		}
		options[len(items)] = huh.NewOption(SuccessStyle.Render("Save and exit"), "save")
		options[len(items)+1] = huh.NewOption(WarningStyle.Render("Reset to defaults"), "reset")

		var selected string
		selectForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Configuration").
					Description("Select an option to modify").
					Options(options...).
					Value(&selected),
			),
		)

		if err := selectForm.Run(); err != nil {
			if err == huh.ErrUserAborted {
				return nil // User pressed ESC, exit the loop to return
			}
			return err
		}

		switch selected {
		case "save":
			if err := cfgMgr.Save(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			fmt.Println(SuccessStyle.Render("Configuration saved!"))
			return nil

		case "reset":
			if err := handleReset(cfg); err != nil {
				return err
			}

		default:
			if err := handleConfigEdit(cfg, selected); err != nil {
				return err
			}
		}
	}
}

func buildConfigMenuItems(cfg *config.Config) []ConfigMenuItem {
	sources := make([]string, len(cfg.DefaultSources))
	for i, s := range cfg.DefaultSources {
		sources[i] = string(s)
	}

	return []ConfigMenuItem{
		{
			Key:         "default_model",
			Label:       "Model",
			Description: "Default AI model",
			Value:       string(cfg.DefaultModel),
		},
		{
			Key:         "default_mode",
			Label:       "Mode",
			Description: "Default search mode",
			Value:       string(cfg.DefaultMode),
		},
		{
			Key:         "default_language",
			Label:       "Language",
			Description: "Response language (e.g., en-US)",
			Value:       cfg.DefaultLanguage,
		},
		{
			Key:         "default_sources",
			Label:       "Sources",
			Description: "Search sources",
			Value:       strings.Join(sources, ", "),
		},
		{
			Key:         "streaming",
			Label:       "Streaming",
			Description: "Enable streaming output",
			Value:       fmt.Sprintf("%v", cfg.Streaming),
		},
		{
			Key:         "incognito",
			Label:       "Incognito",
			Description: "Don't save to history",
			Value:       fmt.Sprintf("%v", cfg.Incognito),
		},
		{
			Key:         "cookie_file",
			Label:       "Cookie file",
			Description: "Path to cookies file",
			Value:       cfg.CookieFile,
		},
		{
			Key:         "history_file",
			Label:       "History file",
			Description: "Path to history file",
			Value:       cfg.HistoryFile,
		},
	}
}

func handleConfigEdit(cfg *config.Config, key string) error {
	switch key {
	case "default_model":
		return editModel(cfg)
	case "default_mode":
		return editMode(cfg)
	case "default_language":
		return editLanguage(cfg)
	case "default_sources":
		return editSources(cfg)
	case "streaming":
		return editBool(cfg, "streaming", "Enable streaming output?", &cfg.Streaming)
	case "incognito":
		return editBool(cfg, "incognito", "Enable incognito mode?", &cfg.Incognito)
	case "cookie_file":
		return editString("Cookie file path", &cfg.CookieFile)
	case "history_file":
		return editString("History file path", &cfg.HistoryFile)
	}
	return nil
}

func editModel(cfg *config.Config) error {
	options := make([]huh.Option[string], len(models.AvailableModels))
	for i, m := range models.AvailableModels {
		options[i] = huh.NewOption(string(m), string(m))
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Model").
				Description("Choose the default AI model (Esc to go back)").
				Options(options...).
				Value(&selected),
		),
	).WithKeyMap(customKeyMap())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil
		}
		return err
	}

	cfg.DefaultModel = models.Model(selected)
	return nil
}

func editMode(cfg *config.Config) error {
	modes := []models.Mode{
		models.ModeFast,
		models.ModePro,
		models.ModeReasoning,
		models.ModeDeepResearch,
		models.ModeDefault,
	}

	options := make([]huh.Option[string], len(modes))
	for i, m := range modes {
		desc := getModeDescription(m)
		options[i] = huh.NewOption(fmt.Sprintf("%-15s %s", string(m), DimStyle.Render(desc)), string(m))
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Mode").
				Description("Choose the default search mode (Esc to go back)").
				Options(options...).
				Value(&selected),
		),
	).WithKeyMap(customKeyMap())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil
		}
		return err
	}

	cfg.DefaultMode = models.Mode(selected)
	return nil
}

func getModeDescription(m models.Mode) string {
	switch m {
	case models.ModeFast:
		return "Quick responses"
	case models.ModePro:
		return "Balanced quality"
	case models.ModeReasoning:
		return "Deep analysis"
	case models.ModeDeepResearch:
		return "Comprehensive research"
	case models.ModeDefault:
		return "Standard mode"
	default:
		return ""
	}
}

func editLanguage(cfg *config.Config) error {
	commonLanguages := []struct {
		code string
		name string
	}{
		{"en-US", "English (US)"},
		{"en-GB", "English (UK)"},
		{"pt-BR", "Portuguese (Brazil)"},
		{"pt-PT", "Portuguese (Portugal)"},
		{"es-ES", "Spanish (Spain)"},
		{"es-MX", "Spanish (Mexico)"},
		{"fr-FR", "French"},
		{"de-DE", "German"},
		{"it-IT", "Italian"},
		{"ja-JP", "Japanese"},
		{"ko-KR", "Korean"},
		{"zh-CN", "Chinese (Simplified)"},
		{"zh-TW", "Chinese (Traditional)"},
	}

	options := make([]huh.Option[string], len(commonLanguages)+1)
	for i, lang := range commonLanguages {
		options[i] = huh.NewOption(fmt.Sprintf("%-7s %s", lang.code, lang.name), lang.code)
	}
	options[len(commonLanguages)] = huh.NewOption("Other (enter custom)", "custom")

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select Language").
				Description("Choose the response language (Esc to go back)").
				Options(options...).
				Value(&selected),
		),
	).WithKeyMap(customKeyMap())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil
		}
		return err
	}

	if selected == "custom" {
		var customLang string
		inputForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Custom Language").
					Description("Enter language code, e.g., en-US (Esc to go back)").
					Placeholder("xx-XX").
					Value(&customLang).
					Validate(func(s string) error {
						if len(s) != 5 || s[2] != '-' {
							return fmt.Errorf("invalid format, use xx-XX")
						}
						return nil
					}),
			),
		).WithKeyMap(customKeyMap())

		if err := inputForm.Run(); err != nil {
			if err == huh.ErrUserAborted {
				return nil
			}
			return err
		}
		cfg.DefaultLanguage = customLang
	} else {
		cfg.DefaultLanguage = selected
	}

	return nil
}

func editSources(cfg *config.Config) error {
	currentSources := make(map[models.Source]bool)
	for _, s := range cfg.DefaultSources {
		currentSources[s] = true
	}

	var selected []string
	options := make([]huh.Option[string], len(models.AvailableSources))
	for i, s := range models.AvailableSources {
		options[i] = huh.NewOption(string(s), string(s))
	}

	// Pre-select current sources
	for _, s := range cfg.DefaultSources {
		selected = append(selected, string(s))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Sources").
				Description("Choose search sources, at least one (Esc to go back)").
				Options(options...).
				Value(&selected).
				Validate(func(s []string) error {
					if len(s) == 0 {
						return fmt.Errorf("select at least one source")
					}
					return nil
				}),
		),
	).WithKeyMap(customKeyMap())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil
		}
		return err
	}

	cfg.DefaultSources = make([]models.Source, len(selected))
	for i, s := range selected {
		cfg.DefaultSources[i] = models.Source(s)
	}

	return nil
}

func editBool(cfg *config.Config, name, title string, value *bool) error {
	options := []huh.Option[bool]{
		huh.NewOption("Yes", true),
		huh.NewOption("No", false),
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title(title + " (Esc to go back)").
				Options(options...).
				Value(value),
		),
	).WithKeyMap(customKeyMap())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil
		}
		return err
	}

	return nil
}

func editString(title string, value *string) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title + " (Esc to go back)").
				Value(value),
		),
	).WithKeyMap(customKeyMap())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil
		}
		return err
	}

	return nil
}

func handleReset(cfg *config.Config) error {
	var confirm bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Reset Configuration (Esc to go back)").
				Description("Are you sure you want to reset all settings to defaults?").
				Affirmative("Yes, reset").
				Negative("Cancel").
				Value(&confirm),
		),
	).WithKeyMap(customKeyMap())

	if err := form.Run(); err != nil {
		if err == huh.ErrUserAborted {
			return nil
		}
		return err
	}

	if confirm {
		cfg.DefaultModel = models.ModelPplxPro
		cfg.DefaultMode = models.ModeDefault
		cfg.DefaultLanguage = "en-US"
		cfg.DefaultSources = []models.Source{models.SourceWeb}
		cfg.Streaming = true
		cfg.Incognito = false
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("Configuration reset to defaults"))
	}

	return nil
}
