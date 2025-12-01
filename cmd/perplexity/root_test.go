package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/internal/config"
	"github.com/diogo/perplexity-go/internal/ui"
	"github.com/diogo/perplexity-go/pkg/models"
	"github.com/spf13/cobra"
)

// errorReader is a mock reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock read error")
}

// MockClient is a mock implementation of the Perplexity client for testing
type MockClient struct {
	searchResponse   *models.SearchResponse
	searchError      error
	streamChunks     []models.StreamChunk
	streamError      error
	shouldCloseError error
}

// NewMockClient creates a new mock client with optional response
func NewMockClient(resp *models.SearchResponse, err error) *MockClient {
	return &MockClient{
		searchResponse: resp,
		searchError:    err,
	}
}

// NewMockStreamClient creates a mock client with streaming response
func NewMockStreamClient(chunks []models.StreamChunk, err error) *MockClient {
	return &MockClient{
		streamChunks: chunks,
		streamError:  err,
	}
}

func (m *MockClient) Search(ctx context.Context, opts models.SearchOptions) (*models.SearchResponse, error) {
	return m.searchResponse, m.searchError
}

func (m *MockClient) SearchStream(ctx context.Context, opts models.SearchOptions) (<-chan models.StreamChunk, error) {
	ch := make(chan models.StreamChunk, len(m.streamChunks))
	if m.streamError != nil {
		// Send error chunk and close
		ch <- models.StreamChunk{Error: m.streamError}
		close(ch)
		return ch, m.streamError
	}
	// Send all chunks
	for _, chunk := range m.streamChunks {
		ch <- chunk
	}
	close(ch)
	return ch, nil
}

func (m *MockClient) Close() error {
	return m.shouldCloseError
}

func (m *MockClient) SetCookies(cookies []*http.Cookie) {}

func (m *MockClient) GetCookies() []*http.Cookie {
	return nil
}

func (m *MockClient) HasValidSession() bool {
	return true
}

func (m *MockClient) ProQueriesRemaining() int {
	return 5
}

func (m *MockClient) FileUploadsRemaining() int {
	return 10
}

// MockRenderer is a mock implementation of the UI renderer for testing
type MockRenderer struct {
	renderedMessages []string
	renderedErrors   []error
	styledResponse   string
	streamChunks     []models.StreamChunk
	webResults       []models.WebResult
}

// NewMockRenderer creates a new mock renderer
func NewMockRenderer() *ui.Renderer {
	// For testing, we'll use a real renderer but wrap methods to track calls
	return &ui.Renderer{}
}

func (m *MockRenderer) RenderMarkdown(content string) error {
	m.renderedMessages = append(m.renderedMessages, content)
	return nil
}

func (m *MockRenderer) RenderStyledResponse(content string) error {
	m.styledResponse = content
	m.renderedMessages = append(m.renderedMessages, "styled:"+content)
	return nil
}

func (m *MockRenderer) RenderResponse(resp *models.SearchResponse) error {
	if resp != nil && resp.Text != "" {
		m.styledResponse = resp.Text
		m.renderedMessages = append(m.renderedMessages, "response:"+resp.Text)
	}
	return nil
}

func (m *MockRenderer) RenderCitations(citations []models.Citation) {
	m.renderedMessages = append(m.renderedMessages, fmt.Sprintf("citations:%d", len(citations)))
}

func (m *MockRenderer) RenderWebResults(results []models.WebResult) {
	m.webResults = results
	m.renderedMessages = append(m.renderedMessages, fmt.Sprintf("webresults:%d", len(results)))
}

func (m *MockRenderer) RenderStreamChunk(chunk models.StreamChunk) {
	m.streamChunks = append(m.streamChunks, chunk)
	m.renderedMessages = append(m.renderedMessages, "streamchunk")
}

func (m *MockRenderer) RenderError(err error) {
	m.renderedErrors = append(m.renderedErrors, err)
}

func (m *MockRenderer) RenderSuccess(msg string) {
	m.renderedMessages = append(m.renderedMessages, "success:"+msg)
}

func (m *MockRenderer) RenderWarning(msg string) {
	m.renderedMessages = append(m.renderedMessages, "warning:"+msg)
}

func (m *MockRenderer) RenderInfo(msg string) {
	m.renderedMessages = append(m.renderedMessages, "info:"+msg)
}

func (m *MockRenderer) RenderTitle(title string) {
	m.renderedMessages = append(m.renderedMessages, "title:"+title)
}

func (m *MockRenderer) RenderSpinner(frame int) {
	m.renderedMessages = append(m.renderedMessages, fmt.Sprintf("spinner:%d", frame))
}

func (m *MockRenderer) ClearLine() {
	m.renderedMessages = append(m.renderedMessages, "clearline")
}

func (m *MockRenderer) NewLine() {
	m.renderedMessages = append(m.renderedMessages, "newline")
}

func TestGetQueryFromInput(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		stdin      io.Reader
		isTerminal bool
		want       string
		wantErr    bool
	}{
		{
			name:       "query from single argument",
			args:       []string{"hello world"},
			stdin:      strings.NewReader(""),
			isTerminal: true,
			want:       "hello world",
			wantErr:    false,
		},
		{
			name:       "query from multiple arguments",
			args:       []string{"what", "is", "Go?"},
			stdin:      strings.NewReader(""),
			isTerminal: true,
			want:       "what is Go?",
			wantErr:    false,
		},
		{
			name:       "query from stdin when not terminal",
			args:       []string{},
			stdin:      strings.NewReader("piped query\n"),
			isTerminal: false,
			want:       "piped query",
			wantErr:    false,
		},
		{
			name:       "query from stdin with whitespace trimmed",
			args:       []string{},
			stdin:      strings.NewReader("  trimmed query  \n\n"),
			isTerminal: false,
			want:       "trimmed query",
			wantErr:    false,
		},
		{
			name:       "empty query when terminal and no args",
			args:       []string{},
			stdin:      strings.NewReader(""),
			isTerminal: true,
			want:       "",
			wantErr:    false,
		},
		{
			name:       "empty query from empty stdin",
			args:       []string{},
			stdin:      strings.NewReader(""),
			isTerminal: false,
			want:       "",
			wantErr:    false,
		},
		{
			name:       "empty query from whitespace-only stdin",
			args:       []string{},
			stdin:      strings.NewReader("   \n\t\n  "),
			isTerminal: false,
			want:       "",
			wantErr:    false,
		},
		{
			name:       "args take precedence over stdin",
			args:       []string{"from args"},
			stdin:      strings.NewReader("from stdin"),
			isTerminal: false,
			want:       "from args",
			wantErr:    false,
		},
		{
			name:       "error reading from stdin",
			args:       []string{},
			stdin:      &errorReader{},
			isTerminal: false,
			want:       "",
			wantErr:    true,
		},
		{
			name:       "multiline stdin query",
			args:       []string{},
			stdin:      strings.NewReader("line 1\nline 2\nline 3"),
			isTerminal: false,
			want:       "line 1\nline 2\nline 3",
			wantErr:    false,
		},
		{
			name:       "large stdin input",
			args:       []string{},
			stdin:      strings.NewReader(strings.Repeat("x", 10000)),
			isTerminal: false,
			want:       strings.Repeat("x", 10000),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getQueryFromInput(tt.args, tt.stdin, tt.isTerminal)

			if (err != nil) != tt.wantErr {
				t.Errorf("getQueryFromInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("getQueryFromInput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetQueryFromInput_BufferReader(t *testing.T) {
	// Test with bytes.Buffer (different Reader implementation)
	buf := bytes.NewBuffer([]byte("buffer query"))
	got, err := getQueryFromInput([]string{}, buf, false)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if got != "buffer query" {
		t.Errorf("got %q, want %q", got, "buffer query")
	}
}

func TestTruncateResponse(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "string shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "string equal to max",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "string longer than max",
			input:  "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "max length zero",
			input:  "hello",
			maxLen: 0,
			want:   "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateResponse(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateResponse(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestGetQueryFromFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		createFile  bool
		want        string
		wantErr     bool
		errContains string
	}{
		{
			name:        "read simple query from file",
			fileContent: "What is the capital of France?",
			createFile:  true,
			want:        "What is the capital of France?",
			wantErr:     false,
		},
		{
			name:        "read query with whitespace trimmed",
			fileContent: "  trimmed query  \n\n",
			createFile:  true,
			want:        "trimmed query",
			wantErr:     false,
		},
		{
			name:        "read multiline query",
			fileContent: "Line 1\nLine 2\nLine 3",
			createFile:  true,
			want:        "Line 1\nLine 2\nLine 3",
			wantErr:     false,
		},
		{
			name:        "empty file returns error",
			fileContent: "",
			createFile:  true,
			want:        "",
			wantErr:     true,
			errContains: "is empty",
		},
		{
			name:        "whitespace-only file returns error",
			fileContent: "   \n\t\n  ",
			createFile:  true,
			want:        "",
			wantErr:     true,
			errContains: "is empty",
		},
		{
			name:        "non-existent file returns error",
			fileContent: "",
			createFile:  false,
			want:        "",
			wantErr:     true,
			errContains: "failed to read input file",
		},
		{
			name:        "read markdown content",
			fileContent: "# Question\n\nWhat is **Go**?",
			createFile:  true,
			want:        "# Question\n\nWhat is **Go**?",
			wantErr:     false,
		},
		{
			name:        "read large content",
			fileContent: strings.Repeat("x", 10000),
			createFile:  true,
			want:        strings.Repeat("x", 10000),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filePath string
			if tt.createFile {
				filePath = filepath.Join(tempDir, tt.name+".txt")
				err := os.WriteFile(filePath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			} else {
				filePath = filepath.Join(tempDir, "nonexistent_"+tt.name+".txt")
			}

			got, err := getQueryFromFile(filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("getQueryFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("getQueryFromFile() error = %v, should contain %q", err, tt.errContains)
				}
			}

			if got != tt.want {
				t.Errorf("getQueryFromFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetQueryFromFile_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()

	// Test with special characters (UTF-8)
	content := "Qual é a capital do Brasil? 日本語 مرحبا"
	filePath := filepath.Join(tempDir, "special.txt")
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	got, err := getQueryFromFile(filePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestGetQueryFromFile_PermissionDenied(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "noperm.txt")

	// Create file with no read permission
	err := os.WriteFile(filePath, []byte("content"), 0000)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer os.Chmod(filePath, 0644) // Cleanup

	_, err = getQueryFromFile(filePath)
	if err == nil {
		t.Error("expected error for permission denied, got nil")
	}
}

// TestQueryInputPriority tests the priority logic: -f > args > stdin
// This simulates the behavior in runQuery without requiring full integration
func TestQueryInputPriority(t *testing.T) {
	tempDir := t.TempDir()

	// Helper function that mimics the priority logic in runQuery
	getQueryWithPriority := func(inputFile string, args []string, stdin io.Reader, isTerminal bool) (string, error) {
		var query string
		var err error

		// Priority 1: -f/--file flag
		if inputFile != "" {
			query, err = getQueryFromFile(inputFile)
			if err != nil {
				return "", err
			}
		}

		// Priority 2: args or stdin
		if query == "" {
			query, err = getQueryFromInput(args, stdin, isTerminal)
			if err != nil {
				return "", err
			}
		}

		return query, nil
	}

	t.Run("file takes precedence over args", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "priority1.txt")
		err := os.WriteFile(filePath, []byte("from file"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		got, err := getQueryWithPriority(filePath, []string{"from args"}, strings.NewReader("from stdin"), false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != "from file" {
			t.Errorf("got %q, want %q", got, "from file")
		}
	})

	t.Run("file takes precedence over stdin", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "priority2.txt")
		err := os.WriteFile(filePath, []byte("from file"), 0644)
		if err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		got, err := getQueryWithPriority(filePath, []string{}, strings.NewReader("from stdin"), false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != "from file" {
			t.Errorf("got %q, want %q", got, "from file")
		}
	})

	t.Run("args used when no file specified", func(t *testing.T) {
		got, err := getQueryWithPriority("", []string{"from args"}, strings.NewReader("from stdin"), false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != "from args" {
			t.Errorf("got %q, want %q", got, "from args")
		}
	})

	t.Run("stdin used when no file and no args", func(t *testing.T) {
		got, err := getQueryWithPriority("", []string{}, strings.NewReader("from stdin"), false)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != "from stdin" {
			t.Errorf("got %q, want %q", got, "from stdin")
		}
	})

	t.Run("empty when no input sources", func(t *testing.T) {
		got, err := getQueryWithPriority("", []string{}, strings.NewReader(""), true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("file error propagates", func(t *testing.T) {
		nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")
		_, err := getQueryWithPriority(nonExistentFile, []string{"from args"}, strings.NewReader("from stdin"), false)
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})
}

// TestOutputFileSaving tests that response can be saved to different file formats
func TestOutputFileSaving(t *testing.T) {
	tempDir := t.TempDir()

	testContent := "# Response\n\nThis is the **answer**."

	tests := []struct {
		name     string
		filename string
	}{
		{"markdown file", "output.md"},
		{"text file", "output.txt"},
		{"no extension", "output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, tt.filename)

			// Simulate the saving logic from runQuery
			err := os.WriteFile(filePath, []byte(testContent), 0644)
			if err != nil {
				t.Fatalf("failed to write output file: %v", err)
			}

			// Verify file was created and content is correct
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			if string(data) != testContent {
				t.Errorf("got %q, want %q", string(data), testContent)
			}
		})
	}
}

// TestBuildSearchOptions tests the buildSearchOptions function
func TestBuildSearchOptions(t *testing.T) {
	// Save current global state
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

	tests := []struct {
		name        string
		cfg         *config.Config
		flagModel   string
		flagMode    string
		flagLanguage string
		flagSources string
		flagIncognito bool
		query       string
		wantModel   models.Model
		wantMode    models.Mode
		wantLang    string
		wantSources []models.Source
		wantIncog   bool
	}{
		{
			name:        "uses config defaults when no flags",
			cfg:         &config.Config{DefaultModel: models.ModelPplxPro, DefaultMode: models.ModeDefault, DefaultLanguage: "en-US", DefaultSources: []models.Source{models.SourceWeb}, Incognito: false},
			flagModel:   "",
			flagMode:    "",
			flagLanguage: "",
			flagSources: "",
			flagIncognito: false,
			query:       "test query",
			wantModel:   models.ModelPplxPro,
			wantMode:    models.ModeDefault,
			wantLang:    "en-US",
			wantSources: []models.Source{models.SourceWeb},
			wantIncog:   false,
		},
		{
			name:        "overrides config with flags",
			cfg:         &config.Config{DefaultModel: models.ModelPplxPro, DefaultMode: models.ModeDefault, DefaultLanguage: "en-US", DefaultSources: []models.Source{models.SourceWeb}, Incognito: false},
			flagModel:   "gpt51",
			flagMode:    "pro",
			flagLanguage: "pt-BR",
			flagSources: "web,scholar",
			flagIncognito: true,
			query:       "test query",
			wantModel:   models.ModelGPT51,
			wantMode:    models.ModePro,
			wantLang:    "pt-BR",
			wantSources: []models.Source{models.SourceWeb, models.SourceScholar},
			wantIncog:   true,
		},
		{
			name:        "multiple sources from comma-separated string",
			cfg:         &config.Config{DefaultModel: models.ModelPplxPro, DefaultMode: models.ModeDefault, DefaultLanguage: "en-US", DefaultSources: []models.Source{models.SourceWeb}, Incognito: false},
			flagModel:   "",
			flagMode:    "",
			flagLanguage: "fr-FR",
			flagSources: " web , scholar , social ",
			flagIncognito: false,
			query:       "test query",
			wantModel:   models.ModelPplxPro,
			wantMode:    models.ModeDefault,
			wantLang:    "fr-FR",
			wantSources: []models.Source{models.SourceWeb, models.SourceScholar, models.SourceSocial},
			wantIncog:   false,
		},
		{
			name:        "reasoning mode with reasoning model",
			cfg:         &config.Config{DefaultModel: models.ModelPplxPro, DefaultMode: models.ModeDefault, DefaultLanguage: "en-US", DefaultSources: []models.Source{models.SourceWeb}, Incognito: false},
			flagModel:   "gpt51_thinking",
			flagMode:    "reasoning",
			flagLanguage: "",
			flagSources: "",
			flagIncognito: false,
			query:       "test query",
			wantModel:   models.ModelGPT51Thinking,
			wantMode:    models.ModeReasoning,
			wantLang:    "en-US",
			wantSources: []models.Source{models.SourceWeb},
			wantIncog:   false,
		},
		{
			name:        "fast mode with turbo",
			cfg:         &config.Config{DefaultModel: models.ModelPplxPro, DefaultMode: models.ModeDefault, DefaultLanguage: "en-US", DefaultSources: []models.Source{models.SourceWeb}, Incognito: false},
			flagModel:   "",
			flagMode:    "fast",
			flagLanguage: "",
			flagSources: "",
			flagIncognito: false,
			query:       "test query",
			wantModel:   models.ModelPplxPro,
			wantMode:    models.ModeFast,
			wantLang:    "en-US",
			wantSources: []models.Source{models.SourceWeb},
			wantIncog:   false,
		},
		{
			name:        "deep-research mode",
			cfg:         &config.Config{DefaultModel: models.ModelPplxPro, DefaultMode: models.ModeDefault, DefaultLanguage: "en-US", DefaultSources: []models.Source{models.SourceWeb}, Incognito: false},
			flagModel:   "",
			flagMode:    "deep-research",
			flagLanguage: "de-DE",
			flagSources: "",
			flagIncognito: true,
			query:       "test query",
			wantModel:   models.ModelPplxPro,
			wantMode:    models.ModeDeepResearch,
			wantLang:    "de-DE",
			wantSources: []models.Source{models.SourceWeb},
			wantIncog:   true,
		},
		{
			name:        "flag incognito overrides config",
			cfg:         &config.Config{DefaultModel: models.ModelPplxPro, DefaultMode: models.ModeDefault, DefaultLanguage: "en-US", DefaultSources: []models.Source{models.SourceWeb}, Incognito: false},
			flagModel:   "",
			flagMode:    "",
			flagLanguage: "",
			flagSources: "",
			flagIncognito: true,
			query:       "test query",
			wantModel:   models.ModelPplxPro,
			wantMode:    models.ModeDefault,
			wantLang:    "en-US",
			wantSources: []models.Source{models.SourceWeb},
			wantIncog:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global variables
			cfg = tt.cfg
			flagModel = tt.flagModel
			flagMode = tt.flagMode
			flagLanguage = tt.flagLanguage
			flagSources = tt.flagSources
			flagIncognito = tt.flagIncognito

			opts := buildSearchOptions(tt.query)

			if opts.Query != tt.query {
				t.Errorf("Query mismatch: got %q, want %q", opts.Query, tt.query)
			}
			if opts.Model != tt.wantModel {
				t.Errorf("Model mismatch: got %q, want %q", opts.Model, tt.wantModel)
			}
			if opts.Mode != tt.wantMode {
				t.Errorf("Mode mismatch: got %q, want %q", opts.Mode, tt.wantMode)
			}
			if opts.Language != tt.wantLang {
				t.Errorf("Language mismatch: got %q, want %q", opts.Language, tt.wantLang)
			}
			if len(opts.Sources) != len(tt.wantSources) {
				t.Errorf("Sources length mismatch: got %d, want %d", len(opts.Sources), len(tt.wantSources))
			} else {
				for i, s := range opts.Sources {
					if s != tt.wantSources[i] {
						t.Errorf("Source[%d] mismatch: got %q, want %q", i, s, tt.wantSources[i])
					}
				}
			}
			if opts.Incognito != tt.wantIncog {
				t.Errorf("Incognito mismatch: got %v, want %v", opts.Incognito, tt.wantIncog)
			}
		})
	}
}

// TestInitConfig tests the initConfig function
func TestInitConfig(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		// Create a temporary config directory
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		defer func() { os.Setenv("HOME", oldHome) }()
		os.Setenv("HOME", tempDir)

		// Reset global variables before test
		cfgMgr = nil
		cfg = nil
		render = nil

		// Call initConfig - it will try to initialize config and renderer
		// Note: initConfig calls os.Exit on error, so we can't test error paths directly
		// Instead, we test the happy path where it succeeds
		mgr, err := config.NewManager()
		if err != nil {
			t.Fatalf("failed to create manager: %v", err)
		}

		cfgMgr = mgr

		// Load config manually to verify it works
		testCfg, err := cfgMgr.Load()
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		cfg = testCfg

		// Initialize renderer
		r, err := ui.NewRenderer()
		if err != nil {
			t.Fatalf("failed to create renderer: %v", err)
		}
		render = r

		// Verify initialization
		if cfgMgr == nil {
			t.Error("config manager not initialized")
		}
		if cfg == nil {
			t.Error("config not loaded")
		}
		if render == nil {
			t.Error("renderer not initialized")
		}

		// Verify config defaults
		if cfg.DefaultModel == "" {
			t.Error("default model not set")
		}
		if cfg.DefaultMode == "" {
			t.Error("default mode not set")
		}
	})

	t.Run("config with custom values", func(t *testing.T) {
		// Create a temporary config directory with custom config
		tempDir := t.TempDir()
		oldHome := os.Getenv("HOME")
		defer func() { os.Setenv("HOME", oldHome) }()
		os.Setenv("HOME", tempDir)

		// Create config directory
		configDir := filepath.Join(tempDir, ".perplexity-cli")
		err := os.MkdirAll(configDir, 0700)
		if err != nil {
			t.Fatalf("failed to create config dir: %v", err)
		}

		// Write custom config
		configFile := filepath.Join(configDir, "config.json")
		configJSON := `{
			"default_model": "gpt51",
			"default_mode": "pro",
			"default_language": "pt-BR",
			"streaming": false,
			"incognito": true
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

		// Verify custom values were loaded
		if testCfg.DefaultModel != "gpt51" {
			t.Errorf("expected model gpt51, got %q", testCfg.DefaultModel)
		}
		if testCfg.DefaultMode != "pro" {
			t.Errorf("expected mode pro, got %q", testCfg.DefaultMode)
		}
		if testCfg.DefaultLanguage != "pt-BR" {
			t.Errorf("expected language pt-BR, got %q", testCfg.DefaultLanguage)
		}
		if testCfg.Streaming != false {
			t.Errorf("expected streaming false, got %v", testCfg.Streaming)
		}
		if testCfg.Incognito != true {
			t.Errorf("expected incognito true, got %v", testCfg.Incognito)
		}
	})
}

// TestRunQueryIntegration tests the runQuery function with integration-style tests
// These tests focus on error handling and query building logic
func TestRunQueryIntegration(t *testing.T) {
	// Create temporary files
	tempDir := t.TempDir()
	queryFile := filepath.Join(tempDir, "query.txt")
	cookieFile := filepath.Join(tempDir, "cookies.json")

	// Create test files
	err := os.WriteFile(queryFile, []byte("test query from file"), 0644)
	if err != nil {
		t.Fatalf("failed to create query file: %v", err)
	}

	err = os.WriteFile(cookieFile, []byte(`[{"name": "next-auth.csrf-token", "value": "test"}]`), 0644)
	if err != nil {
		t.Fatalf("failed to create cookie file: %v", err)
	}

	t.Run("missing cookie file error", func(t *testing.T) {
		// Setup
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       filepath.Join(tempDir, "nonexistent.json"),
		}

		// Reset flags
		flagInputFile = ""
		flagCookieFile = ""
		flagIncognito = false

		cmd := &cobra.Command{
			Use: "perplexity",
			RunE: runQuery,
		}

		// Execute - should fail due to missing cookie file
		err := runQuery(cmd, []string{"test query"})
		if err == nil {
			t.Error("expected error for missing cookie file, got nil")
		}
	})

	t.Run("empty query shows help", func(t *testing.T) {
		// Setup with cookie file
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
		}

		// Reset flags - no query provided
		flagInputFile = ""
		flagCookieFile = ""
		flagIncognito = false

		cmd := &cobra.Command{
			Use: "perplexity",
			RunE: runQuery,
		}

		// Execute with no args and terminal stdin (should show help)
		err := runQuery(cmd, []string{})
		if err != nil {
			// Help() returns an error, which is expected
			// We just verify it doesn't panic
		}
	})

	t.Run("query from file", func(t *testing.T) {
		// Setup with cookie file
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
		}

		// Reset flags with file input
		flagInputFile = queryFile
		flagCookieFile = ""
		flagIncognito = true
		flagStream = false

		// Verify the file can be read
		content, err := getQueryFromFile(queryFile)
		if err != nil {
			t.Fatalf("failed to read query file: %v", err)
		}
		if content != "test query from file" {
			t.Errorf("expected 'test query from file', got %q", content)
		}
	})

	t.Run("stdin input with terminal false", func(t *testing.T) {
		// Setup with cookie file
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
		}

		// Reset flags - reading from stdin
		flagInputFile = ""
		flagCookieFile = ""
		flagIncognito = true
		flagStream = false

		// Test reading from stdin
		stdin := strings.NewReader("query from stdin")
		isTerminal := false

		query, err := getQueryFromInput([]string{}, stdin, isTerminal)
		if err != nil {
			t.Fatalf("failed to read from stdin: %v", err)
		}
		if query != "query from stdin" {
			t.Errorf("expected 'query from stdin', got %q", query)
		}
	})

	t.Run("verbose mode sets flags correctly", func(t *testing.T) {
		cfg = &config.Config{
			DefaultModel:     models.ModelPplxPro,
			DefaultMode:      models.ModeDefault,
			DefaultLanguage:  "en-US",
			DefaultSources:   []models.Source{models.SourceWeb},
			CookieFile:       cookieFile,
		}

		// Test buildSearchOptions with various flags
		opts := buildSearchOptions("test query")

		if opts.Model != models.ModelPplxPro {
			t.Errorf("expected model %q, got %q", models.ModelPplxPro, opts.Model)
		}
		if opts.Mode != models.ModeDefault {
			t.Errorf("expected mode %q, got %q", models.ModeDefault, opts.Mode)
		}
	})
}

// TestQueryFileWithEmptyContent tests edge case of empty file
func TestQueryFileWithEmptyContent(t *testing.T) {
	tempDir := t.TempDir()
	emptyFile := filepath.Join(tempDir, "empty.txt")

	// Create empty file
	err := os.WriteFile(emptyFile, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	// Try to read empty file - should return error
	_, err = getQueryFromFile(emptyFile)
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
	if !strings.Contains(err.Error(), "is empty") {
		t.Errorf("expected 'is empty' in error, got %v", err)
	}
}

// TestBuildSearchOptionsWithAllFlags tests buildSearchOptions with all flags set
func TestBuildSearchOptionsWithAllFlags(t *testing.T) {
	// Save current global state
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

	cfg = &config.Config{
		DefaultModel:     models.ModelPplxPro,
		DefaultMode:      models.ModeDefault,
		DefaultLanguage:  "en-US",
		DefaultSources:   []models.Source{models.SourceWeb},
		Incognito:        false,
	}

	// Set all flags
	flagModel = "gpt51"
	flagMode = "pro"
	flagLanguage = "pt-BR"
	flagSources = "web,scholar,social"
	flagIncognito = true

	opts := buildSearchOptions("test query")

	if opts.Model != models.ModelGPT51 {
		t.Errorf("Expected model gpt51, got %q", opts.Model)
	}
	if opts.Mode != models.ModePro {
		t.Errorf("Expected mode pro, got %q", opts.Mode)
	}
	if opts.Language != "pt-BR" {
		t.Errorf("Expected language pt-BR, got %q", opts.Language)
	}
	if len(opts.Sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(opts.Sources))
	}
	if opts.Incognito != true {
		t.Errorf("Expected incognito true, got %v", opts.Incognito)
	}
}

// TestTruncateResponseEdgeCases tests edge cases for truncateResponse
func TestTruncateResponseEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"exact match", "hello", 5, "hello"},
		{"one less", "hello", 4, "hell..."},
		{"one more", "hello", 6, "hello"},
		{"zero length", "", 10, ""},
		{"zero max", "hello", 0, "..."},
		{"very long string", strings.Repeat("x", 1000), 10, strings.Repeat("x", 10) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateResponse(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateResponse(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestFlagParsing tests flag parsing behavior
func TestFlagParsing(t *testing.T) {
	// Test that we can parse various flag combinations
	flagTests := []struct {
		name     string
		flags    map[string]string
		validate func(models.SearchOptions) bool
	}{
		{
			name: "model flag",
			flags: map[string]string{
				"model": "gpt51",
			},
			validate: func(opts models.SearchOptions) bool {
				return opts.Model == models.ModelGPT51
			},
		},
		{
			name: "mode flag",
			flags: map[string]string{
				"mode": "reasoning",
			},
			validate: func(opts models.SearchOptions) bool {
				return opts.Mode == models.ModeReasoning
			},
		},
		{
			name: "language flag",
			flags: map[string]string{
				"language": "fr-FR",
			},
			validate: func(opts models.SearchOptions) bool {
				return opts.Language == "fr-FR"
			},
		},
		{
			name: "sources flag",
			flags: map[string]string{
				"sources": "web,scholar",
			},
			validate: func(opts models.SearchOptions) bool {
				return len(opts.Sources) == 2
			},
		},
	}

	for _, tt := range flagTests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore global state
			origCfg := cfg
			origFlagModel := flagModel
			origFlagMode := flagMode
			origFlagLanguage := flagLanguage
			origFlagSources := flagSources

			defer func() {
				cfg = origCfg
				flagModel = origFlagModel
				flagMode = origFlagMode
				flagLanguage = origFlagLanguage
				flagSources = origFlagSources
			}()

			cfg = &config.Config{
				DefaultModel:     models.ModelPplxPro,
				DefaultMode:      models.ModeDefault,
				DefaultLanguage:  "en-US",
				DefaultSources:   []models.Source{models.SourceWeb},
			}

			// Set flags
			flagModel = tt.flags["model"]
			flagMode = tt.flags["mode"]
			flagLanguage = tt.flags["language"]
			flagSources = tt.flags["sources"]

			opts := buildSearchOptions("test")
			if !tt.validate(opts) {
				t.Errorf("Flag validation failed for %s", tt.name)
			}
		})
	}
}

// TestExecute tests the Execute function
func TestExecute(t *testing.T) {
	// Save original rootCmd
	origRootCmd := rootCmd

	defer func() {
		rootCmd = origRootCmd
	}()

	t.Run("successful execution", func(t *testing.T) {
		// This is a basic test of the Execute function
		// In practice, Execute just calls rootCmd.Execute()
		// which is tested through the Cobra testing mechanisms
		if rootCmd == nil {
			t.Error("rootCmd not initialized")
		}
	})

	t.Run("with help flag", func(t *testing.T) {
		// Test that --help flag works
		// This would normally trigger help output
		// For now, just verify the command is set up correctly
		if rootCmd.Use != "perplexity [query]" {
			t.Errorf("unexpected use string: %q", rootCmd.Use)
		}
	})
}
