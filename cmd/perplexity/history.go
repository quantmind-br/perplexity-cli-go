package main

import (
	"fmt"
	"strconv"

	"github.com/diogo/perplexity-go/internal/history"
	"github.com/spf13/cobra"
)

var (
	historyCount int
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View query history",
	Long:  `View and manage your query history.`,
	RunE:  runHistoryList,
}

var historyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent queries",
	RunE:  runHistoryList,
}

var historySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := history.NewReader(cfg.HistoryFile)
		entries, err := reader.Search(args[0])
		if err != nil {
			return fmt.Errorf("failed to search history: %v", err)
		}

		if len(entries) == 0 {
			render.RenderInfo("No matching entries found")
			return nil
		}

		render.RenderTitle(fmt.Sprintf("Search Results: %d matches", len(entries)))
		for i, entry := range entries {
			fmt.Printf("[%d] %s\n", i+1, entry.Timestamp.Format("2006-01-02 15:04"))
			fmt.Printf("    Query: %s\n", entry.Query)
			fmt.Printf("    Mode: %s, Model: %s\n", entry.Mode, entry.Model)
			fmt.Println()
		}

		return nil
	},
}

var historyShowCmd = &cobra.Command{
	Use:   "show <index>",
	Short: "Show details of a history entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idx, err := strconv.Atoi(args[0])
		if err != nil || idx < 1 {
			return fmt.Errorf("invalid index: %s", args[0])
		}

		reader := history.NewReader(cfg.HistoryFile)
		entries, err := reader.ReadAll()
		if err != nil {
			return fmt.Errorf("failed to read history: %v", err)
		}

		if idx > len(entries) {
			return fmt.Errorf("index out of range: %d (max: %d)", idx, len(entries))
		}

		entry := entries[idx-1]
		render.RenderTitle("History Entry")
		fmt.Printf("Timestamp: %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("Query:     %s\n", entry.Query)
		fmt.Printf("Mode:      %s\n", entry.Mode)
		fmt.Printf("Model:     %s\n", entry.Model)
		if entry.Response != "" {
			fmt.Println("\nResponse:")
			render.RenderMarkdown(entry.Response)
		}

		return nil
	},
}

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all history",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := history.NewReader(cfg.HistoryFile)
		if err := reader.Clear(); err != nil {
			return fmt.Errorf("failed to clear history: %v", err)
		}
		render.RenderSuccess("History cleared")
		return nil
	},
}

func runHistoryList(cmd *cobra.Command, args []string) error {
	reader := history.NewReader(cfg.HistoryFile)

	var entries []any
	var err error

	if historyCount > 0 {
		entries, err = readLastN(reader, historyCount)
	} else {
		entries, err = readLastN(reader, 20)
	}

	if err != nil {
		return fmt.Errorf("failed to read history: %v", err)
	}

	if len(entries) == 0 {
		render.RenderInfo("No history entries")
		return nil
	}

	render.RenderTitle("Recent Queries")
	for i, e := range entries {
		entry := e.(historyEntry)
		fmt.Printf("[%d] %s\n", i+1, entry.Timestamp)
		fmt.Printf("    %s\n", entry.Query)
		if entry.Mode != "" {
			fmt.Printf("    Mode: %s", entry.Mode)
			if entry.Model != "" {
				fmt.Printf(", Model: %s", entry.Model)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	return nil
}

type historyEntry struct {
	Timestamp string
	Query     string
	Mode      string
	Model     string
}

func readLastN(reader *history.Reader, n int) ([]any, error) {
	entries, err := reader.ReadLast(n)
	if err != nil {
		return nil, err
	}

	result := make([]any, len(entries))
	for i, e := range entries {
		result[i] = historyEntry{
			Timestamp: e.Timestamp.Format("2006-01-02 15:04"),
			Query:     e.Query,
			Mode:      e.Mode,
			Model:     e.Model,
		}
	}
	return result, nil
}

func init() {
	historyCmd.AddCommand(historyListCmd)
	historyCmd.AddCommand(historySearchCmd)
	historyCmd.AddCommand(historyShowCmd)
	historyCmd.AddCommand(historyClearCmd)

	historyCmd.Flags().IntVarP(&historyCount, "count", "n", 20, "Number of entries to show")
	historyListCmd.Flags().IntVarP(&historyCount, "count", "n", 20, "Number of entries to show")
}
