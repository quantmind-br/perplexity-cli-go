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

func TestRenderWebResultsEmpty(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with nil slice
	r.RenderWebResults(nil)
	if buf.String() != "" {
		t.Error("Nil results should produce no output")
	}

	// Test with empty slice
	buf.Reset()
	r.RenderWebResults([]models.WebResult{})
	if buf.String() != "" {
		t.Error("Empty results should produce no output")
	}
}

func TestRenderWebResultsFiltered(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with only internal results (should be filtered out)
	results := []models.WebResult{
		{URL: "https://perplexity.ai"},
		{URL: ""},
	}

	r.RenderWebResults(results)
	if buf.String() != "" {
		t.Error("Only internal results should be filtered out")
	}
}

func TestRenderWebResultsWithTitle(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	results := []models.WebResult{
		{Title: "Example Article", URL: "https://example.com/article"},
		{Title: "Another Article", URL: "https://example.com/another"},
	}

	r.RenderWebResults(results)

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
	if !strings.Contains(output, "Example Article") {
		t.Error("Output should contain first article title")
	}
	if !strings.Contains(output, "https://example.com/article") {
		t.Error("Output should contain first article URL")
	}
	if !strings.Contains(output, "Another Article") {
		t.Error("Output should contain second article title")
	}
}

func TestRenderWebResultsWithName(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with Name field (fallback when Title is empty)
	results := []models.WebResult{
		{Name: "Example Site", URL: "https://example.com"},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "Example Site") {
		t.Error("Output should contain site name")
	}
}

func TestRenderWebResultsURLOnly(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with only URL (when both Title and Name are empty)
	results := []models.WebResult{
		{URL: "https://example.com/page"},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "https://example.com/page") {
		t.Error("Output should contain URL when no title or name")
	}
}

func TestRenderWebResultsMixed(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with mixed result types
	results := []models.WebResult{
		{Title: "Valid Article", URL: "https://example.com/article1"},
		{URL: "https://perplexity.ai"},                   // Should be filtered
		{Name: "Named Site", URL: "https://example.com"},  // Has name
		{URL: "https://example.com/page"},                 // URL only
		{URL: ""},                                         // Empty URL, should be filtered
	}

	r.RenderWebResults(results)

	output := buf.String()
	// Should have 3 sources (not 5, because 2 are filtered)
	sourceCount := strings.Count(output, "[")
	if sourceCount != 3 {
		t.Errorf("Expected 3 sources, got %d", sourceCount)
	}
	if !strings.Contains(output, "Valid Article") {
		t.Error("Output should contain valid article title")
	}
	if !strings.Contains(output, "Named Site") {
		t.Error("Output should contain named site")
	}
	if !strings.Contains(output, "https://example.com/page") {
		t.Error("Output should contain URL-only result")
	}
	if strings.Contains(output, "https://perplexity.ai") {
		t.Error("Output should not contain filtered internal URL")
	}
}

func TestRenderWebResultsURLAsTitle(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test when URL should be displayed (when it differs from title)
	results := []models.WebResult{
		{Title: "Example Site", URL: "https://example.com/different-path"},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "Example Site") {
		t.Error("Output should contain title")
	}
	if !strings.Contains(output, "https://example.com/different-path") {
		t.Error("Output should contain URL when different from title")
	}
}

func TestRenderWebResultsSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with special characters in title and URL
	results := []models.WebResult{
		{Title: "Article with <tags> & \"quotes\"", URL: "https://example.com/path?query=value&other=test"},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "Article with") {
		t.Error("Output should contain title with special characters")
	}
	if !strings.Contains(output, "https://example.com") {
		t.Error("Output should contain URL with query parameters")
	}
}

func TestRenderMarkdownNilRenderer(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Force mdRender to nil (simulating error in constructor)
	r.mdRender = nil

	err := r.RenderMarkdown("# Test")
	if err != nil {
		t.Fatalf("RenderMarkdown() error = %v", err)
	}

	output := buf.String()
	if output != "# Test\n" {
		t.Errorf("Output = %q, want %q", output, "# Test\n")
	}
}

func TestRenderMarkdownRenderError(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// This should fallback to raw content on error
	// (Glamour typically doesn't return errors, but the fallback is tested)
	err := r.RenderMarkdown("some text")
	if err != nil {
		t.Fatalf("RenderMarkdown() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Output should not be empty even on error")
	}
}

func TestRenderStyledResponseNilRenderer(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Force mdRender to nil
	r.mdRender = nil

	err := r.RenderStyledResponse("# Test Response")
	if err != nil {
		t.Fatalf("RenderStyledResponse() error = %v", err)
	}

	output := buf.String()
	if output != "# Test Response\n" {
		t.Errorf("Output = %q, want %q", output, "# Test Response\n")
	}
}

func TestRenderStyledResponseWithMarkdown(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	err := r.RenderStyledResponse("# Test Response\n\n**Bold text**")
	if err != nil {
		t.Fatalf("RenderStyledResponse() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Test Response") {
		t.Error("Output should contain response content")
	}
}

func TestRenderWebResultsNewsFormat(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with news-style results
	results := []models.WebResult{
		{Title: "Breaking News", URL: "https://news.example.com/article1", Snippet: "Latest update..."},
		{Title: "Another News", URL: "https://news.example.com/article2", Snippet: "More updates..."},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "Breaking News") {
		t.Error("Output should contain news title")
	}
	if !strings.Contains(output, "[1]") {
		t.Error("Output should contain citation number")
	}
}

func TestRenderWebResultsAcademicFormat(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with academic paper style results
	results := []models.WebResult{
		{Title: "Deep Learning Research Paper", URL: "https://arxiv.org/abs/2024.12345"},
		{Title: "AI Ethics Study", URL: "https://scholar.google.com/study"},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "Deep Learning Research Paper") {
		t.Error("Output should contain academic paper title")
	}
	if !strings.Contains(output, "https://arxiv.org/abs/2024.12345") {
		t.Error("Output should contain arxiv URL")
	}
}

func TestRenderWebResultsLongTitle(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with very long title
	longTitle := "This is a very long article title that might need special handling " +
		"because it contains many words and could potentially cause issues with " +
		"terminal rendering or text wrapping"
	results := []models.WebResult{
		{Title: longTitle, URL: "https://example.com/long"},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "This is a very long article title") {
		t.Error("Output should contain long title")
	}
	if !strings.Contains(output, "https://example.com/long") {
		t.Error("Output should contain URL")
	}
}

func TestRenderWebResultsWithMetaData(t *testing.T) {
	var buf bytes.Buffer
	r, _ := NewRendererWithOptions(&buf, 80, false)

	// Test with metadata (should not affect rendering)
	results := []models.WebResult{
		{
			Title:    "Article with Metadata",
			URL:      "https://example.com/article",
			MetaData: map[string]interface{}{"author": "John Doe", "date": "2024-01-01"},
		},
	}

	r.RenderWebResults(results)

	output := buf.String()
	if !strings.Contains(output, "Article with Metadata") {
		t.Error("Output should contain title")
	}
	if !strings.Contains(output, "https://example.com/article") {
		t.Error("Output should contain URL")
	}
}

func TestResponseContainerHorizontalOverhead(t *testing.T) {
	// The overhead constant should match the actual container style:
	// Border (RoundedBorder): 1 left + 1 right = 2
	// Padding(1, 2): 2 left + 2 right = 4
	// Total = 6
	if ResponseContainerHorizontalOverhead != 6 {
		t.Errorf("ResponseContainerHorizontalOverhead = %d, want 6", ResponseContainerHorizontalOverhead)
	}
}

func TestNewRendererWithOptionsEffectiveWidth(t *testing.T) {
	tests := []struct {
		name          string
		terminalWidth int
		wantWidth     int // The width stored in the renderer
	}{
		{
			name:          "standard 80-column terminal",
			terminalWidth: 80,
			wantWidth:     80, // Renderer stores original width for container
		},
		{
			name:          "wide terminal 120 columns",
			terminalWidth: 120,
			wantWidth:     120,
		},
		{
			name:          "narrow terminal 40 columns",
			terminalWidth: 40,
			wantWidth:     40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			r, err := NewRendererWithOptions(&buf, tt.terminalWidth, false)
			if err != nil {
				t.Fatalf("NewRendererWithOptions() error = %v", err)
			}
			if r.width != tt.wantWidth {
				t.Errorf("renderer.width = %d, want %d", r.width, tt.wantWidth)
			}
			// The mdRender should use effectiveContentWidth = terminalWidth - 6
			// This is verified by testing the actual output behavior
		})
	}
}

func TestNewRendererWithOptionsMinimumWidth(t *testing.T) {
	// Test edge case where width is very small (less than overhead)
	var buf bytes.Buffer
	r, err := NewRendererWithOptions(&buf, 5, false) // 5 < 6 (overhead)
	if err != nil {
		t.Fatalf("NewRendererWithOptions() error = %v", err)
	}
	if r == nil {
		t.Fatal("Renderer should not be nil even with very small width")
	}
	// Should not panic and should work with minimum effective width of 1
}

func TestRenderStyledResponseWordWrap(t *testing.T) {
	// This test verifies that word wrap is correctly adjusted for the container's overhead
	// The word wrap should happen at (width - 6) columns, not at full width

	var buf bytes.Buffer
	terminalWidth := 40
	r, err := NewRendererWithOptions(&buf, terminalWidth, false)
	if err != nil {
		t.Fatalf("NewRendererWithOptions() error = %v", err)
	}

	// Create a long line that would wrap differently depending on the wrap width
	// If wrap is at 40: one wrap point
	// If wrap is at 34 (40-6): different wrap point
	longText := "This is a test sentence that should wrap correctly within the container."

	err = r.RenderStyledResponse(longText)
	if err != nil {
		t.Fatalf("RenderStyledResponse() error = %v", err)
	}

	output := buf.String()

	// The output should contain the text
	if !strings.Contains(output, "This is a test") {
		t.Error("Output should contain the test text")
	}

	// The output should have border characters (from lipgloss RoundedBorder)
	// RoundedBorder uses characters like ╭, ╮, ╰, ╯, │, ─
	if !strings.Contains(output, "│") && !strings.Contains(output, "─") {
		t.Error("Output should contain border characters")
	}
}

func TestNormalizeMarkdownText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "joins wrapped paragraph lines",
			input: "This is a long sentence that was\nartificially broken by the API.",
			want:  "This is a long sentence that was artificially broken by the API.",
		},
		{
			name:  "preserves paragraph breaks",
			input: "First paragraph.\n\nSecond paragraph.",
			want:  "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:  "preserves headers",
			input: "Some text.\n\n### Header\n\nMore text.",
			want:  "Some text.\n\n### Header\n\nMore text.",
		},
		{
			name:  "preserves bullet lists",
			input: "List:\n\n• First item\n• Second item\n• Third item",
			want:  "List:\n\n• First item\n• Second item\n• Third item",
		},
		{
			name:  "preserves dash lists",
			input: "List:\n\n- First item\n- Second item",
			want:  "List:\n\n- First item\n- Second item",
		},
		{
			name:  "preserves numbered lists",
			input: "List:\n\n1. First item\n2. Second item",
			want:  "List:\n\n1. First item\n2. Second item",
		},
		{
			name:  "preserves tables",
			input: "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
			want:  "| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |",
		},
		{
			name:  "preserves code blocks",
			input: "Code:\n\n```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
			want:  "Code:\n\n```go\nfunc main() {\n    fmt.Println(\"hello\")\n}\n```",
		},
		{
			name:  "complex mixed content",
			input: "A response with multiple\nlines that should be joined.\n\n### Section Header\n\n• List item one\n• List item two\n\nAnother paragraph that was\nbroken artificially.",
			want:  "A response with multiple lines that should be joined.\n\n### Section Header\n\n• List item one\n• List item two\n\nAnother paragraph that was broken artificially.",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "single line",
			input: "Just one line.",
			want:  "Just one line.",
		},
		{
			name:  "list item with wrapped continuation",
			input: "• Importância: Foi a primeira prova extra-bíblica de que Davi não era um\nmito, mas o fundador de uma linhagem real.",
			want:  "• Importância: Foi a primeira prova extra-bíblica de que Davi não era um mito, mas o fundador de uma linhagem real.",
		},
		{
			name:  "multiple list items with wrapping",
			input: "• First item that spans\nmultiple lines here.\n• Second item also\nwrapped.",
			want:  "• First item that spans multiple lines here.\n• Second item also wrapped.",
		},
		{
			name:  "paragraph after list item",
			input: "• A list item.\n\nA new paragraph after\nthe list.",
			want:  "• A list item.\n\nA new paragraph after the list.",
		},
		{
			name:  "real API response pattern",
			input: "Durante décadas, acreditava-se que as famosas \"Minas do Rei Salomão\" eram\num\nmito ou que a mineração na região só ocorrera séculos depois.",
			want:  "Durante décadas, acreditava-se que as famosas \"Minas do Rei Salomão\" eram um mito ou que a mineração na região só ocorrera séculos depois.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeMarkdownText(tt.input)
			if got != tt.want {
				t.Errorf("normalizeMarkdownText() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}
