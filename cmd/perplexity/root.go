package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/diogo/perplexity-go/internal/auth"
	"github.com/diogo/perplexity-go/internal/config"
	"github.com/diogo/perplexity-go/internal/history"
	"github.com/diogo/perplexity-go/internal/ui"
	"github.com/diogo/perplexity-go/pkg/client"
	"github.com/diogo/perplexity-go/pkg/models"
	"github.com/spf13/cobra"
)

var (
	// Flags
	flagModel      string
	flagMode       string
	flagSources    string
	flagLanguage   string
	flagStream     bool
	flagNoStream   bool
	flagIncognito  bool
	flagOutputFile string
	flagCookieFile string
	flagVerbose    bool

	// Global config
	cfg     *config.Config
	cfgMgr  *config.Manager
	render  *ui.Renderer
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "perplexity [query]",
	Short: "Perplexity AI CLI - Search with AI",
	Long: `Perplexity CLI is a command-line interface for Perplexity AI.

It allows you to perform AI-powered searches directly from your terminal
with support for multiple models, streaming output, and file attachments.

Examples:
  perplexity "What is the capital of France?"
  perplexity "Explain quantum computing" --model gpt5 --mode pro
  perplexity "Latest news on AI" --sources web,scholar --stream`,
	Args: cobra.ArbitraryArgs,
	RunE: runQuery,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Query flags
	rootCmd.Flags().StringVarP(&flagModel, "model", "m", "", "AI model to use (pplx_pro, gpt5, claude45sonnet, etc.)")
	rootCmd.Flags().StringVar(&flagMode, "mode", "", "Search mode (fast, pro, reasoning, deep-research, default)")
	rootCmd.Flags().StringVarP(&flagSources, "sources", "s", "", "Search sources (web,scholar,social)")
	rootCmd.Flags().StringVarP(&flagLanguage, "language", "l", "", "Response language (e.g., en-US, pt-BR)")
	rootCmd.Flags().BoolVar(&flagStream, "stream", false, "Enable streaming output")
	rootCmd.Flags().BoolVar(&flagNoStream, "no-stream", false, "Disable streaming output")
	rootCmd.Flags().BoolVarP(&flagIncognito, "incognito", "i", false, "Don't save to history")
	rootCmd.Flags().StringVarP(&flagOutputFile, "output", "o", "", "Save response to file")
	rootCmd.Flags().StringVarP(&flagCookieFile, "cookies", "c", "", "Path to cookies.json file")
	rootCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output")

	// Add subcommands
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(cookiesCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	var err error

	// Initialize config manager
	cfgMgr, err = config.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Load configuration
	cfg, err = cfgMgr.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize renderer
	render, err = ui.NewRenderer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing renderer: %v\n", err)
		os.Exit(1)
	}
}

func runQuery(cmd *cobra.Command, args []string) error {
	// Check if query provided
	if len(args) == 0 {
		return cmd.Help()
	}

	query := strings.Join(args, " ")

	// Determine cookie file
	cookieFile := cfg.CookieFile
	if flagCookieFile != "" {
		cookieFile = flagCookieFile
	}

	// Check if cookies exist
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		render.RenderError(fmt.Errorf("cookies file not found: %s", cookieFile))
		render.RenderInfo("Run 'perplexity cookies import <file>' to import cookies from browser")
		return fmt.Errorf("no cookies found")
	}

	// Load cookies
	cookies, err := auth.LoadCookiesFromFile(cookieFile)
	if err != nil {
		render.RenderError(fmt.Errorf("failed to load cookies: %v", err))
		return err
	}

	// Create client
	cli, err := client.NewWithCookies(cookies)
	if err != nil {
		render.RenderError(fmt.Errorf("failed to create client: %v", err))
		return err
	}
	defer cli.Close()

	// Build search options
	opts := buildSearchOptions(query)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Determine if streaming
	streaming := cfg.Streaming
	if flagStream {
		streaming = true
	}
	if flagNoStream {
		streaming = false
	}
	opts.Stream = streaming

	if flagVerbose {
		render.RenderInfo(fmt.Sprintf("Query: %s", query))
		render.RenderInfo(fmt.Sprintf("Mode: %s, Model: %s", opts.Mode, opts.Model))
		render.RenderInfo(fmt.Sprintf("Streaming: %v", streaming))
		render.NewLine()
	}

	var responseText string

	if streaming {
		// Streaming mode
		ch, err := cli.SearchStream(ctx, opts)
		if err != nil {
			render.RenderError(err)
			return err
		}

		var fullResponse strings.Builder
		var allWebResults []models.WebResult
		for chunk := range ch {
			if chunk.Error != nil {
				if chunk.Error == context.Canceled {
					render.NewLine()
					render.RenderWarning("Search cancelled")
					break
				}
				render.RenderError(chunk.Error)
				return chunk.Error
			}

			// For new step-based format, only render FINAL step
			if chunk.StepType == "FINAL" && chunk.Text != "" {
				// Render as markdown instead of raw text
				if err := render.RenderMarkdown(chunk.Text); err != nil {
					render.RenderStreamChunk(chunk)
				}
				fullResponse.WriteString(chunk.Text)
				allWebResults = append(allWebResults, chunk.WebResults...)
			} else if chunk.StepType == "" {
				// Legacy format - render as stream
				render.RenderStreamChunk(chunk)
				if chunk.Delta != "" {
					fullResponse.WriteString(chunk.Delta)
				} else if chunk.Text != "" {
					fullResponse.WriteString(chunk.Text)
				}
			}
		}
		render.NewLine()

		// Render web results if any
		if len(allWebResults) > 0 {
			render.RenderWebResults(allWebResults)
		}

		responseText = fullResponse.String()
	} else {
		// Non-streaming mode with spinner
		done := make(chan struct{})
		go func() {
			frame := 0
			for {
				select {
				case <-done:
					render.ClearLine()
					return
				case <-time.After(100 * time.Millisecond):
					render.RenderSpinner(frame)
					frame++
				}
			}
		}()

		resp, err := cli.Search(ctx, opts)
		close(done)

		if err != nil {
			if err == context.Canceled {
				render.RenderWarning("Search cancelled")
				return nil
			}
			render.RenderError(err)
			return err
		}

		if err := render.RenderResponse(resp); err != nil {
			render.RenderError(err)
			return err
		}
		responseText = resp.Text
	}

	// Save to output file if specified
	if flagOutputFile != "" {
		if err := os.WriteFile(flagOutputFile, []byte(responseText), 0644); err != nil {
			render.RenderError(fmt.Errorf("failed to save output: %v", err))
		} else {
			render.RenderSuccess(fmt.Sprintf("Saved to %s", flagOutputFile))
		}
	}

	// Save to history if not incognito
	if !flagIncognito && !cfg.Incognito {
		hw, err := history.NewWriter(cfg.HistoryFile)
		if err == nil {
			hw.Append(models.HistoryEntry{
				Query:    query,
				Mode:     string(opts.Mode),
				Model:    string(opts.Model),
				Response: truncateResponse(responseText, 500),
			})
		}
	}

	return nil
}

func buildSearchOptions(query string) models.SearchOptions {
	opts := models.DefaultSearchOptions(query)

	// Apply config defaults
	opts.Model = cfg.DefaultModel
	opts.Mode = cfg.DefaultMode
	opts.Language = cfg.DefaultLanguage
	opts.Sources = cfg.DefaultSources
	opts.Incognito = cfg.Incognito

	// Override with flags
	if flagModel != "" {
		opts.Model = models.Model(flagModel)
	}
	if flagMode != "" {
		opts.Mode = models.Mode(flagMode)
	}
	if flagLanguage != "" {
		opts.Language = flagLanguage
	}
	if flagSources != "" {
		sources := strings.Split(flagSources, ",")
		opts.Sources = make([]models.Source, 0, len(sources))
		for _, s := range sources {
			opts.Sources = append(opts.Sources, models.Source(strings.TrimSpace(s)))
		}
	}
	if flagIncognito {
		opts.Incognito = true
	}

	return opts
}

func truncateResponse(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
