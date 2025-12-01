package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/diogo/perplexity-go/pkg/models"
	"github.com/google/uuid"
)

// buildSearchPayload creates the JSON payload for a search request.
func (c *Client) buildSearchPayload(opts models.SearchOptions) ([]byte, error) {
	c.applyDefaults(&opts)

	// Determine effective mode based on model
	effectiveMode := string(opts.Mode)
	modelPref := string(opts.Model)
	isProReasoning := false

	// Map mode to API values (unless overridden by special model)
	switch opts.Mode {
	case models.ModeFast:
		effectiveMode = "concise"
		modelPref = "turbo"
	case models.ModePro, models.ModeDefault:
		effectiveMode = "copilot"
	case models.ModeReasoning:
		effectiveMode = "copilot"
		isProReasoning = true
	case models.ModeDeepResearch:
		effectiveMode = "copilot"
		modelPref = "pplx_alpha"
	}

	// Handle special gpt5_thinking model (forces fast mode regardless of mode setting)
	if opts.Model == models.ModelGPT5Thinking {
		effectiveMode = "concise"
		modelPref = "turbo"
	}

	// Convert sources to strings
	sources := make([]string, len(opts.Sources))
	for i, s := range opts.Sources {
		sources[i] = string(s)
	}

	// Generate UUIDs
	frontendUUID := uuid.New().String()

	// Build request payload
	req := models.SearchRequest{
		Version:            "2.18",
		Source:             "default",
		Language:           opts.Language,
		Timezone:           "America/New_York",
		SearchFocus:        "internet",
		FrontendUUID:       frontendUUID,
		Mode:               effectiveMode,
		IsIncognito:        opts.Incognito,
		Query:              opts.Query,
		Sources:            sources,
		IsProReasoningMode: isProReasoning,
	}

	// Only set model preference if not using turbo
	if modelPref != "" && modelPref != "turbo" {
		req.ModelPreference = &modelPref
	} else if modelPref == "turbo" {
		// For fast mode, explicitly set turbo
		req.ModelPreference = &modelPref
	}

	// Handle follow-up context
	if opts.FollowUp != nil {
		req.BackendUUID = opts.FollowUp.BackendUUID
		req.Attachments = opts.FollowUp.Attachments
	}

	return json.Marshal(req)
}

// searchNonStream performs a non-streaming search.
func (c *Client) searchNonStream(ctx context.Context, opts models.SearchOptions) (*models.SearchResponse, error) {
	chunks := make([]models.StreamChunk, 0)

	ch, err := c.searchStreamChannel(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Collect all chunks
	for chunk := range ch {
		if chunk.Error != nil {
			return nil, chunk.Error
		}
		chunks = append(chunks, chunk)
	}

	// Build response from chunks
	response := &models.SearchResponse{}
	var fullText strings.Builder

	for _, chunk := range chunks {
		if chunk.Text != "" {
			fullText.WriteString(chunk.Text)
		}
		if chunk.Delta != "" {
			fullText.WriteString(chunk.Delta)
		}
		if chunk.BackendUUID != "" {
			response.BackendUUID = chunk.BackendUUID
		}
		if len(chunk.Blocks) > 0 {
			response.Blocks = chunk.Blocks
		}
	}

	response.Text = fullText.String()
	return response, nil
}

// searchStream performs a streaming search and returns the response.
func (c *Client) searchStream(ctx context.Context, opts models.SearchOptions) (*models.SearchResponse, error) {
	return c.searchNonStream(ctx, opts)
}

// searchStreamChannel performs a streaming search and returns a channel.
func (c *Client) searchStreamChannel(ctx context.Context, opts models.SearchOptions) (<-chan models.StreamChunk, error) {
	payload, err := c.buildSearchPayload(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build payload: %w", err)
	}

	ch := make(chan models.StreamChunk, 100)

	go func() {
		defer close(ch)

		resp, err := c.http.Post(searchPath, payload)
		if err != nil {
			ch <- models.StreamChunk{Error: fmt.Errorf("request failed: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			ch <- models.StreamChunk{Error: fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))}
			return
		}

		// Parse SSE stream
		c.parseSSEStream(ctx, resp.Body, ch)
	}()

	return ch, nil
}

// parseSSEStream parses Server-Sent Events from the response body.
func (c *Client) parseSSEStream(ctx context.Context, body io.Reader, ch chan<- models.StreamChunk) {
	scanner := bufio.NewScanner(body)
	// Use larger buffer for SSE chunks
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Custom split function for SSE (double CRLF delimiter)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		// Look for \r\n\r\n delimiter
		if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
			return i + 4, data[0:i], nil
		}

		// Also handle \n\n for compatibility
		if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
			return i + 2, data[0:i], nil
		}

		if atEOF {
			return len(data), data, nil
		}

		return 0, nil, nil
	})

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- models.StreamChunk{Error: ctx.Err()}
			return
		default:
		}

		chunk := scanner.Text()
		if chunk == "" {
			continue
		}

		// Parse SSE format: "event: message\r\ndata: {...}"
		parsed := c.parseSSEChunk(chunk)
		if parsed.Error != nil || parsed.Text != "" || parsed.Delta != "" || parsed.Done {
			ch <- parsed
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- models.StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}

// parseSSEChunk parses a single SSE chunk.
func (c *Client) parseSSEChunk(chunk string) models.StreamChunk {
	// Strip SSE prefix
	data := chunk
	if strings.HasPrefix(chunk, "event: message") {
		// Find data: prefix
		lines := strings.Split(chunk, "\n")
		for _, line := range lines {
			line = strings.TrimPrefix(line, "\r")
			if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
				break
			}
		}
	} else if strings.HasPrefix(chunk, "data: ") {
		data = strings.TrimPrefix(chunk, "data: ")
	}

	if data == "" || data == "[DONE]" {
		return models.StreamChunk{Done: true}
	}

	// Parse outer JSON
	var outer map[string]interface{}
	if err := json.Unmarshal([]byte(data), &outer); err != nil {
		// Not JSON, might be plain text
		return models.StreamChunk{Text: data}
	}

	result := models.StreamChunk{}

	// Extract backend_uuid
	if uuid, ok := outer["backend_uuid"].(string); ok {
		result.BackendUUID = uuid
	}

	// Parse inner text field (double JSON parsing)
	if textField, ok := outer["text"].(string); ok {
		var inner map[string]interface{}
		if err := json.Unmarshal([]byte(textField), &inner); err == nil {
			// Successfully parsed inner JSON
			if blocks, ok := inner["blocks"].([]interface{}); ok {
				result.Blocks = c.parseBlocks(blocks)
			}
		} else {
			// Inner text is not JSON, use as-is
			result.Text = textField
		}
	}

	// Check for delta/content updates
	if delta, ok := outer["delta"].(string); ok {
		result.Delta = delta
	}

	// Check for completion
	if finished, ok := outer["finished"].(bool); ok && finished {
		result.Done = true
	}
	if finishReason, ok := outer["finish_reason"].(string); ok && finishReason != "" {
		result.Done = true
	}

	return result
}

// parseBlocks parses response blocks from JSON.
func (c *Client) parseBlocks(blocks []interface{}) []models.ResponseBlock {
	result := make([]models.ResponseBlock, 0, len(blocks))

	for _, b := range blocks {
		block, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		rb := models.ResponseBlock{}

		// Parse markdown_block
		if mb, ok := block["markdown_block"].(map[string]interface{}); ok {
			rb.MarkdownBlock = &models.MarkdownBlock{}
			if answer, ok := mb["answer"].(string); ok {
				rb.MarkdownBlock.Answer = answer
			}
			// Parse citations
			if cites, ok := mb["citations"].([]interface{}); ok {
				rb.MarkdownBlock.Citations = make([]models.Citation, 0, len(cites))
				for _, cite := range cites {
					if cm, ok := cite.(map[string]interface{}); ok {
						c := models.Citation{}
						if url, ok := cm["url"].(string); ok {
							c.URL = url
						}
						if title, ok := cm["title"].(string); ok {
							c.Title = title
						}
						if snippet, ok := cm["snippet"].(string); ok {
							c.Snippet = snippet
						}
						rb.MarkdownBlock.Citations = append(rb.MarkdownBlock.Citations, c)
					}
				}
			}
		}

		// Parse web_search_results
		if wsr, ok := block["web_search_results"].(map[string]interface{}); ok {
			rb.WebSearchResults = &models.WebSearchResults{}
			if results, ok := wsr["results"].([]interface{}); ok {
				rb.WebSearchResults.Results = make([]models.WebSearchResult, 0, len(results))
				for _, r := range results {
					if rm, ok := r.(map[string]interface{}); ok {
						wr := models.WebSearchResult{}
						if url, ok := rm["url"].(string); ok {
							wr.URL = url
						}
						if title, ok := rm["title"].(string); ok {
							wr.Title = title
						}
						if snippet, ok := rm["snippet"].(string); ok {
							wr.Snippet = snippet
						}
						rb.WebSearchResults.Results = append(rb.WebSearchResults.Results, wr)
					}
				}
			}
		}

		result = append(result, rb)
	}

	return result
}
