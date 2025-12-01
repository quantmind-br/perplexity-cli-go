// Package ui handles terminal output and formatting.
package ui

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/diogo/perplexity-go/pkg/models"
)

// Renderer handles terminal output formatting.
type Renderer struct {
	out       io.Writer
	mdRender  *glamour.TermRenderer
	width     int
	useColors bool
}

// Styles for different output elements.
var (
	// Custom Warm Colors for Theme
	WarmColorPrimary = lipgloss.Color("#F9C74F") // Light orange/yellow
	WarmColorDark    = lipgloss.Color("#E36414") // Darker orange
	WarmColorBg      = lipgloss.Color("#1E1E1E") // Dark background for code

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(WarmColorPrimary).
			MarginBottom(1)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("82"))

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214"))

	SpinnerStyle = lipgloss.NewStyle().
		Foreground(WarmColorPrimary)

	// Style for the main response container (border, padding)
	ResponseContainerStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(WarmColorPrimary).
		Padding(1, 2).
		Margin(1, 0, 0, 0)

	CitationStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Underline(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	SpinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// NewRenderer creates a new output renderer.
func NewRenderer() (*Renderer, error) {
	return NewRendererWithOptions(os.Stdout, 80, true)
}

// NewRendererWithOptions creates a renderer with custom options.
func NewRendererWithOptions(out io.Writer, width int, useColors bool) (*Renderer, error) {
	style := "dark"
	if !useColors {
		style = "notty"
	}

	mdRender, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithStylePath("dracula"), // A good warm-ish, dark theme for code highlighting
		glamour.WithWordWrap(width),
		glamour.WithStylePath(style),
	)
	if err != nil {
		// Fallback to basic renderer
		mdRender, _ = glamour.NewTermRenderer(
			glamour.WithWordWrap(width),
		)
	}

	return &Renderer{
		out:       out,
		mdRender:  mdRender,
		width:     width,
		useColors: useColors,
	}, nil
}

// RenderMarkdown renders markdown content.
func (r *Renderer) RenderMarkdown(content string) error {
	if r.mdRender == nil {
		// Fallback: print raw content
		fmt.Fprintln(r.out, content)
		return nil
	}

	rendered, err := r.mdRender.Render(content)
	if err != nil {
		// Fallback to raw content on error
		fmt.Fprintln(r.out, content)
		return nil
	}

	fmt.Fprint(r.out, rendered)
	return nil
}

// RenderStyledResponse renders content inside the stylized container with Markdown formatting.
func (r *Renderer) RenderStyledResponse(content string) error {
	if r.mdRender == nil {
		fmt.Fprintln(r.out, content)
		return nil
	}

	// Normalize text to remove artificial line breaks from API
	normalizedContent := normalizeMarkdownText(content)

	// 1. Render Markdown content internally using glamour
	rendered, err := r.mdRender.Render(normalizedContent)
	if err != nil {
		return r.RenderMarkdown(normalizedContent) // Fallback to basic markdown render
	}
	// 2. Wrap the rendered content in the container style
	styledContent := ResponseContainerStyle.
		Width(r.width).
		Foreground(lipgloss.Color("252")). // Light gray text for readability inside the box
		Render(rendered)
	fmt.Fprintln(r.out, styledContent)
	return nil
}

// RenderResponse renders a complete search response.
func (r *Renderer) RenderResponse(resp *models.SearchResponse) error {
	// First check for new format with direct Text and WebResults
	if resp.Text != "" {
		if err := r.RenderStyledResponse(resp.Text); err != nil {
			return err
		}

		// Render web results from new format
		if len(resp.WebResults) > 0 {
			r.RenderWebResults(resp.WebResults)
		}
		return nil
	}

	// Fallback: Find and render markdown blocks (legacy format)
	for _, block := range resp.Blocks {
		if block.MarkdownBlock != nil {
			if err := r.RenderStyledResponse(block.MarkdownBlock.Answer); err != nil {
				return err
			}

			// Render citations if present
			if len(block.MarkdownBlock.Citations) > 0 {
				r.RenderCitations(block.MarkdownBlock.Citations)
			}
		}
	}

	return nil
}

// RenderCitations renders source citations.
func (r *Renderer) RenderCitations(citations []models.Citation) {
	if len(citations) == 0 {
		return
	}

	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, DimStyle.Render("Sources:"))

	for i, cite := range citations {
		title := cite.Title
		if title == "" {
			title = cite.URL
		}

		num := fmt.Sprintf("[%d]", i+1)
		fmt.Fprintf(r.out, "%s %s\n", DimStyle.Render(num), CitationStyle.Render(title))
		if cite.URL != "" && cite.URL != title {
			fmt.Fprintf(r.out, "    %s\n", DimStyle.Render(cite.URL))
		}
	}
}

// RenderWebResults renders web search results from new API format.
func (r *Renderer) RenderWebResults(results []models.WebResult) {
	if len(results) == 0 {
		return
	}

	// Filter out internal calculator results
	var filteredResults []models.WebResult
	for _, wr := range results {
		if wr.URL != "https://perplexity.ai" && wr.URL != "" {
			filteredResults = append(filteredResults, wr)
		}
	}

	if len(filteredResults) == 0 {
		return
	}

	fmt.Fprintln(r.out)
	fmt.Fprintln(r.out, DimStyle.Render("Sources:"))

	for i, wr := range filteredResults {
		title := wr.Title
		if title == "" {
			title = wr.Name
		}
		if title == "" {
			title = wr.URL
		}

		num := fmt.Sprintf("[%d]", i+1)
		fmt.Fprintf(r.out, "%s %s\n", DimStyle.Render(num), CitationStyle.Render(title))
		if wr.URL != "" && wr.URL != title {
			fmt.Fprintf(r.out, "    %s\n", DimStyle.Render(wr.URL))
		}
	}
}

// RenderStreamChunk renders a streaming chunk (token-by-token).
func (r *Renderer) RenderStreamChunk(chunk models.StreamChunk) {
	if chunk.Delta != "" {
		fmt.Fprint(r.out, chunk.Delta)
	} else if chunk.Text != "" {
		fmt.Fprint(r.out, chunk.Text)
	}
}

// RenderError renders an error message.
func (r *Renderer) RenderError(err error) {
	if r.useColors {
		fmt.Fprintln(r.out, ErrorStyle.Render("Error: "+err.Error()))
	} else {
		fmt.Fprintln(r.out, "Error: "+err.Error())
	}
}

// RenderSuccess renders a success message.
func (r *Renderer) RenderSuccess(msg string) {
	if r.useColors {
		fmt.Fprintln(r.out, SuccessStyle.Render(msg))
	} else {
		fmt.Fprintln(r.out, msg)
	}
}

// RenderWarning renders a warning message.
func (r *Renderer) RenderWarning(msg string) {
	if r.useColors {
		fmt.Fprintln(r.out, WarningStyle.Render("Warning: "+msg))
	} else {
		fmt.Fprintln(r.out, "Warning: "+msg)
	}
}

// RenderInfo renders an info message.
func (r *Renderer) RenderInfo(msg string) {
	if r.useColors {
		fmt.Fprintln(r.out, InfoStyle.Render(msg))
	} else {
		fmt.Fprintln(r.out, msg)
	}
}

// RenderTitle renders a title.
func (r *Renderer) RenderTitle(title string) {
	if r.useColors {
		fmt.Fprintln(r.out, TitleStyle.Render(title))
	} else {
		fmt.Fprintln(r.out, strings.ToUpper(title))
		fmt.Fprintln(r.out, strings.Repeat("=", len(title)))
	}
}

// RenderSpinner renders a spinner character.
func (r *Renderer) RenderSpinner(frame int) {
	idx := frame % len(SpinnerChars)
	fmt.Fprintf(r.out, "\r%s ", SpinnerStyle.Render(SpinnerChars[idx]))
}

// ClearLine clears the current line.
func (r *Renderer) ClearLine() {
	fmt.Fprint(r.out, "\r\033[K")
}

// NewLine prints a newline.
func (r *Renderer) NewLine() {
	fmt.Fprintln(r.out)
}

// normalizeMarkdownText removes artificial line breaks from API responses
// while preserving markdown structure (headers, lists, tables, code blocks, paragraphs).
func normalizeMarkdownText(text string) string {
	// Normalize CRLF to LF
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Patterns for structural elements
	headerPattern := regexp.MustCompile(`^\s*#{1,6}\s`)
	listPattern := regexp.MustCompile(`^\s*([-*•]|\d+\.)\s`)
	tablePattern := regexp.MustCompile(`^\s*\|`)
	codeBlockPattern := regexp.MustCompile("^\\s*```")

	lines := strings.Split(text, "\n")
	var result []string
	var currentBlock strings.Builder
	inCodeBlock := false

	flushBlock := func() {
		if currentBlock.Len() > 0 {
			result = append(result, strings.TrimSpace(currentBlock.String()))
			currentBlock.Reset()
		}
	}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track code blocks
		if codeBlockPattern.MatchString(line) {
			flushBlock()
			inCodeBlock = !inCodeBlock
			result = append(result, line)
			continue
		}

		// Inside code block - preserve exactly
		if inCodeBlock {
			result = append(result, line)
			continue
		}

		// Empty line = end of current block
		if trimmedLine == "" {
			flushBlock()
			result = append(result, "")
			continue
		}

		// Headers - always standalone
		if headerPattern.MatchString(line) {
			flushBlock()
			result = append(result, line)
			continue
		}

		// Tables - always standalone (each line)
		if tablePattern.MatchString(line) {
			flushBlock()
			result = append(result, line)
			continue
		}

		// List item start
		if listPattern.MatchString(line) {
			flushBlock()
			currentBlock.WriteString(trimmedLine)
			continue
		}

		// Continuation of list item or paragraph
		if currentBlock.Len() > 0 {
			currentBlock.WriteString(" ")
			currentBlock.WriteString(trimmedLine)
		} else {
			// Start of a new paragraph
			currentBlock.WriteString(trimmedLine)
		}
	}

	// Flush remaining block
	flushBlock()

	return strings.Join(result, "\n")
}
