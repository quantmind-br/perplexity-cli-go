╭───────────────────────────────────────────────────────────────────────────────────╮
│                                                                                   │
│   diff --git a/cmd/perplexity/history.go b/cmd/perplexity/history.go              │
│   index 4.2KB..4.2KB 100644                                                       │
│   --- a/cmd/perplexity/history.go                                                 │
│   +++ b/cmd/perplexity/history.go                                                 │
│   @@ -85,7 +85,7 @@                                                               │
│   fmt.Printf("Model:     %s\n", entry.Model)                                      │
│   if entry.Response != "" {                                                       │
│   fmt.Println("\nResponse:")                                                      │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           render.RenderMarkdown(entry.Response)                                   │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           render.RenderStyledResponse(entry.Response)                             │
│       }                                                                           │
│                                                                                   │
│       return nil                                                                  │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   diff --git a/cmd/perplexity/root.go b/cmd/perplexity/root.go                    │
│   index 8.4KB..8.3KB 100644                                                       │
│   --- a/cmd/perplexity/root.go                                                    │
│   +++ b/cmd/perplexity/root.go                                                    │
│   @@ -207,29 +207,24 @@                                                           │
│   // Streaming mode                                                               │
│   ch, err := cli.SearchStream(ctx, opts)                                          │
│   if err != nil {                                                                 │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           render.RenderError(err)                                                 │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           render.RenderError(err) // Render error from stream initiation          │
│           return err                                                              │
│       }                                                                           │
│                                                                                   │
│       var fullResponse strings.Builder                                            │
│       var allWebResults []models.WebResult                                        │
│       for chunk := range ch {                                                     │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if chunk.Error != nil {                                                 │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               if chunk.Error == context.Canceled {                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   render.NewLine()                                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   render.RenderWarning("Search cancelled")                        │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   break                                                           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               }                                                                   │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if chunk.Error != nil { // Handle all chunk errors                      │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               if chunk.Error == context.Canceled {                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   render.NewLine()                                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   render.RenderWarning("Search cancelled")                        │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   break // Exit loop on cancel                                    │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               }                                                                   │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               // Report other errors                                              │
│               render.RenderError(chunk.Error)                                     │
│               return chunk.Error                                                  │
│           }                                                                       │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           // For new step-based format, only render FINAL step                    │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if chunk.StepType == "FINAL" && chunk.Text != "" {                      │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               // Render as markdown instead of raw text                           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               if err := render.RenderMarkdown(chunk.Text); err != nil {           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   render.RenderStreamChunk(chunk)                                 │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               }                                                                   │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               fullResponse.WriteString(chunk.Text)                                │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           // Accumulate all output text/deltas, rendering the raw stream          │
│   as it arrives.                                                                  │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if chunk.Delta != "" {                                                  │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               render.RenderStreamChunk(chunk)                                     │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               fullResponse.WriteString(chunk.Delta)                               │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           } else if chunk.Text != "" {                                            │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               render.RenderStreamChunk(chunk)                                     │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               fullResponse.WriteString(chunk.Text)                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           }                                                                       │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           // Collect web results from FINAL steps (as these contain the           │
│   full set of results)                                                            │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if chunk.StepType == "FINAL" && len(chunk.WebResults) > 0 {             │
│               allWebResults = append(allWebResults, chunk.WebResults...)          │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           } else if chunk.StepType == "" {                                        │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               // Legacy format - render as stream                                 │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               render.RenderStreamChunk(chunk)                                     │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               if chunk.Delta != "" {                                              │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   fullResponse.WriteString(chunk.Delta)                           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               } else if chunk.Text != "" {                                        │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│                   fullResponse.WriteString(chunk.Text)                            │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               }                                                                   │
│           }                                                                       │
│       }                                                                           │
│       render.NewLine()                                                            │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       // Post-stream rendering of the full accumulated response (with             │
│   new styling/markdown)                                                           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       if fullResponse.Len() > 0 {                                                 │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if err := render.RenderStyledResponse(fullResponse.String());           │
│   err != nil {                                                                    │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               // If final styled rendering fails, the raw stream output is        │
│   still there.                                                                    │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               render.RenderError(fmt.Errorf("failed to render final               │
│   response: %w", err))                                                            │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           }                                                                       │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       }                                                                           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       // Render web results if any                                                │
│       if len(allWebResults) > 0 {                                                 │
│           render.RenderWebResults(allWebResults)                                  │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   diff --git a/internal/ui/renderer.go b/internal/ui/renderer.go                  │
│   index 6.4KB..7.0KB 100644                                                       │
│   --- a/internal/ui/renderer.go                                                   │
│   +++ b/internal/ui/renderer.go                                                   │
│   @@ -14,14 +14,28 @@                                                             │
│   // Styles for different output elements.                                        │
│   var (                                                                           │
│   TitleStyle = lipgloss.NewStyle().                                               │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       // Custom Warm Colors for Theme                                             │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       WarmColorPrimary = lipgloss.Color("#F9C74F") // Light                       │
│   orange/yellow                                                                   │
│     ----------                                                                    │
│                                                                                   │
│   * WarmColorDark    = lipgloss.Color("#E36414") // Darker orange                 │
│   * WarmColorBg      = lipgloss.Color("#1E1E1E") // Dark background               │
│   for code                                                                        │
│   *                                                                               │
│   * TitleStyle = lipgloss.NewStyle().                                             │
│   Bold(true).                                                                     │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           Foreground(lipgloss.Color("99")).                                       │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           Foreground(WarmColorPrimary).                                           │
│           MarginBottom(1)                                                         │
│     ----------                                                                    │
│   InfoStyle = lipgloss.NewStyle().                                                │
│   Foreground(lipgloss.Color("241"))ErrorStyle = lipgloss.NewStyle().              │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           Bold(true).                                                             │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           Foreground(lipgloss.Color("196"))                                       │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│   * SuccessStyle = lipgloss.NewStyle().                                           │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           Bold(true).                                                             │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           Foreground(lipgloss.Color("82"))                                        │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│   * WarningStyle = lipgloss.NewStyle().                                           │
│   Bold(true).                                                                     │
│   Foreground(lipgloss.Color("214"))                                               │
│                                                                                   │
│   @@ -35,9 +49,17 @@                                                              │
│   DimStyle = lipgloss.NewStyle().                                                 │
│   Foreground(lipgloss.Color("245"))                                               │
│                                                                                   │
│   * SpinnerStyle = lipgloss.NewStyle().                                           │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       Foreground(WarmColorPrimary)                                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│   * // Style for the main response container (border, padding)                    │
│   * ResponseContainerStyle = lipgloss.NewStyle().                                 │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       Border(lipgloss.RoundedBorder()).                                           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       BorderForeground(WarmColorPrimary).                                         │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       Padding(1, 2).                                                              │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       Margin(1, 0, 0, 0)                                                          │
│     ----------                                                                    │
│                                                                                   │
│   * SpinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧",               │
│   "⠇", "⠏"}                                                                       │
│   )                                                                               │
│                                                                                   │
│   @@ -53,6 +75,14 @@                                                              │
│                                                                                   │
│                                                                                   │
│     ----------                                                                    │
│     mdRender, err := glamour.NewTermRenderer(                                     │
│         glamour.WithAutoStyle(),                                                  │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       glamour.WithStylePath("dracula"), // A good warm-ish, dark theme            │
│   for code highlighting                                                           │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       glamour.WithStyles(glamour.StyleConfig{                                     │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           CodeBlock: lipgloss.NewStyle().                                         │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               Background(WarmColorBg). // Dark background for code blocks         │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               Border(lipgloss.NormalBorder()).                                    │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               BorderForeground(lipgloss.Color("245")).                            │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               Padding(1, 1).                                                      │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│               Get,                                                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       }),                                                                         │
│       glamour.WithWordWrap(width),                                                │
│       glamour.WithStylePath(style),                                               │
│     ----------                                                                    │
│   )                                                                               │
│   @@ -75,6 +105,26 @@                                                             │
│   return nil                                                                      │
│   }                                                                               │
│                                                                                   │
│   +// RenderStyledResponse renders content inside the stylized                    │
│   container with Markdown formatting.                                             │
│   +func (r *Renderer) RenderStyledResponse(content string) error {                │
│                                                                                   │
│   * if r.mdRender == nil {                                                        │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       fmt.Fprintln(r.out, content)                                                │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       return nil                                                                  │
│     ----------                                                                    │
│                                                                                   │
│   * }                                                                             │
│   *                                                                               │
│   * // 1. Render Markdown content internally using glamour                        │
│   * rendered, err := r.mdRender.Render(content)                                   │
│   * if err != nil {                                                               │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       return r.RenderMarkdown(content) // Fallback to basic markdown              │
│   render                                                                          │
│     ----------                                                                    │
│                                                                                   │
│   * }                                                                             │
│   *                                                                               │
│   * // 2. Wrap the rendered content in the container style                        │
│   * styledContent := ResponseContainerStyle.                                      │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       Width(r.width).                                                             │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       Foreground(lipgloss.Color("252")). // Light gray text for                   │
│   readability inside the box                                                      │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       Render(rendered)                                                            │
│     ----------                                                                    │
│                                                                                   │
│   *                                                                               │
│   * fmt.Fprintln(r.out, styledContent)                                            │
│   * return nil                                                                    │
│   +}                                                                              │
│   *                                                                               │
│                                                                                   │
│   // RenderResponse renders a complete search response.                           │
│   func (r *Renderer) RenderResponse(resp *models.SearchResponse) error            │
│   {                                                                               │
│   // First check for new format with direct Text and WebResults                   │
│   if resp.Text != "" {                                                            │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       if err := r.RenderMarkdown(resp.Text); err != nil {                         │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│       if err := r.RenderStyledResponse(resp.Text); err != nil {                   │
│           return err                                                              │
│       }                                                                           │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   @@ -87,7 +137,7 @@                                                              │
│   // Fallback: Find and render markdown blocks (legacy format)                    │
│   for _, block := range resp.Blocks {                                             │
│   if block.MarkdownBlock != nil {                                                 │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if err := r.RenderMarkdown(block.MarkdownBlock.Answer); err !=          │
│   nil {                                                                           │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   *                                                                               │
│                                                                                   │
│     ----------                                                                    │
│           if err := r.RenderStyledResponse(block.MarkdownBlock.Answer);           │
│   err != nil {                                                                    │
│               return err                                                          │
│           }                                                                       │
│     ----------                                                                    │
│                                                                                   │
│                                                                                   │
│   @@ -198,7 +248,7 @@                                                             │
│   // RenderSpinner renders a spinner character.                                   │
│   func (r *Renderer) RenderSpinner(frame int) {                                   │
│   idx := frame % len(SpinnerChars)                                                │
│                                                                                   │
│   * fmt.Fprintf(r.out, "\r%s ", SpinnerChars[idx])                                │
│                                                                                   │
│   * fmt.Fprintf(r.out, "\r%s ", SpinnerStyle.                                     │
│   Render(SpinnerChars[idx]))                                                      │
│   }                                                                               │
│                                                                                   │
│   // ClearLine clears the current line.                                           │
╰───────────────────────────────────────────────────────────────────────────────────╯