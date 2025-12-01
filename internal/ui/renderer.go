// Package ui handles terminal output and formatting.
package ui

import (
	"fmt"
	"io"
	"os"
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
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
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

// RenderResponse renders a complete search response.
func (r *Renderer) RenderResponse(resp *models.SearchResponse) error {
	// First check for new format with direct Text and WebResults
	if resp.Text != "" {
		if err := r.RenderMarkdown(resp.Text); err != nil {
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
			if err := r.RenderMarkdown(block.MarkdownBlock.Answer); err != nil {
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
	fmt.Fprintf(r.out, "\r%s ", SpinnerChars[idx])
}

// ClearLine clears the current line.
func (r *Renderer) ClearLine() {
	fmt.Fprint(r.out, "\r\033[K")
}

// NewLine prints a newline.
func (r *Renderer) NewLine() {
	fmt.Fprintln(r.out)
}
