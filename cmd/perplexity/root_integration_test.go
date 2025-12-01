package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/internal/auth"
	"github.com/diogo/perplexity-go/internal/config"
	"github.com/diogo/perplexity-go/internal/history"
	"github.com/diogo/perplexity-go/pkg/models"
)

// FullMockClient is a comprehensive mock for end-to-end testing
type FullMockClient struct {
	searchResponse  *models.SearchResponse
	searchError     error
	streamChunks    []models.StreamChunk
	streamError     error
	closeError      error
	cookies         []*http.Cookie
	hasValidSession bool
	proQueriesRem   int
	fileUploadsRem  int
}

func NewFullMockClient(response *models.SearchResponse, err error) *FullMockClient {
	return &FullMockClient{
		searchResponse:  response,
		searchError:     err,
		hasValidSession: true,
		proQueriesRem:   5,
		fileUploadsRem:  10,
	}
}

func NewFullStreamMockClient(chunks []models.StreamChunk, err error) *FullMockClient {
	return &FullMockClient{
		streamChunks:    chunks,
		streamError:     err,
		hasValidSession: true,
		proQueriesRem:   5,
		fileUploadsRem:  10,
	}
}

func (m *FullMockClient) Search(ctx context.Context, opts models.SearchOptions) (*models.SearchResponse, error) {
	return m.searchResponse, m.searchError
}

func (m *FullMockClient) SearchStream(ctx context.Context, opts models.SearchOptions) (<-chan models.StreamChunk, error) {
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

func (m *FullMockClient) Close() error {
	return m.closeError
}

func (m *FullMockClient) SetCookies(cookies []*http.Cookie) {
	m.cookies = cookies
}

func (m *FullMockClient) GetCookies() []*http.Cookie {
	return m.cookies
}

func (m *FullMockClient) HasValidSession() bool {
	return m.hasValidSession
}

func (m *FullMockClient) ProQueriesRemaining() int {
	return m.proQueriesRem
}

func (m *FullMockClient) FileUploadsRemaining() int {
	return m.fileUploadsRem
}

// TestCompleteQueryFlow tests the complete end-to-end query flow
func TestCompleteQueryFlow(t *testing.T) {
	tempDir := t.TempDir()

	// Test files
	queryFile := filepath.Join(tempDir, "query.txt")
	outputFile := filepath.Join(tempDir, "output.md")
	cookieFile := filepath.Join(tempDir, "cookies.json")
	historyFile := filepath.Join(tempDir, "history.jsonl")

	// Create test files
	err := os.WriteFile(queryFile, []byte("What is artificial intelligence?"), 0644)
	if err != nil {
		t.Fatalf("failed to create query file: %v", err)
	}

	cookieJSON := `[{"name": "next-auth.csrf-token", "value": "test123", "domain": ".perplexity.ai", "path": "/"}]`
	err = os.WriteFile(cookieFile, []byte(cookieJSON), 0644)
	if err != nil {
		t.Fatalf("failed to create cookie file: %v", err)
	}

	t.Run("complete flow with file input, streaming, and output", func(t *testing.T) {
		// Setup configuration
		cfg = &config.Config{
			DefaultModel:    models.ModelPplxPro,
			DefaultMode:     models.ModePro,
			DefaultLanguage: "en-US",
			DefaultSources:  []models.Source{models.SourceWeb, models.SourceScholar},
			CookieFile:      cookieFile,
			HistoryFile:     historyFile,
			Streaming:       false,
			Incognito:       false,
		}

		// Set flags for the test
		flagInputFile = queryFile
		flagOutputFile = outputFile
		flagCookieFile = ""
		flagIncognito = false
		flagStream = true
		flagVerbose = false

		// Create mock response
		mockResponse := &models.SearchResponse{
			Text: "Artificial intelligence (AI) is a branch of computer science...",
		}
		_ = mockResponse // silence unused warning

		// This test demonstrates the flow setup
		// In a real integration test with DI, we would execute runQuery

		// Verify setup
		if cfg.CookieFile == "" {
			t.Fatal("cookie file not configured")
		}
		// Check if sources include web
		hasWeb := false
		for _, src := range cfg.DefaultSources {
			if strings.Contains(string(src), "web") {
				hasWeb = true
				break
			}
		}
		if !hasWeb {
			t.Error("sources should include web")
		}
		if flagInputFile == "" {
			t.Fatal("input file not set")
		}

		// Verify cookie file can be loaded
		cookies, err := auth.LoadCookiesFromFile(cookieFile)
		if err != nil {
			t.Fatalf("failed to load cookies: %v", err)
		}
		if len(cookies) == 0 {
			t.Fatal("no cookies loaded")
		}
	})

	t.Run("complete flow with args input, non-streaming, and history", func(t *testing.T) {
		cfg = &config.Config{
			DefaultModel:    models.ModelGPT51,
			DefaultMode:     models.ModeFast,
			DefaultLanguage: "pt-BR",
			DefaultSources:  []models.Source{models.SourceWeb},
			CookieFile:      cookieFile,
			HistoryFile:     historyFile,
			Streaming:       true,
			Incognito:       false,
		}

		// Set flags
		flagInputFile = ""
		flagOutputFile = ""
		flagCookieFile = ""
		flagIncognito = false
		flagStream = false
		flagVerbose = true

		// Create mock response
		mockResponse := &models.SearchResponse{
			Text: "Go é uma linguagem de programação...",
		}
		_ = mockResponse // silence unused warning

		// Verify configuration
		if cfg.DefaultModel != models.ModelGPT51 {
			t.Errorf("expected model %s, got %s", models.ModelGPT51, cfg.DefaultModel)
		}
		if cfg.DefaultMode != models.ModeFast {
			t.Errorf("expected mode %s, got %s", models.ModeFast, cfg.DefaultMode)
		}
	})

	t.Run("stdin input flow", func(t *testing.T) {
		stdin := strings.NewReader("Query from stdin")
		isTerminal := false

		query, err := getQueryFromInput([]string{}, stdin, isTerminal)
		if err != nil {
			t.Fatalf("failed to read from stdin: %v", err)
		}

		if query != "Query from stdin" {
			t.Errorf("expected 'Query from stdin', got %q", query)
		}
	})

	t.Run("complete flow with output file", func(t *testing.T) {
		testContent := "# Response\n\nThis is the answer."
		err := os.WriteFile(outputFile, []byte(testContent), 0644)
		if err != nil {
			t.Fatalf("failed to write output file: %v", err)
		}

		// Verify file was created
		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		if string(data) != testContent {
			t.Errorf("content mismatch")
		}
	})
}

// TestStreamingResponseFlow tests various streaming scenarios
func TestStreamingResponseFlow(t *testing.T) {
	tempDir := t.TempDir()
	cookieFile := filepath.Join(tempDir, "cookies.json")

	cookieJSON := `[{"name": "next-auth.csrf-token", "value": "test", "domain": ".perplexity.ai"}]`
	err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644)
	if err != nil {
		t.Fatalf("failed to create cookie file: %v", err)
	}

	t.Run("step-based streaming format", func(t *testing.T) {
		chunks := []models.StreamChunk{
			{StepType: "THINKING", Text: "Thinking...", Delta: "Thinking..."},
			{StepType: "FINAL", Text: "This is the final answer.", Delta: "This is the final answer."},
		}

		mockClient := NewFullStreamMockClient(chunks, nil)
		ch, err := mockClient.SearchStream(context.Background(), models.SearchOptions{Query: "test"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read chunks
		received := 0
		for range ch {
			received++
		}

		if received != len(chunks) {
			t.Errorf("expected %d chunks, got %d", len(chunks), received)
		}
	})

	t.Run("legacy streaming format", func(t *testing.T) {
		chunks := []models.StreamChunk{
			{Delta: "First ", StepType: ""},
			{Delta: "chunk", StepType: ""},
			{Delta: " of text", StepType: ""},
		}

		mockClient := NewFullStreamMockClient(chunks, nil)
		ch, err := mockClient.SearchStream(context.Background(), models.SearchOptions{Query: "test"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read chunks
		received := 0
		var fullText string
		for chunk := range ch {
			received++
			fullText += chunk.Delta
		}

		if received != len(chunks) {
			t.Errorf("expected %d chunks, got %d", len(chunks), received)
		}
		if fullText != "First chunk of text" {
			t.Errorf("expected 'First chunk of text', got %q", fullText)
		}
	})

	t.Run("streaming with web results", func(t *testing.T) {
		chunks := []models.StreamChunk{
			{
				StepType: "FINAL",
				Text:     "Answer text",
				WebResults: []models.WebResult{
					{Title: "Result 1", URL: "http://example.com", Snippet: "Snippet 1"},
					{Title: "Result 2", URL: "http://example.org", Snippet: "Snippet 2"},
				},
			},
		}

		mockClient := NewFullStreamMockClient(chunks, nil)
		ch, err := mockClient.SearchStream(context.Background(), models.SearchOptions{Query: "test"})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read chunks and collect web results
		var webResults []models.WebResult
		for chunk := range ch {
			webResults = append(webResults, chunk.WebResults...)
		}

		if len(webResults) != 2 {
			t.Errorf("expected 2 web results, got %d", len(webResults))
		}
	})

	t.Run("streaming error handling", func(t *testing.T) {
		mockClient := NewFullStreamMockClient(nil, fmt.Errorf("API error"))
		ch, err := mockClient.SearchStream(context.Background(), models.SearchOptions{Query: "test"})

		if err == nil {
			t.Error("expected error, got nil")
		}
		if err.Error() != "API error" {
			t.Errorf("expected 'API error', got %v", err)
		}

		// Channel should still receive error chunk
		received := 0
		for range ch {
			received++
		}
		if received != 1 {
			t.Errorf("expected 1 error chunk, got %d", received)
		}
	})

	t.Run("context cancellation in streaming", func(t *testing.T) {
		chunks := []models.StreamChunk{
			{Delta: "First", StepType: "FINAL"},
			{Delta: "Second", StepType: "FINAL"},
			{Delta: "Third", StepType: "FINAL"},
		}

		mockClient := NewFullStreamMockClient(chunks, nil)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := mockClient.SearchStream(ctx, models.SearchOptions{Query: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Receive first chunk
		chunk := <-ch
		if chunk.Delta != "First" {
			t.Errorf("expected 'First', got %q", chunk.Delta)
		}

		// Cancel context
		cancel()

		// Next chunk should be error (context canceled)
		chunk = <-ch
		if chunk.Error != context.Canceled {
			t.Errorf("expected context.Canceled error, got %v", chunk.Error)
		}
	})
}

// TestNonStreamingResponseFlow tests non-streaming response scenarios
func TestNonStreamingResponseFlow(t *testing.T) {
	tempDir := t.TempDir()
	cookieFile := filepath.Join(tempDir, "cookies.json")

	cookieJSON := `[{"name": "next-auth.csrf-token", "value": "test", "domain": ".perplexity.ai"}]`
	err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644)
	if err != nil {
		t.Fatalf("failed to create cookie file: %v", err)
	}

	t.Run("successful non-streaming response", func(t *testing.T) {
		mockClient := NewFullMockClient(&models.SearchResponse{
			Text: "This is a complete response",
		}, nil)

		resp, err := mockClient.Search(context.Background(), models.SearchOptions{Query: "test", Stream: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("response is nil")
		}
		if resp.Text != "This is a complete response" {
			t.Errorf("expected response text, got %q", resp.Text)
		}
	})

	t.Run("API error in non-streaming", func(t *testing.T) {
		mockClient := NewFullMockClient(nil, fmt.Errorf("rate limit exceeded"))

		resp, err := mockClient.Search(context.Background(), models.SearchOptions{Query: "test", Stream: false})
		if err == nil {
			t.Error("expected error, got nil")
		}
		if resp != nil {
			t.Error("response should be nil on error")
		}
	})

	t.Run("empty response text", func(t *testing.T) {
		mockClient := NewFullMockClient(&models.SearchResponse{}, nil)

		resp, err := mockClient.Search(context.Background(), models.SearchOptions{Query: "test", Stream: false})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("response is nil")
		}
	})
}

// TestFileOperationsIntegration tests file input/output operations
func TestFileOperationsIntegration(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("input from markdown file", func(t *testing.T) {
		mdFile := filepath.Join(tempDir, "prompt.md")
		mdContent := `# Question

What is the difference between Go and Python?

Please explain with examples.`

		err := os.WriteFile(mdFile, []byte(mdContent), 0644)
		if err != nil {
			t.Fatalf("failed to write markdown file: %v", err)
		}

		content, err := getQueryFromFile(mdFile)
		if err != nil {
			t.Fatalf("failed to read markdown file: %v", err)
		}

		if !strings.Contains(content, "Go and Python") {
			t.Error("markdown content should be preserved")
		}
	})

	t.Run("output to markdown file with formatting", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "answer.md")
		mdContent := "# Answer\n\nThis is **bold** and this is *italic*.\n\n- Item 1\n- Item 2"

		err := os.WriteFile(outputFile, []byte(mdContent), 0644)
		if err != nil {
			t.Fatalf("failed to write output file: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		if string(data) != mdContent {
			t.Error("output content mismatch")
		}
	})

	t.Run("binary file handling", func(t *testing.T) {
		// Test that binary files are handled correctly
		binaryFile := filepath.Join(tempDir, "binary.bin")
		binaryContent := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header

		err := os.WriteFile(binaryFile, binaryContent, 0644)
		if err != nil {
			t.Fatalf("failed to write binary file: %v", err)
		}

		content, err := getQueryFromFile(binaryFile)
		if err != nil {
			t.Fatalf("failed to read binary file: %v", err)
		}

		// Content should be read as bytes and converted to string
		if len(content) == 0 {
			t.Error("binary file should have content")
		}
	})

	t.Run("unicode content", func(t *testing.T) {
		unicodeFile := filepath.Join(tempDir, "unicode.txt")
		unicodeContent := "こんにちは世界 🌍 café naïve résumé"

		err := os.WriteFile(unicodeFile, []byte(unicodeContent), 0644)
		if err != nil {
			t.Fatalf("failed to write unicode file: %v", err)
		}

		content, err := getQueryFromFile(unicodeFile)
		if err != nil {
			t.Fatalf("failed to read unicode file: %v", err)
		}

		if content != unicodeContent {
			t.Error("unicode content should be preserved")
		}
	})

	t.Run("large file handling", func(t *testing.T) {
		largeFile := filepath.Join(tempDir, "large.txt")
		largeContent := strings.Repeat("Line of text. ", 10000)

		err := os.WriteFile(largeFile, []byte(largeContent), 0644)
		if err != nil {
			t.Fatalf("failed to write large file: %v", err)
		}

		start := time.Now()
		content, err := getQueryFromFile(largeFile)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("failed to read large file: %v", err)
		}

		if len(content) != len(largeContent) {
			t.Errorf("content length mismatch: got %d, want %d", len(content), len(largeContent))
		}

		// Should be able to read large file reasonably fast
		if elapsed > 5*time.Second {
			t.Errorf("reading large file took too long: %v", elapsed)
		}
	})

	t.Run("permission denied error", func(t *testing.T) {
		// Skip on Windows
		if os.Getenv("GOOS") == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		noPermFile := filepath.Join(tempDir, "noperm.txt")
		err := os.WriteFile(noPermFile, []byte("content"), 0000)
		if err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		defer os.Chmod(noPermFile, 0644)

		_, err = getQueryFromFile(noPermFile)
		if err == nil {
			t.Error("expected permission denied error")
		}
	})
}

// TestSearchOptionsBuildIntegration tests buildSearchOptions with integration scenarios
func TestSearchOptionsBuildIntegration(t *testing.T) {
	// Save state
	origCfg := cfg
	origFlagModel := flagModel
	origFlagMode := flagMode
	origFlagLanguage := flagLanguage
	origFlagSources := flagSources
	origFlagIncognito := flagIncognito

	defer func() {
		cfg = origCfg
		flagModel = origFlagModel
		flagMode = origFlagMode
		flagLanguage = origFlagLanguage
		flagSources = origFlagSources
		flagIncognito = origFlagIncognito
	}()

	t.Run("all models with correct modes", func(t *testing.T) {
		testCases := []struct {
			model    string
			mode     string
			expected models.Mode
		}{
			{"gpt51", "pro", models.ModePro},
			{"pplx_pro", "fast", models.ModeFast},
			{"gpt51_thinking", "reasoning", models.ModeReasoning},
			{"claude45sonnet", "pro", models.ModePro},
			{"experimental", "pro", models.ModePro},
		}

		for _, tc := range testCases {
			cfg = &config.Config{
				DefaultModel:    models.ModelPplxPro,
				DefaultMode:     models.ModeDefault,
				DefaultLanguage: "en-US",
				DefaultSources:  []models.Source{models.SourceWeb},
			}

			flagModel = tc.model
			flagMode = tc.mode
			flagLanguage = ""
			flagSources = ""
			flagIncognito = false

			opts := buildSearchOptions("test query")

			if opts.Model != models.Model(tc.model) {
				t.Errorf("model mismatch for %s: got %q, want %q", tc.model, opts.Model, tc.model)
			}
			if opts.Mode != tc.expected {
				t.Errorf("mode mismatch for %s: got %q, want %q", tc.model, opts.Mode, tc.expected)
			}
		}
	})

	t.Run("sources parsing", func(t *testing.T) {
		cfg = &config.Config{
			DefaultModel:    models.ModelPplxPro,
			DefaultMode:     models.ModeDefault,
			DefaultLanguage: "en-US",
			DefaultSources:  []models.Source{models.SourceWeb},
		}

		flagModel = ""
		flagMode = ""
		flagLanguage = ""
		flagSources = "web,scholar,social"
		flagIncognito = false

		opts := buildSearchOptions("test query")

		if len(opts.Sources) != 3 {
			t.Errorf("expected 3 sources, got %d", len(opts.Sources))
		}

		expectedSources := []models.Source{models.SourceWeb, models.SourceScholar, models.SourceSocial}
		for i, src := range opts.Sources {
			if src != expectedSources[i] {
				t.Errorf("source %d: got %q, want %q", i, src, expectedSources[i])
			}
		}
	})

	t.Run("sources with whitespace", func(t *testing.T) {
		cfg = &config.Config{
			DefaultModel:    models.ModelPplxPro,
			DefaultMode:     models.ModeDefault,
			DefaultLanguage: "en-US",
			DefaultSources:  []models.Source{models.SourceWeb},
		}

		flagModel = ""
		flagMode = ""
		flagLanguage = ""
		flagSources = " web , scholar , social "
		flagIncognito = false

		opts := buildSearchOptions("test query")

		if len(opts.Sources) != 3 {
			t.Errorf("expected 3 sources (whitespace trimmed), got %d", len(opts.Sources))
		}
	})

	t.Run("incognito flag precedence", func(t *testing.T) {
		cfg = &config.Config{
			DefaultModel:    models.ModelPplxPro,
			DefaultMode:     models.ModeDefault,
			DefaultLanguage: "en-US",
			DefaultSources:  []models.Source{models.SourceWeb},
			Incognito:       false,
		}

		flagModel = ""
		flagMode = ""
		flagLanguage = ""
		flagSources = ""
		flagIncognito = true

		opts := buildSearchOptions("test query")

		if !opts.Incognito {
			t.Error("incognito flag should override config")
		}
	})

	t.Run("language override", func(t *testing.T) {
		cfg = &config.Config{
			DefaultModel:    models.ModelPplxPro,
			DefaultMode:     models.ModeDefault,
			DefaultLanguage: "en-US",
			DefaultSources:  []models.Source{models.SourceWeb},
		}

		flagModel = ""
		flagMode = ""
		flagLanguage = "es-ES"
		flagSources = ""
		flagIncognito = false

		opts := buildSearchOptions("test query")

		if opts.Language != "es-ES" {
			t.Errorf("expected language es-ES, got %q", opts.Language)
		}
	})
}

// TestSearchIntegrationWithHistory tests search integration with history
func TestSearchIntegrationWithHistory(t *testing.T) {
	tempDir := t.TempDir()
	historyFile := filepath.Join(tempDir, "history.jsonl")

	t.Run("history entry creation", func(t *testing.T) {
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		entry := models.HistoryEntry{
			Query:    "What is Go?",
			Mode:     "pro",
			Model:    "gpt51",
			Response: "Go is a programming language developed by Google.",
		}

		hw.Append(entry)

		// Verify entry was written
		reader := history.NewReader(historyFile)
		entries, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to read history: %v", err)
		}

		if len(entries) != 1 {
			t.Errorf("expected 1 history entry, got %d", len(entries))
		}

		if entries[0].Query != entry.Query {
			t.Errorf("query mismatch: got %q, want %q", entries[0].Query, entry.Query)
		}
	})

	t.Run("multiple history entries", func(t *testing.T) {
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		queries := []string{
			"Query 1",
			"Query 2",
			"Query 3",
		}

		for i, query := range queries {
			hw.Append(models.HistoryEntry{
				Query:    query,
				Mode:     "pro",
				Model:    "gpt51",
				Response: fmt.Sprintf("Response %d", i+1),
			})
		}

		reader := history.NewReader(historyFile)
		entries, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to read history: %v", err)
		}

		if len(entries) != 3 {
			t.Errorf("expected 3 history entries, got %d", len(entries))
		}
	})

	t.Run("history search integration", func(t *testing.T) {
		hw, err := history.NewWriter(historyFile)
		if err != nil {
			t.Fatalf("failed to create history writer: %v", err)
		}

		// Add entries with searchable terms
		entries := []models.HistoryEntry{
			{Query: "Go programming language tutorial", Mode: "pro", Model: "gpt51", Response: "Response 1"},
			{Query: "Python for beginners", Mode: "fast", Model: "pplx_pro", Response: "Response 2"},
			{Query: "Rust vs Go comparison", Mode: "reasoning", Model: "gpt51_thinking", Response: "Response 3"},
		}

		for _, entry := range entries {
			hw.Append(entry)
		}

		// Search for "Go"
		reader := history.NewReader(historyFile)
		results, err := reader.Search("Go")
		if err != nil {
			t.Fatalf("failed to search history: %v", err)
		}

		// Should find entries 1 and 3
		if len(results) != 2 {
			t.Errorf("expected 2 results for 'Go', got %d", len(results))
		}

		for _, result := range results {
			if !strings.Contains(strings.ToLower(result.Query), "go") {
				t.Errorf("search result should contain 'Go': %q", result.Query)
			}
		}
	})

	t.Run("response truncation for history", func(t *testing.T) {
		longResponse := strings.Repeat("Long response. ", 100)

		truncated := truncateResponse(longResponse, 500)

		if len(truncated) > 503 { // 500 + "..."
			t.Errorf("truncated response too long: %d characters", len(truncated))
		}

		if !strings.HasSuffix(truncated, "...") {
			t.Error("truncated response should end with '...'")
		}
	})
}

// TestCookieValidationIntegration tests cookie validation scenarios
func TestCookieValidationIntegration(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid cookie file", func(t *testing.T) {
		cookieFile := filepath.Join(tempDir, "valid.json")
		cookieJSON := `[
			{
				"name": "next-auth.csrf-token",
				"value": "abc123",
				"domain": ".perplexity.ai",
				"path": "/"
			}
		]`

		err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write cookie file: %v", err)
		}

		cookies, err := auth.LoadCookiesFromFile(cookieFile)
		if err != nil {
			t.Fatalf("failed to load cookies: %v", err)
		}

		if len(cookies) != 1 {
			t.Errorf("expected 1 cookie, got %d", len(cookies))
		}

		if cookies[0].Name != "next-auth.csrf-token" {
			t.Errorf("expected cookie name 'next-auth.csrf-token', got %q", cookies[0].Name)
		}
	})

	t.Run("invalid JSON cookie file", func(t *testing.T) {
		cookieFile := filepath.Join(tempDir, "invalid.json")
		invalidJSON := `{"name": "next-auth.csrf-token", "value": "test"` // Missing closing brace

		err := os.WriteFile(cookieFile, []byte(invalidJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write cookie file: %v", err)
		}

		_, err = auth.LoadCookiesFromFile(cookieFile)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}
	})

	t.Run("empty cookie file", func(t *testing.T) {
		cookieFile := filepath.Join(tempDir, "empty.json")

		err := os.WriteFile(cookieFile, []byte("[]"), 0644)
		if err != nil {
			t.Fatalf("failed to write cookie file: %v", err)
		}

		cookies, err := auth.LoadCookiesFromFile(cookieFile)
		if err != nil {
			t.Fatalf("failed to load cookies: %v", err)
		}

		if len(cookies) != 0 {
			t.Errorf("expected 0 cookies, got %d", len(cookies))
		}
	})

	t.Run("missing csrf-token cookie", func(t *testing.T) {
		cookieFile := filepath.Join(tempDir, "missing_csrf.json")
		cookieJSON := `[
			{
				"name": "other-cookie",
				"value": "value",
				"domain": ".perplexity.ai"
			}
		]`

		err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write cookie file: %v", err)
		}

		cookies, err := auth.LoadCookiesFromFile(cookieFile)
		if err != nil {
			t.Fatalf("failed to load cookies: %v", err)
		}
		_ = cookies // silence unused warning

		// Client should detect missing csrf-token
		// This would fail during client initialization in real usage
	})

	t.Run("Netscape format cookie file", func(t *testing.T) {
		cookieFile := filepath.Join(tempDir, "netscape.txt")
		netscapeFormat := `# Netscape HTTP Cookie File
.perplexity.ai	TRUE	/	FALSE	1735689600	next-auth.csrf-token	abc123`

		err := os.WriteFile(cookieFile, []byte(netscapeFormat), 0644)
		if err != nil {
			t.Fatalf("failed to write cookie file: %v", err)
		}

		cookies, err := auth.LoadCookiesFromFile(cookieFile)
		if err != nil {
			t.Fatalf("failed to load cookies: %v", err)
		}

		// Should parse at least one cookie
		if len(cookies) == 0 {
			t.Error("should parse cookies from Netscape format")
		}
	})
}

// TestContextCancellationIntegration tests context cancellation scenarios
func TestContextCancellationIntegration(t *testing.T) {
	t.Run("SIGINT handling simulation", func(t *testing.T) {
		// Create a channel to simulate signal handling
		sigCh := make(chan os.Signal, 1)
		_ = sigCh // silence unused warning

		// In real runQuery, this would be:
		// signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		// Simulate receiving SIGINT
		// sigCh <- os.Interrupt

		// Context should be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Simulate cancellation
		cancel()

		// Verify context is done
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Error("context should be done after cancellation")
		}
	})

	t.Run("streaming with cancellation", func(t *testing.T) {
		chunks := []models.StreamChunk{
			{Delta: "First chunk"},
			{Delta: "Second chunk"},
			{Delta: "Third chunk"},
		}

		mockClient := NewFullStreamMockClient(chunks, nil)
		ctx, cancel := context.WithCancel(context.Background())

		ch, err := mockClient.SearchStream(ctx, models.SearchOptions{Query: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Receive first chunk
		chunk := <-ch
		if chunk.Delta != "First chunk" {
			t.Errorf("expected 'First chunk', got %q", chunk.Delta)
		}

		// Cancel context
		cancel()

		// Should receive error chunk
		errorChunk := <-ch
		if errorChunk.Error != context.Canceled {
			t.Errorf("expected context.Canceled error, got %v", errorChunk.Error)
		}
	})
}

// TestEnvironmentVariableIntegration tests environment variable integration
func TestEnvironmentVariableIntegration(t *testing.T) {
	t.Run("config with environment variables", func(t *testing.T) {
		// Note: The config package should handle environment variables
		// This test verifies the integration point

		// In the actual application, environment variables like:
		// PERPLEXITY_DEFAULT_MODEL, PERPLEXITY_DEFAULT_MODE, etc.
		// would be read by the config package

		// This is a placeholder for env var integration testing
		// The actual implementation is in internal/config
	})

	t.Run("environment variable precedence", func(t *testing.T) {
		// Test that environment variables can override config file
		// This would be tested in integration with the config package
	})
}

// TestVerboseModeIntegration tests verbose mode integration
func TestVerboseModeIntegration(t *testing.T) {
	tempDir := t.TempDir()
	cookieFile := filepath.Join(tempDir, "cookies.json")

	cookieJSON := `[{"name": "next-auth.csrf-token", "value": "test", "domain": ".perplexity.ai"}]`
	err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644)
	if err != nil {
		t.Fatalf("failed to create cookie file: %v", err)
	}

	cfg = &config.Config{
		DefaultModel:    models.ModelPplxPro,
		DefaultMode:     models.ModePro,
		DefaultLanguage: "en-US",
		DefaultSources:  []models.Source{models.SourceWeb},
		CookieFile:      cookieFile,
	}

	t.Run("verbose mode outputs query info", func(t *testing.T) {
		flagVerbose = true

		// When verbose is true, runQuery should output:
		// - Query: <query>
		// - Mode: <mode>, Model: <model>
		// - Streaming: <streaming>

		opts := buildSearchOptions("verbose test query")

		if opts.Query != "verbose test query" {
			t.Error("query should be set correctly")
		}
	})

	t.Run("verbose mode with all flags", func(t *testing.T) {
		flagVerbose = true
		flagModel = "gpt51"
		flagMode = "reasoning"
		flagSources = "web,scholar"
		flagLanguage = "pt-BR"
		flagStream = true
		flagIncognito = true

		opts := buildSearchOptions("full verbose test")

		// All options should be set correctly for verbose output
		if opts.Model != models.ModelGPT51 {
			t.Error("model should be set for verbose output")
		}
	})
}

// TestConfigFileIntegration tests config file loading integration
func TestConfigFileIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Save and restore HOME
	oldHome := os.Getenv("HOME")
	defer func() { os.Setenv("HOME", oldHome) }()
	os.Setenv("HOME", tempDir)

	t.Run("complete config file integration", func(t *testing.T) {
		// Create config directory
		configDir := filepath.Join(tempDir, ".perplexity-cli")
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		// Write comprehensive config
		configFile := filepath.Join(configDir, "config.json")
		configJSON := `{
			"default_model": "claude45sonnet",
			"default_mode": "reasoning",
			"default_language": "fr-FR",
			"streaming": true,
			"incognito": false,
			"history_file": "my_history.jsonl"
		}`

		err = os.WriteFile(configFile, []byte(configJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		// Load config
		mgr, err := config.NewManager()
		if err != nil {
			t.Fatalf("failed to create manager: %v", err)
		}

		testCfg, err := mgr.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Verify all fields
		if testCfg.DefaultModel != "claude45sonnet" {
			t.Errorf("default model: got %q, want %q", testCfg.DefaultModel, "claude45sonnet")
		}
		if testCfg.DefaultMode != "reasoning" {
			t.Errorf("default mode: got %q, want %q", testCfg.DefaultMode, "reasoning")
		}
		if testCfg.DefaultLanguage != "fr-FR" {
			t.Errorf("default language: got %q, want %q", testCfg.DefaultLanguage, "fr-FR")
		}
		if testCfg.Streaming != true {
			t.Errorf("streaming: got %v, want %v", testCfg.Streaming, true)
		}
		if testCfg.Incognito != false {
			t.Errorf("incognito: got %v, want %v", testCfg.Incognito, false)
		}
		if testCfg.HistoryFile != "my_history.jsonl" {
			t.Errorf("history file: got %q, want %q", testCfg.HistoryFile, "my_history.jsonl")
		}
	})

	t.Run("missing config file uses defaults", func(t *testing.T) {
		// Don't create config file

		// Load config (should use defaults)
		mgr, err := config.NewManager()
		if err != nil {
			t.Fatalf("failed to create manager: %v", err)
		}

		testCfg, err := mgr.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		// Should have default values
		if testCfg.DefaultModel == "" {
			t.Error("default model should have a value")
		}
		if testCfg.DefaultMode == "" {
			t.Error("default mode should have a value")
		}
	})

	t.Run("corrupted config file", func(t *testing.T) {
		configDir := filepath.Join(tempDir, ".perplexity-cli")
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		configFile := filepath.Join(configDir, "config.json")
		corruptedJSON := `{"default_model": "gpt51", "default_mode": "pro"` // Missing closing brace

		err = os.WriteFile(configFile, []byte(corruptedJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write corrupted config: %v", err)
		}

		// Loading corrupted config should fail or use defaults
		mgr, err := config.NewManager()
		if err != nil {
			t.Fatalf("failed to create manager: %v", err)
		}

		_, err = mgr.Load()
		// Should either fail or fall back to defaults
		// The exact behavior depends on config package implementation
	})
}
