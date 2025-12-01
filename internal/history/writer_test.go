package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/diogo/perplexity-go/pkg/models"
)

func TestNewWriter(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "history.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if w.path != path {
		t.Errorf("path = %q, want %q", w.path, path)
	}

	// Verify directory was created
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

func TestWriterAppend(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	entry := models.HistoryEntry{
		Query:    "test query",
		Mode:     "default",
		Model:    "pplx_pro",
		Response: "test response",
	}

	if err := w.Append(entry); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("History file was not created")
	}

	// Read and verify
	reader := NewReader(path)
	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	if entries[0].Query != "test query" {
		t.Errorf("Query = %q, want %q", entries[0].Query, "test query")
	}
	if entries[0].Timestamp.IsZero() {
		t.Error("Timestamp should be set automatically")
	}
}

func TestWriterAppendMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	for i := 0; i < 5; i++ {
		entry := models.HistoryEntry{
			Query: "query " + string(rune('0'+i)),
			Mode:  "default",
		}
		if err := w.Append(entry); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	reader := NewReader(path)
	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("len(entries) = %d, want 5", len(entries))
	}
}

func TestReaderReadAll(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	// Write test data
	content := `{"timestamp":"2024-01-01T10:00:00Z","query":"query1","mode":"default"}
{"timestamp":"2024-01-01T11:00:00Z","query":"query2","mode":"fast"}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reader := NewReader(path)
	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}

	if entries[0].Query != "query1" {
		t.Errorf("entries[0].Query = %q, want %q", entries[0].Query, "query1")
	}
	if entries[1].Query != "query2" {
		t.Errorf("entries[1].Query = %q, want %q", entries[1].Query, "query2")
	}
}

func TestReaderReadAllEmpty(t *testing.T) {
	reader := NewReader("/nonexistent/path/history.jsonl")
	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestReaderReadLast(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	w, _ := NewWriter(path)
	for i := 0; i < 10; i++ {
		w.Append(models.HistoryEntry{
			Query:     "query " + string(rune('0'+i)),
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
		})
	}

	reader := NewReader(path)

	// Read last 3
	entries, err := reader.ReadLast(3)
	if err != nil {
		t.Fatalf("ReadLast() error = %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("len(entries) = %d, want 3", len(entries))
	}

	// Read more than available
	entries, err = reader.ReadLast(20)
	if err != nil {
		t.Fatalf("ReadLast() error = %v", err)
	}

	if len(entries) != 10 {
		t.Errorf("len(entries) = %d, want 10", len(entries))
	}
}

func TestReaderSearch(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	w, _ := NewWriter(path)
	w.Append(models.HistoryEntry{Query: "how to cook pasta"})
	w.Append(models.HistoryEntry{Query: "best restaurants nearby"})
	w.Append(models.HistoryEntry{Query: "pasta recipes italian"})
	w.Append(models.HistoryEntry{Query: "weather tomorrow"})

	reader := NewReader(path)

	// Search for "pasta"
	entries, err := reader.Search("pasta")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(entries))
	}

	// Case insensitive search
	entries, err = reader.Search("PASTA")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2 (case insensitive)", len(entries))
	}

	// Search no results
	entries, err = reader.Search("nonexistent")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestReaderClear(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	w, _ := NewWriter(path)
	w.Append(models.HistoryEntry{Query: "test"})
	w.Append(models.HistoryEntry{Query: "test2"})

	reader := NewReader(path)

	// Verify entries exist
	entries, _ := reader.ReadAll()
	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries before clear")
	}

	// Clear
	if err := reader.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify empty
	entries, _ = reader.ReadAll()
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0 after clear", len(entries))
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "World", true},
		{"Hello World", "xyz", false},
		{"hello", "hello", true},
		{"hello", "HELLO", true},
		{"short", "longer", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestWriterAppend_Unicode(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	entry := models.HistoryEntry{
		Query:    "こんにちは世界 🌍 café naïve résumé",
		Mode:     "default",
		Model:    "pplx_pro",
		Response: "Unicode response: αβγδε",
	}

	if err := w.Append(entry); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	reader := NewReader(path)
	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	if entries[0].Query != entry.Query {
		t.Errorf("Query = %q, want %q", entries[0].Query, entry.Query)
	}
}

func TestReaderReadAll_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	// Write invalid JSON
	content := `{"query": "valid"}
invalid json line
{"query": "also valid"}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reader := NewReader(path)
	entries, err := reader.ReadAll()
	// Should skip invalid lines and return valid entries
	if err != nil {
		t.Errorf("ReadAll() should skip invalid lines, got error: %v", err)
	}
	// Should have 2 valid entries
	if len(entries) != 2 {
		t.Errorf("len(entries) = %d, want 2 (skipped invalid line)", len(entries))
	}
}

func TestReaderReadLast_Zero(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	w, _ := NewWriter(path)
	w.Append(models.HistoryEntry{Query: "test1"})
	w.Append(models.HistoryEntry{Query: "test2"})

	reader := NewReader(path)
	// Zero should return empty slice
	entries, err := reader.ReadLast(0)
	if err != nil {
		t.Fatalf("ReadLast() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

func TestReaderSearch_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "history.jsonl")

	// Create empty file
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	reader := NewReader(path)
	entries, err := reader.Search("test")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}
