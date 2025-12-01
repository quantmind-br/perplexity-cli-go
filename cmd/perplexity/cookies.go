package main

import (
	"fmt"
	"os"

	"github.com/diogo/perplexity-go/internal/auth"
	"github.com/spf13/cobra"
)

var cookiesCmd = &cobra.Command{
	Use:   "cookies",
	Short: "Manage authentication cookies",
	Long:  `Manage authentication cookies for Perplexity API access.`,
}

// importCookiesCmd is a root-level command for importing cookies.
var importCookiesCmd = &cobra.Command{
	Use:   "import-cookies <file>",
	Short: "Import cookies from file",
	Long: `Import cookies from a JSON or Netscape format file.

Supported formats:
  - JSON: Browser extension export (cookies.json)
  - Netscape: curl/wget format (cookies.txt)

Example:
  perplexity import-cookies ~/Downloads/cookies.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		importFile := args[0]

		// Check if source file exists
		if _, err := os.Stat(importFile); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", importFile)
		}

		// Try JSON format first
		cookies, err := auth.LoadCookiesFromFile(importFile)
		if err != nil {
			// Try Netscape format
			cookies, err = auth.LoadCookiesFromNetscape(importFile)
			if err != nil {
				return fmt.Errorf("failed to parse cookies (tried JSON and Netscape formats): %v", err)
			}
		}

		if len(cookies) == 0 {
			return fmt.Errorf("no Perplexity cookies found in file")
		}

		// Save to config cookie file
		if err := auth.SaveCookiesToFile(cookies, cfg.CookieFile); err != nil {
			return fmt.Errorf("failed to save cookies: %v", err)
		}

		render.RenderSuccess(fmt.Sprintf("Imported %d cookies to %s", len(cookies), cfg.CookieFile))

		if !auth.HasCSRFToken(cookies) {
			render.RenderWarning("CSRF token not found - session may not work")
		}

		return nil
	},
}

var cookiesStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cookieFile := cfg.CookieFile
		if flagCookieFile != "" {
			cookieFile = flagCookieFile
		}

		// Check if file exists
		if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
			render.RenderWarning("Not authenticated")
			render.RenderInfo(fmt.Sprintf("Cookie file not found: %s", cookieFile))
			render.RenderInfo("Run 'perplexity import-cookies <file>' to import cookies")
			return nil
		}

		// Load and validate cookies
		cookies, err := auth.LoadCookiesFromFile(cookieFile)
		if err != nil {
			render.RenderError(fmt.Errorf("failed to load cookies: %v", err))
			return err
		}

		if len(cookies) == 0 {
			render.RenderWarning("No Perplexity cookies found in file")
			return nil
		}

		// Check for CSRF token
		if auth.HasCSRFToken(cookies) {
			render.RenderSuccess("Authenticated")
			fmt.Printf("Cookie file: %s\n", cookieFile)
			fmt.Printf("Cookies loaded: %d\n", len(cookies))

			// Show cookie names
			fmt.Println("\nCookies:")
			for _, c := range cookies {
				fmt.Printf("  - %s\n", c.Name)
			}
		} else {
			render.RenderWarning("Cookies found but CSRF token missing")
			render.RenderInfo("Session may be expired. Re-export cookies from browser.")
		}

		return nil
	},
}

var cookiesClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear saved cookies",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.Remove(cfg.CookieFile); err != nil {
			if os.IsNotExist(err) {
				render.RenderInfo("No cookies to clear")
				return nil
			}
			return fmt.Errorf("failed to remove cookies: %v", err)
		}

		render.RenderSuccess("Cookies cleared")
		return nil
	},
}

var cookiesPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show cookie file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(cfg.CookieFile)
		return nil
	},
}

func init() {
	cookiesCmd.AddCommand(cookiesStatusCmd)
	cookiesCmd.AddCommand(cookiesClearCmd)
	cookiesCmd.AddCommand(cookiesPathCmd)

	// NOTE: importCookiesCmd is added to the rootCmd in root.go
}
