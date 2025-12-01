package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/diogo/perplexity-go/internal/config"
	"github.com/diogo/perplexity-go/internal/ui"
)

func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Store original values
	originalCfg := cfg
	originalCfgMgr := cfgMgr
	originalRender := render

	// Setup test config
	var err error
	cfgMgr, err = config.NewManager()
	if err != nil {
		t.Fatalf("Failed to create config manager: %v", err)
	}

	cfg = &config.Config{
		CookieFile:  filepath.Join(tmpDir, "cookies.json"),
		HistoryFile: filepath.Join(tmpDir, "history.jsonl"),
	}

	render, err = ui.NewRenderer()
	if err != nil {
		t.Fatalf("Failed to create renderer: %v", err)
	}

	cleanup := func() {
		cfg = originalCfg
		cfgMgr = originalCfgMgr
		render = originalRender
	}

	return tmpDir, cleanup
}

func TestImportCookiesCmd_FileNotFound(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Call RunE directly instead of Execute
	err := importCookiesCmd.RunE(importCookiesCmd, []string{"/nonexistent/file.json"})

	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("Error should mention 'file not found', got: %v", err)
	}
}

func TestImportCookiesCmd_ValidJSON(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test cookie file to import
	cookieJSON := `[
		{
			"name": "imported_cookie",
			"value": "imported_value",
			"domain": ".perplexity.ai",
			"path": "/"
		},
		{
			"name": "next-auth.csrf-token",
			"value": "csrf|hash",
			"domain": ".perplexity.ai",
			"path": "/"
		}
	]`
	importFile := filepath.Join(tmpDir, "import_cookies.json")
	if err := os.WriteFile(importFile, []byte(cookieJSON), 0644); err != nil {
		t.Fatalf("Failed to write import file: %v", err)
	}

	err := importCookiesCmd.RunE(importCookiesCmd, []string{importFile})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify cookies were saved to config location
	if _, err := os.Stat(cfg.CookieFile); os.IsNotExist(err) {
		t.Error("Cookie file was not created at config location")
	}
}

func TestImportCookiesCmd_NoCookies(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create file with only non-perplexity cookies
	cookieJSON := `[
		{
			"name": "other_cookie",
			"value": "value",
			"domain": ".example.com",
			"path": "/"
		}
	]`
	emptyCookieFile := filepath.Join(tmpDir, "empty.json")
	if err := os.WriteFile(emptyCookieFile, []byte(cookieJSON), 0644); err != nil {
		t.Fatalf("Failed to write empty cookie file: %v", err)
	}

	err := importCookiesCmd.RunE(importCookiesCmd, []string{emptyCookieFile})

	if err == nil {
		t.Error("Expected error for empty cookies")
	}
	if !strings.Contains(err.Error(), "no Perplexity cookies") {
		t.Errorf("Error should mention 'no Perplexity cookies', got: %v", err)
	}
}

func TestImportCookiesCmd_InvalidFormat(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create invalid format file that doesn't look like any valid format
	invalidFile := filepath.Join(tmpDir, "invalid.txt")
	if err := os.WriteFile(invalidFile, []byte("not valid json or netscape"), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	err := importCookiesCmd.RunE(importCookiesCmd, []string{invalidFile})

	if err == nil {
		t.Error("Expected error for invalid format")
	}
	// The error can be either "failed to parse" or "no Perplexity cookies" depending on parsing
	if !strings.Contains(err.Error(), "no Perplexity cookies") && !strings.Contains(err.Error(), "failed to parse cookies") {
		t.Errorf("Error should mention 'no Perplexity cookies' or 'failed to parse cookies', got: %v", err)
	}
}

func TestCookiesStatusCmd_NoCookies(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Make sure cookie file doesn't exist
	cfg.CookieFile = "/nonexistent/cookies.json"

	err := cookiesStatusCmd.RunE(cookiesStatusCmd, []string{})

	// Should not error, just show warning
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCookiesStatusCmd_ValidCookies(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create cookie file with CSRF token
	cookieJSON := `[
		{
			"name": "next-auth.csrf-token",
			"value": "token|hash",
			"domain": ".perplexity.ai",
			"path": "/"
		}
	]`
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	if err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644); err != nil {
		t.Fatalf("Failed to write cookie file: %v", err)
	}
	cfg.CookieFile = cookieFile

	err := cookiesStatusCmd.RunE(cookiesStatusCmd, []string{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCookiesStatusCmd_MissingCSRFToken(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create cookie file without CSRF token
	cookieJSON := `[
		{
			"name": "other_cookie",
			"value": "value",
			"domain": ".perplexity.ai",
			"path": "/"
		}
	]`
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	if err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644); err != nil {
		t.Fatalf("Failed to write cookie file: %v", err)
	}
	cfg.CookieFile = cookieFile

	err := cookiesStatusCmd.RunE(cookiesStatusCmd, []string{})

	// Should not error, just show warning about missing CSRF token
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCookiesClearCmd_NoCookies(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Make sure cookie file doesn't exist
	cfg.CookieFile = "/nonexistent/cookies.json"

	err := cookiesClearCmd.RunE(cookiesClearCmd, []string{})

	// Should not error when no cookies exist
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCookiesClearCmd_ClearExisting(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create cookie file
	cookieFile := filepath.Join(tmpDir, "cookies.json")
	if err := os.WriteFile(cookieFile, []byte("[]"), 0644); err != nil {
		t.Fatalf("Failed to write cookie file: %v", err)
	}
	cfg.CookieFile = cookieFile

	// Verify file exists
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		t.Fatal("Cookie file should exist before clear")
	}

	err := cookiesClearCmd.RunE(cookiesClearCmd, []string{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(cookieFile); !os.IsNotExist(err) {
		t.Error("Cookie file should be deleted after clear")
	}
}

func TestCookiesPathCmd(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	err := cookiesPathCmd.RunE(cookiesPathCmd, []string{})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCookiesCmdStructure(t *testing.T) {
	// Verify command structure (import was moved to root level as import-cookies)
	subcommands := []string{"status", "clear", "path"}

	for _, name := range subcommands {
		found := false
		for _, cmd := range cookiesCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %q not found", name)
		}
	}
}

func TestImportCookiesCmd_NetscapeFormat(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create Netscape format cookie file
	netscapeContent := `# Netscape HTTP Cookie File
.perplexity.ai	TRUE	/	TRUE	1735689600	session	session_value
.perplexity.ai	TRUE	/	FALSE	0	next-auth.csrf-token	token|hash
`
	importFile := filepath.Join(tmpDir, "cookies.txt")
	if err := os.WriteFile(importFile, []byte(netscapeContent), 0644); err != nil {
		t.Fatalf("Failed to write Netscape file: %v", err)
	}

	err := importCookiesCmd.RunE(importCookiesCmd, []string{importFile})

	if err != nil {
		t.Errorf("Unexpected error importing Netscape format: %v", err)
	}

	// Verify cookies were saved
	if _, err := os.Stat(cfg.CookieFile); os.IsNotExist(err) {
		t.Error("Cookie file was not created")
	}
}

func TestImportCookiesCmd_MissingCSRFToken(t *testing.T) {
	tmpDir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create cookie file without CSRF token
	cookieJSON := `[
		{
			"name": "some_cookie",
			"value": "some_value",
			"domain": ".perplexity.ai",
			"path": "/"
		}
	]`
	importFile := filepath.Join(tmpDir, "no_csrf.json")
	if err := os.WriteFile(importFile, []byte(cookieJSON), 0644); err != nil {
		t.Fatalf("Failed to write import file: %v", err)
	}

	// Should succeed but show warning (warning is just printed, not returned as error)
	err := importCookiesCmd.RunE(importCookiesCmd, []string{importFile})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify cookies were still saved
	if _, err := os.Stat(cfg.CookieFile); os.IsNotExist(err) {
		t.Error("Cookie file was not created despite missing CSRF token")
	}
}

func TestCookiesCmd_Help(t *testing.T) {
	// Test that help text exists
	if cookiesCmd.Short == "" {
		t.Error("cookiesCmd should have a short description")
	}
	if cookiesCmd.Long == "" {
		t.Error("cookiesCmd should have a long description")
	}
}

func TestImportCookiesCmd_Help(t *testing.T) {
	// Test that help text exists
	if importCookiesCmd.Short == "" {
		t.Error("importCookiesCmd should have a short description")
	}
	if importCookiesCmd.Long == "" {
		t.Error("importCookiesCmd should have a long description")
	}
}
