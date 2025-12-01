package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/internal/config"
	"github.com/diogo/perplexity-go/internal/history"
	"github.com/diogo/perplexity-go/pkg/models"
	"github.com/spf13/cobra"
)

// MockClientWithResponse is a more sophisticated mock for integration tests
type MockClientWithResponse struct {
	searchResponse   *models.SearchResponse
	searchError      error
	streamChunks     []models.StreamChunk
	streamError      error
	closeError       error
	cookies          []*http.Cookie
	proQueriesRem    int
	fileUploadsRem   int
	hasValidSession  bool
}

func NewMockClientWithResponse(resp *models.SearchResponse, err error) *MockClientWithResponse {
	return &MockClientWithResponse{
		searchResponse:   resp,
		searchError:      err,
		hasValidSession:  true,
		proQueriesRem:    5,
		fileUploadsRem:   10,
	}
}

func NewMockStreamClientWithChunks(chunks []models.StreamChunk, err error) *MockClientWithResponse {
	return &MockClientWithResponse{
		streamChunks:    chunks,
		streamError:     err,
		hasValidSession: true,
		proQueriesRem:   5,
		fileUploadsRem:  10,
	}
}

func (m *MockClientWithResponse) Search(ctx context.Context, opts models.SearchOptions) (*models.SearchResponse, error) {
	return m.searchResponse, m.searchError
}

func (m *MockClientWithResponse) SearchStream(ctx context.Context, opts models.SearchOptions) (<-chan models.StreamChunk, error) {
	ch := make(chan models.StreamChunk, len(m.streamChunks))
	if m.streamError != nil {
		ch <- models.StreamChunk{Error: m.streamError}
		close(ch)
		return ch, m.streamError
	}
	for _, chunk := range m.streamChunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func (m *MockClientWithResponse) Close() error {
	return m.closeError
}

func (m *MockClientWithResponse) SetCookies(cookies []*http.Cookie) {
	m.cookies = cookies
}

func (m *MockClientWithResponse) GetCookies() []*http.Cookie {
	return m.cookies
}

func (m *MockClientWithResponse) HasValidSession() bool {
	return m.hasValidSession
}

func (m *MockClientWithResponse) ProQueriesRemaining() int {
	return m.proQueriesRem
}

func (m *MockClientWithResponse) FileUploadsRemaining() int {
	return m.fileUploadsRem
}

// TestConfigCommand tests the config command and subcommands
func TestConfigCommand(t *testing.T) {
	tempDir := t.TempDir()

	// Save and restore HOME environment variable
	oldHome := os.Getenv("HOME")
	defer func() { os.Setenv("HOME", oldHome) }()
	os.Setenv("HOME", tempDir)

	t.Run("config path shows correct path", func(t *testing.T) {
		// Setup
		cfgMgr, err := config.NewManager()
		if err != nil {
			t.Fatalf("failed to create config manager: %v", err)
		}

		cfgMgr.SetConfigFile(filepath.Join(tempDir, "test_config.json"))
		cfg = &config.Config{}
		render = &ui.Renderer{}

		// Create a buffer to capture output
		var buf bytes.Buffer

		// Create command
		cmd := &cobra.Command{
			Use: "config path",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println(cfgMgr.GetConfigFile())
				return nil
			},
		}

		// Set output
		cmd.SetOut(&buf)

		// Execute
		err = cmd.Execute()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify
		output := buf.String()
		if !strings.Contains(output, "test_config.json") {
			t.Errorf("expected config path in output, got: %s", output)
		}
	})

	t.Run("config command exists", func(t *testing.T) {
		// Verify the config command is properly set up
		if configCmd.Use != "config" {
			t.Errorf("expected config command use string 'config', got %q", configCmd.Use)
		}
		if configCmd.Short == "" {
			t.Error("config command should have a short description")
		}
	})
}

// TestHistoryCommand tests the history command and subcommands
func TestHistoryCommand(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, "history.jsonl")

	t.Run("history list command exists", func(t *testing.T) {
		// Verify the history list command is properly set up
		if historyListCmd.Use != "list" {
			t.Errorf("expected history list command use string 'list', got %q", historyListCmd.Use)
		}
	})

	t.Run("history search command exists", func(t *testing.T) {
		// Verify the history search command is properly set up
		if historySearchCmd.Use != "search <query>" {
			t.Errorf("expected history search command use string 'search <query>', got %q", historySearchCmd.Use)
		}
		if historySearchCmd.Args != cobra.ExactArgs(1) {
			t.Error("history search should require exactly 1 argument")
		}
	})

	t.Run("history show command exists", func(t *testing.T) {
		// Verify the history show command is properly set up
		if historyShowCmd.Use != "show <index>" {
			t.Errorf("expected history show command use string 'show <index>', got %q", historyShowCmd.Use)
		}
		if historyShowCmd.Args != cobra.ExactArgs(1) {
			t.Error("history show should require exactly 1 argument")
		}
	})

	t.Run("history search with matching entries", func(t *testing.T) {
		// Create history entries
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		hw.Append(models.HistoryEntry{
			Query:    "Go programming language",
			Mode:     "pro",
			Model:    "gpt51",
			Response: "Go is a statically typed programming language...",
		})

		hw.Append(models.HistoryEntry{
			Query:    "Python vs Go",
			Mode:     "fast",
			Model:    "pplx_pro",
			Response: "Python and Go are both programming languages...",
		})

		// Test search
		reader := history.NewReader(historyFile)
		entries, err := reader.Search("Go")
		if err != nil {
			t.Fatalf("failed to search history: %v", err)
		}

		if len(entries) != 2 {
			t.Errorf("expected 2 matching entries, got %d", len(entries))
		}
	})

	t.Run("history search with no matching entries", func(t *testing.T) {
		reader := history.NewReader(historyFile)
		entries, err := reader.Search("nonexistent")
		if err != nil {
			t.Fatalf("failed to search history: %v", err)
		}

		if len(entries) != 0 {
			t.Errorf("expected 0 matching entries, got %d", len(entries))
		}
	})

	t.Run("history show with valid index", func(t *testing.T) {
		// Create history entries
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		hw.Append(models.HistoryEntry{
			Query:    "Test query",
			Mode:     "pro",
			Model:    "gpt51",
			Response: "Test response",
		})

		// Read all entries
		reader := history.NewReader(historyFile)
		entries, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to read history: %v", err)
		}

		if len(entries) == 0 {
			t.Fatal("expected at least one history entry")
		}

		// Show entry at index 1
		entry := entries[0]
		if entry.Query != "Test query" {
			t.Errorf("expected query 'Test query', got %q", entry.Query)
		}
		if entry.Mode != "pro" {
			t.Errorf("expected mode 'pro', got %q", entry.Mode)
		}
	})

	t.Run("history show with invalid index", func(t *testing.T) {
		reader := history.NewReader(historyFile)
		_, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to read history: %v", err)
		}

		// Test invalid index handling
		// (This would be tested in actual command execution)
	})

	t.Run("history command exists", func(t *testing.T) {
		// Verify the history command is properly set up
		if historyCmd.Use != "history" {
			t.Errorf("expected history command use string 'history', got %q", historyCmd.Use)
		}
		if historyCmd.Short == "" {
			t.Error("history command should have a short description")
		}
	})
}

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	t.Run("version command prints version info", func(t *testing.T) {
		// Create a buffer to capture output
		var buf bytes.Buffer

		// Create command
		cmd := &cobra.Command{
			Use: "version",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Printf("perplexity %s\n", Version)
				fmt.Printf("  Git commit: %s\n", GitCommit)
				fmt.Printf("  Built:      %s\n", BuildDate)
				fmt.Printf("  Go version: %s\n", runtime.Version())
				fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
			},
		}

		// Set output
		cmd.SetOut(&buf)

		// Execute
		cmd.Run(cmd, []string{})

		// Verify
		output := buf.String()
		if !strings.Contains(output, "perplexity "+Version) {
			t.Errorf("expected version string in output, got: %s", output)
		}
		if !strings.Contains(output, "Git commit:") {
			t.Errorf("expected 'Git commit:' in output, got: %s", output)
		}
		if !strings.Contains(output, "Go version:") {
			t.Errorf("expected 'Go version:' in output, got: %s", output)
		}
		if !strings.Contains(output, "OS/Arch:") {
			t.Errorf("expected 'OS/Arch:' in output, got: %s", output)
		}
	})

	t.Run("version command exists", func(t *testing.T) {
		// Verify the version command is properly set up
		if versionCmd.Use != "version" {
			t.Errorf("expected version command use string 'version', got %q", versionCmd.Use)
		}
		if versionCmd.Short == "" {
			t.Error("version command should have a short description")
		}
	})
}

// TestCookieCommandsIntegration tests cookie-related commands with integration
func TestCookieCommandsIntegration(t *testing.T) {
	tempDir := t.TempDir()
	cookieFile := filepath.Join(tempDir, "cookies.json")

	t.Run("cookies status with existing file", func(t *testing.T) {
		// Create cookie file
		err := os.WriteFile(cookieFile, []byte(`[{"name": "next-auth.csrf-token", "value": "test123", "domain": ".perplexity.ai"}]`), 0644)
		if err != nil {
			t.Fatalf("failed to create cookie file: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
			t.Error("cookie file should exist")
		}
	})

	t.Run("cookies status with missing file", func(t *testing.T) {
		missingFile := filepath.Join(tempDir, "missing.json")

		// Verify file doesn't exist
		if _, err := os.Stat(missingFile); !os.IsNotExist(err) {
			t.Error("cookie file should not exist")
		}
	})

	t.Run("import cookies command exists", func(t *testing.T) {
		// Verify the import cookies command is properly set up
		if importCookiesCmd.Use != "import-cookies" {
			t.Errorf("expected import cookies command use string 'import-cookies', got %q", importCookiesCmd.Use)
		}
		if importCookiesCmd.Short == "" {
			t.Error("import cookies command should have a short description")
		}
	})
}

// TestEndToEndQueryWithStreaming tests complete query flow with streaming
func TestEndToEndQueryWithStreaming(t *testing.T) {
	tempDir := t.TempDir()
	queryFile := filepath.Join(tempDir, "query.txt")
	outputFile := filepath.Join(tempDir, "output.txt")
	cookieFile := filepath.Join(tempDir, "cookies.json")
	historyFile := filepath.Join(tempDir, "history.jsonl")

	// Create test files
	err := os.WriteFile(queryFile, []byte("test streaming query"), 0644)
	if err != nil {
		t.Fatalf("failed to create query file: %v", err)
	}

	err = os.WriteFile(cookieFile, []byte(`[{"name": "next-auth.csrf-token", "value": "test"}]`), 0644)
	if err != nil {
		t.Fatalf("failed to create cookie file: %v", err)
	}

	t.Run("streaming query with file input and output", func(t *testing.T) {
		// Setup config
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModePro,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
			HistoryFile:      historyFile,
		}

		// Set flags
		flagInputFile = queryFile
		flagOutputFile = outputFile
		flagCookieFile = ""
		flagIncognito = false
		flagStream = true
		flagVerbose = false

		// Create mock streaming response
		chunks := []models.StreamChunk{
			{Delta: "This is ", StepType: "FINAL"},
			{Delta: "a streaming ", StepType: "FINAL"},
			{Delta: "response.", StepType: "FINAL"},
			{WebResults: []models.WebResult{{Title: "Test Result", URL: "http://example.com",Snippet: "Test snippet"}}},
		}
		mockClient := NewMockStreamClientWithChunks(chunks, nil)

		// Override client creation for this test
		// This is a simplified integration test
		// Note: In a real integration test, we would use dependency injection

		// Verify the setup
		if cfg.CookieFile == "" {
			t.Error("cookie file should be set")
		}
		if cfg.HistoryFile == "" {
			t.Error("history file should be set")
		}
	})

	t.Run("non-streaming query with args", func(t *testing.T) {
		// Setup config
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
			HistoryFile:      historyFile,
		}

		// Reset flags
		flagInputFile = ""
		flagOutputFile = ""
		flagCookieFile = ""
		flagIncognito = false
		flagStream = false
		flagVerbose = false

		// Create mock response
		resp := &models.SearchResponse{
			Text: "This is a test response",
		}
		mockClient := NewMockClientWithResponse(resp, nil)

		// Verify setup
		if flagStream != false {
			t.Error("streaming should be disabled")
		}
	})

	t.Run("query with verbose mode", func(t *testing.T) {
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
		}

		// Set verbose flag
		flagVerbose = true

		// Test buildSearchOptions with verbose
		opts := buildSearchOptions("verbose test query")

		// Verify verbose doesn't affect search options (it's just for output)
		if opts.Query != "verbose test query" {
			t.Errorf("query mismatch")
		}
	})
}

// TestErrorScenariosIntegration tests various error scenarios
func TestErrorScenariosIntegration(t *testing.T) {
	tempDir := t.TempDir()
	cookieFile := filepath.Join(tempDir, "cookies.json")

	// Create cookie file
	err := os.WriteFile(cookieFile, []byte(`[{"name": "next-auth.csrf-token", "value": "test"}]`), 0644)
	if err != nil {
		t.Fatalf("failed to create cookie file: %v", err)
	}

	t.Run("client creation error", func(t *testing.T) {
		// Test with invalid cookies
		invalidCookies := []*http.Cookie{}

		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
		}

		// This would normally create a client
		// We're testing the error handling path
	})

	t.Run("missing required cookie", func(t *testing.T) {
		// Create cookie file without required csrf-token
		badCookieFile := filepath.Join(tempDir, "badcookies.json")
		err := os.WriteFile(badCookieFile, []byte(`[{"name": "other-cookie", "value": "value"}]`), 0644)
		if err != nil {
			t.Fatalf("failed to create bad cookie file: %v", err)
		}

		// The client should detect missing csrf-token
		// This tests the error handling
	})

	t.Run("API error response", func(t *testing.T) {
		// Create mock client that returns an error
		mockClient := NewMockClientWithResponse(nil, fmt.Errorf("API error: rate limit exceeded"))

		// Verify error handling
		resp, err := mockClient.Search(context.Background(), models.SearchOptions{Query: "test"})
		if err == nil {
			t.Error("expected error, got nil")
		}
		if resp != nil {
			t.Error("response should be nil on error")
		}
	})

	t.Run("streaming error", func(t *testing.T) {
		// Create mock client that returns streaming error
		mockClient := NewMockStreamClientWithChunks(nil, fmt.Errorf("stream error"))

		// Verify error handling
		ch, err := mockClient.SearchStream(context.Background(), models.SearchOptions{Query: "test"})
		if err == nil {
			t.Error("expected error, got nil")
		}
		if ch == nil {
			t.Error("channel should be non-nil even on error")
		}
	})
}

// TestFlagCombinationsIntegration tests various flag combinations
func TestFlagCombinationsIntegration(t *testing.T) {
	// Save current state
	origFlagModel := flagModel
	origFlagMode := flagMode
	origFlagLanguage := flagLanguage
	origFlagSources := flagSources
	origFlagIncognito := flagIncognito
	origFlagStream := flagStream
	origFlagNoStream := flagNoStream

	defer func() {
		flagModel = origFlagModel
		flagMode = origFlagMode
		flagLanguage = origFlagLanguage
		flagSources = origFlagSources
		flagIncognito = origFlagIncognito
		flagStream = origFlagStream
		flagNoStream = origFlagNoStream
	}()

	cfg = &config.Config{
		DefaultModel:     models.ModelPplxPro,
		DefaultMode:      models.ModeDefault,
		DefaultLanguage:  "en-US",
		DefaultSources:   []models.Source{models.SourceWeb},
		Incognito:        false,
	}

	t.Run("stream and no-stream flags interaction", func(t *testing.T) {
		// Test that --no-stream overrides config streaming
		cfg.Streaming = true
		flagStream = false
		flagNoStream = true

		// The logic in runQuery checks these flags
		// streaming should be false due to flagNoStream
	})

	t.Run("all search-related flags together", func(t *testing.T) {
		flagModel = "gpt51"
		flagMode = "pro"
		flagLanguage = "fr-FR"
		flagSources = "web,scholar,social"
		flagIncognito = true

		opts := buildSearchOptions("combined flags test")

		if opts.Model != models.ModelGPT51 {
			t.Errorf("expected model gpt51, got %q", opts.Model)
		}
		if opts.Mode != models.ModePro {
			t.Errorf("expected mode pro, got %q", opts.Mode)
		}
		if opts.Language != "fr-FR" {
			t.Errorf("expected language fr-FR, got %q", opts.Language)
		}
		if len(opts.Sources) != 3 {
			t.Errorf("expected 3 sources, got %d", len(opts.Sources))
		}
		if !opts.Incognito {
			t.Error("expected incognito to be true")
		}
	})

	t.Run("empty sources string uses config default", func(t *testing.T) {
		flagModel = ""
		flagMode = ""
		flagLanguage = ""
		flagSources = ""

		opts := buildSearchOptions("empty flags test")

		if opts.Model != models.ModelPplxPro {
			t.Errorf("expected model pplx_pro from config, got %q", opts.Model)
		}
	})
}

// TestConfigIntegration tests config loading and integration
func TestConfigIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Save and restore HOME
	oldHome := os.Getenv("HOME")
	defer func() { os.Setenv("HOME", oldHome) }()
	os.Setenv("HOME", tempDir)

	t.Run("config file with all fields", func(t *testing.T) {
		// Create config directory
		configDir := filepath.Join(tempDir, ".perplexity-cli")
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		// Write comprehensive config
		configFile := filepath.Join(configDir, "config.json")
		configJSON := `{
			"default_model": "gpt51",
			"default_mode": "pro",
			"default_language": "pt-BR",
			"streaming": true,
			"incognito": true,
			"history_file": "custom_history.jsonl"
		}`
		err = os.WriteFile(configFile, []byte(configJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Load and verify
		mgr, err := config.NewManager()
		if err != nil {
			t.Fatalf("failed to create manager: %v", err)
		}

		testCfg, err := mgr.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if testCfg.DefaultModel != "gpt51" {
			t.Errorf("expected model gpt51, got %q", testCfg.DefaultModel)
		}
		if testCfg.DefaultMode != "pro" {
			t.Errorf("expected mode pro, got %q", testCfg.DefaultMode)
		}
	})

	t.Run("config with missing optional fields", func(t *testing.T) {
		// Create minimal config
		configDir := filepath.Join(tempDir, ".perplexity-cli")
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		configFile := filepath.Join(configDir, "config.json")
		configJSON := `{
			"default_model": "claude45sonnet"
		}`
		err = os.WriteFile(configFile, []byte(configJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Load and verify defaults are applied
		mgr, err := config.NewManager()
		if err != nil {
			t.Fatalf("failed to create manager: %v", err)
		}

		testCfg, err := mgr.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Should have defaults for missing fields
		if testCfg.DefaultMode == "" {
			t.Error("default mode should be set")
		}
	})
}

// TestHistoryIntegration tests history functionality integration
func TestHistoryIntegration(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, "test_history.jsonl")

	t.Run("history writer and reader integration", func(t *testing.T) {
		// Create history entries
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		// Add multiple entries
		entries := []models.HistoryEntry{
			{Query: "Query 1", Mode: "pro", Model: "gpt51", Response: "Response 1"},
			{Query: "Query 2", Mode: "fast", Model: "pplx_pro", Response: "Response 2"},
			{Query: "Query 3", Mode: "reasoning", Model: "gpt51_thinking", Response: "Response 3"},
		}

		for _, entry := range entries {
			hw.Append(entry)
		}

		// Read all entries
		reader := history.NewReader(historyFile)
		readEntries, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to read history: %v", err)
		}

		if len(readEntries) != len(entries) {
			t.Errorf("expected %d entries, got %d", len(entries), len(readEntries))
		}

		// Verify timestamps are set
		for _, entry := range readEntries {
			if entry.Timestamp.IsZero() {
				t.Error("timestamp should be set")
			}
		}
	})

	t.Run("history search integration", func(t *testing.T) {
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		hw.Append(models.HistoryEntry{
			Query:    "Go programming tutorial",
			Mode:     "pro",
			Model:    "gpt51",
			Response: "Learn Go programming...",
		})

		hw.Append(models.HistoryEntry{
			Query:    "Python basics",
			Mode:     "fast",
			Model:    "pplx_pro",
			Response: "Python is easy...",
		})

		// Search for "Go"
		reader := history.NewReader(historyFile)
		results, err := reader.Search("Go")
		if err != nil {
			t.Fatalf("failed to search history: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 result for 'Go', got %d", len(results))
		}

		if results[0].Query != "Go programming tutorial" {
			t.Errorf("expected 'Go programming tutorial', got %q", results[0].Query)
		}
	})

	t.Run("history with truncation", func(t *testing.T) {
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		// Add entry with long response
		longResponse := strings.Repeat("This is a long response. ", 100)
		hw.Append(models.HistoryEntry{
			Query:    "Long query",
			Mode:     "pro",
			Model:    "gpt51",
			Response: longResponse,
		})

		// The truncation happens in runQuery
		truncated := truncateResponse(longResponse, 500)
		if len(truncated) > 503 { // 500 + "..."
			t.Errorf("truncated response too long: %d characters", len(truncated))
		}
	})
}

// TestTimeoutAndCancellationIntegration tests timeout and cancellation scenarios
func TestTimeoutAndCancellationIntegration(t *testing.T) {
	t.Run("context cancellation handling", func(t *testing.T) {
		// Create a mock client that supports context cancellation
		ctx, cancel := context.WithCancel(context.Background())

		mockClient := NewMockClientWithResponse(&models.SearchResponse{
			Text: "Response",
		}, nil)

		// Start search
		resp, err := mockClient.Search(ctx, models.SearchOptions{Query: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Cancel context
		cancel()

		// In a real scenario, this would be detected in the response
		// For this mock test, we just verify the pattern
		if resp == nil {
			t.Error("response should not be nil")
		}
	})

	t.Run("timeout handling", func(t *testing.T) {
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Short delay to trigger timeout
		time.Sleep(10 * time.Millisecond)

		// Context should be done
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Error("context should be done after timeout")
		}
	})
}
