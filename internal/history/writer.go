// Package history manages query history.
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/diogo/perplexity-go/pkg/models"
)

// Writer handles writing history entries.
type Writer struct {
	path string
}

// NewWriter creates a new history writer.
func NewWriter(path string) (*Writer, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	return &Writer{path: path}, nil
}

// Append adds a new entry to the history file.
func (w *Writer) Append(entry models.HistoryEntry) error {
	// Open file in append mode
	file, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal history entry: %w", err)
	}

	// Write with newline
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write history entry: %w", err)
	}

	return nil
}

// Reader handles reading history entries.
type Reader struct {
	path string
}

// NewReader creates a new history reader.
func NewReader(path string) *Reader {
	return &Reader{path: path}
}

// ReadAll reads all history entries.
func (r *Reader) ReadAll() ([]models.HistoryEntry, error) {
	file, err := os.Open(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.HistoryEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var entries []models.HistoryEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry models.HistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading history: %w", err)
	}

	return entries, nil
}

// ReadLast reads the last n entries.
func (r *Reader) ReadLast(n int) ([]models.HistoryEntry, error) {
	entries, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(entries) <= n {
		return entries, nil
	}

	return entries[len(entries)-n:], nil
}

// Clear removes all history entries.
func (r *Reader) Clear() error {
	// Truncate the file
	return os.Truncate(r.path, 0)
}

// Search finds entries matching the query.
func (r *Reader) Search(query string) ([]models.HistoryEntry, error) {
	entries, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var results []models.HistoryEntry
	for _, entry := range entries {
		if containsIgnoreCase(entry.Query, query) {
			results = append(results, entry)
		}
	}

	return results, nil
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || containsLower(toLower(s), toLower(substr)))
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
