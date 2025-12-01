package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/diogo/perplexity-go/pkg/models"
)

func TestNewRenderer(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
	if r == nil {
		t.Fatal("NewRenderer() returned nil")
	}
	if r.width != 80 {
		t.Errorf("width = %d, want 80", r.width)
	}
	if !r.useColors {
		t.Error("useColors should be true by default")
	}
}

func TestNewRendererWithOptions(t *testing.T) {
	var buf bytes.Buffer

	r, err := NewRendererWithOptions(&buf, 120, false)
	if err != nil {
		t.Fatalf("NewRendererWithOptions() error = %v", err)
	}

	if r.width != 120 {
		t.Errorf("width = %d, want 120", r.width)
	}
	if r.useColors {
		t.Error("useColors should be false")
	}
}

func TestRenderMarkdown(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	err := r.RenderMarkdown("# Hello World\n\nThis is a test.")
	if err != nil {
		t.Fatalf("RenderMarkdown() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Output should not be empty")
	}
}

func TestRenderStreamChunk(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with delta
	chunk := models.StreamChunk{Delta: "Hello "}
	r.RenderStreamChunk(chunk)

	if buf.String() != "Hello " {
		t.Errorf("Output = %q, want %q", buf.String(), "Hello ")
	}

	// Test with text
	buf.Reset()
	chunk = models.StreamChunk{Text: "World"}
	r.RenderStreamChunk(chunk)

	if buf.String() != "World" {
		t.Errorf("Output = %q, want %q", buf.String(), "World")
	}
}

func TestRenderError(t *testing.T) {
	var buf bytes.Buffer

	// With colors
	r, _ := NewRendererWithOptions(&buf, 80, true)
	r.RenderError(errors.New("test error"))
	if !strings.Contains(buf.String(), "test error") {
		t.Error("Output should contain error message")
	}

	// Without colors
	buf.Reset()
	r, _ = NewRendererWithOptions(&buf, 80, false)
	r.RenderError(errors.New("test error"))
	if buf.String() != "Error: test error\n" {
		t.Errorf("Output = %q, want %q", buf.String(), "Error: test error\n")
	}
}

func TestRenderSuccess(t *testing.T) {
	var buf bytes.Buffer

	// With colors
	r, _ := NewRendererWithOptions(&buf, 80, true)
	r.RenderSuccess("Operation completed")
	if !strings.Contains(buf.String(), "Operation completed") {
		t.Error("Output should contain success message")
	}

	// Without colors
	buf.Reset()
	r, _ = NewRendererWithOptions(&buf, 80, false)
	r.RenderSuccess("Operation completed")
	if buf.String() != "Operation completed\n" {
		t.Errorf("Output = %q, want %q", buf.String(), "Operation completed\n")
	}
}

func TestRenderWarning(t *testing.T) {
	var buf bytes.Buffer

	// With colors
	r, _ := NewRendererWithOptions(&buf, 80, true)
	r.RenderWarning("Be careful")
	if !strings.Contains(buf.String(), "Be careful") {
		t.Error("Output should contain warning message")
	}

	// Without colors
	buf.Reset()
	r, _ = NewRendererWithOptions(&buf, 80, false)
	r.RenderWarning("Be careful")
	if buf.String() != "Warning: Be careful\n" {
		t.Errorf("Output = %q, want %q", buf.String(), "Warning: Be careful\n")
	}
}

func TestRenderInfo(t *testing.T) {
	var buf bytes.Buffer

	// With colors
	r, _ := NewRendererWithOptions(&buf, 80, true)
	r.RenderInfo("Some information")
	if !strings.Contains(buf.String(), "Some information") {
		t.Error("Output should contain info message")
	}

	// Without colors
	buf.Reset()
	r, _ = NewRendererWithOptions(&buf, 80, false)
	r.RenderInfo("Some information")
	if buf.String() != "Some information\n" {
		t.Errorf("Output = %q, want %q", buf.String(), "Some information\n")
	}
}

func TestRenderTitle(t *testing.T) {
	var buf bytes.Buffer

	// With colors
	r, _ := NewRendererWithOptions(&buf, 80, true)
	r.RenderTitle("My Title")
	if !strings.Contains(buf.String(), "My Title") {
		t.Error("Output should contain title")
	}

	// Without colors
	buf.Reset()
	r, _ = NewRendererWithOptions(&buf, 80, false)
	r.RenderTitle("My Title")
	output := buf.String()
	if !strings.Contains(output, "MY TITLE") {
		t.Error("Output should contain uppercase title")
	}
	if !strings.Contains(output, "========") {
		t.Error("Output should contain underline")
	}
}

func TestRenderCitations(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	citations := []models.Citation{
		{URL: "https://example.com/1", Title: "Example 1"},
		{URL: "https://example.com/2", Title: "Example 2"},
	}

	r.RenderCitations(citations)

	output := buf.String()
	if !strings.Contains(output, "Sources:") {
		t.Error("Output should contain 'Sources:' header")
	}
	if !strings.Contains(output, "[1]") {
		t.Error("Output should contain citation number [1]")
	}
	if !strings.Contains(output, "[2]") {
		t.Error("Output should contain citation number [2]")
	}
	if !strings.Contains(output, "Example 1") {
		t.Error("Output should contain citation title")
	}
}

func TestRenderCitationsEmpty(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	r.RenderCitations(nil)
	if buf.String() != "" {
		t.Error("Empty citations should produce no output")
	}

	r.RenderCitations([]models.Citation{})
	if buf.String() != "" {
		t.Error("Empty citations slice should produce no output")
	}
}

func TestRenderCitationsURLOnly(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	citations := []models.Citation{
		{URL: "https://example.com/page"},
	}

	r.RenderCitations(citations)

	output := buf.String()
	if !strings.Contains(output, "https://example.com/page") {
		t.Error("Output should contain URL when no title")
	}
}

func TestRenderSpinner(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, true)

	for i := 0; i < len(SpinnerChars)+1; i++ {
		buf.Reset()
		r.RenderSpinner(i)
		output := buf.String()
		if !strings.HasPrefix(output, "\r") {
			t.Error("Spinner should start with carriage return")
		}
	}
}

func TestClearLine(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, true)

	r.ClearLine()
	if !strings.Contains(buf.String(), "\r") {
		t.Error("ClearLine should contain carriage return")
	}
}

func TestNewLine(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, true)

	r.NewLine()
	if buf.String() != "\n" {
		t.Errorf("NewLine() output = %q, want %q", buf.String(), "\n")
	}
}

func TestRenderResponse(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	resp := &models.SearchResponse{
		Blocks: []models.ResponseBlock{
			{
				MarkdownBlock: &models.MarkdownBlock{
					Answer: "This is the answer.",
					Citations: []models.Citation{
						{URL: "https://example.com", Title: "Source"},
					},
				},
			},
		},
	}

	err := r.RenderResponse(resp)
	if err != nil {
		t.Fatalf("RenderResponse() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "This is the answer") {
		t.Error("Output should contain the answer")
	}
	if !strings.Contains(output, "Sources:") {
		t.Error("Output should contain citations")
	}
}

func TestRenderResponseNoBlocks(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	resp := &models.SearchResponse{
		Text: "Raw text response",
	}

	err := r.RenderResponse(resp)
	if err != nil {
		t.Fatalf("RenderResponse() error = %v", err)
	}

	if !strings.Contains(buf.String(), "Raw text response") {
		t.Error("Output should contain raw text when no blocks")
	}
}

func TestSpinnerChars(t *testing.T) {
	if len(SpinnerChars) == 0 {
		t.Error("SpinnerChars should not be empty")
	}

	// Verify all are valid unicode characters
	for i, char := range SpinnerChars {
		if char == "" {
			t.Errorf("SpinnerChars[%d] should not be empty", i)
		}
	}
}
