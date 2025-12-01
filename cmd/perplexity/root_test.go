package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// errorReader is a mock reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("mock read error")
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
