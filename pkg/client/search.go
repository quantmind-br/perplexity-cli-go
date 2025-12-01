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

	// Handle special thinking models (forces fast mode regardless of mode setting)
	if opts.Model == models.ModelGPT51Thinking {
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
		Params: models.SearchParams{
			Version:            "2.18",
			Source:             "default",
			Language:           opts.Language,
			Timezone:           "America/New_York",
			SearchFocus:        "internet",
			FrontendUUID:       frontendUUID,
			Mode:               effectiveMode,
			IsIncognito:        opts.Incognito,
			Sources:            sources,
			IsProReasoningMode: isProReasoning,
		},
		QueryStr: opts.Query,
	}

	// Only set model preference if not using turbo
	if modelPref != "" && modelPref != "turbo" {
		req.Params.ModelPreference = &modelPref
	} else if modelPref == "turbo" {
		// For fast mode, explicitly set turbo
		req.Params.ModelPreference = &modelPref
	}

	// Handle follow-up context
	if opts.FollowUp != nil {
		req.Params.BackendUUID = opts.FollowUp.BackendUUID
		req.Params.Attachments = opts.FollowUp.Attachments
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
		// Collect web results from new format
		if len(chunk.WebResults) > 0 {
			response.WebResults = append(response.WebResults, chunk.WebResults...)
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

		resp, err := c.http.Post(searchPath, bytes.NewReader(payload), nil)
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
	// Ignore SSE comments (lines starting with ':') - these are keep-alive pings
	// Format: ": ping - 2025-12-01 15:52:17.152450"
	if strings.HasPrefix(chunk, ":") {
		return models.StreamChunk{}
	}

	// Ignore ping events
	if strings.HasPrefix(chunk, "event: ping") || strings.Contains(chunk, "\nevent: ping") {
		return models.StreamChunk{}
	}

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

	// Handle end of stream
	if data == "" || data == "{}" {
		return models.StreamChunk{Done: true}
	}

	// Check for end_of_stream event
	if strings.HasPrefix(data, "event: end_of_stream") {
		return models.StreamChunk{Done: true}
	}

	// Check for [DONE] marker
	if strings.Contains(data, "[DONE]") {
		return models.StreamChunk{Done: true}
	}

	// Try to parse as new step-based format (array of steps - direct)
	trimmedData := strings.TrimSpace(data)
	if strings.HasPrefix(trimmedData, "[{") && strings.Contains(trimmedData, "step_type") {
		return c.parseStepBasedResponse(trimmedData)
	}

	// Parse as outer JSON object
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

	// Parse inner text field (new format: text contains step array as string)
	if textField, ok := outer["text"].(string); ok {
		textTrimmed := strings.TrimSpace(textField)
		// Check if text contains step-based format
		if strings.HasPrefix(textTrimmed, "[{") && strings.Contains(textTrimmed, "step_type") {
			// Parse the step array from the text field
			stepResult := c.parseStepBasedResponse(textTrimmed)
			if stepResult.StepType == "FINAL" && stepResult.Text != "" {
				result.Text = stepResult.Text
				result.StepType = stepResult.StepType
				result.WebResults = stepResult.WebResults
				result.Chunks = stepResult.Chunks
				result.Done = true
				return result
			}
			// For non-FINAL steps, just note the step type
			result.StepType = stepResult.StepType
			return result
		}

		// Try legacy format: inner JSON with blocks
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

// parseStepBasedResponse parses the new step-based response format.
func (c *Client) parseStepBasedResponse(data string) models.StreamChunk {
	// Handle case where data might contain trailing SSE markers
	// The data might look like: [...JSON...]event: end_of_stream\ndata: {}
	if idx := strings.Index(data, "]event:"); idx > 0 {
		data = data[:idx+1]
	}
	if idx := strings.Index(data, "]\nevent:"); idx > 0 {
		data = data[:idx+1]
	}
	if idx := strings.Index(data, "]\r\nevent:"); idx > 0 {
		data = data[:idx+1]
	}
	data = strings.TrimSpace(data)

	var steps []models.SSEStep
	if err := json.Unmarshal([]byte(data), &steps); err != nil {
		// Try to extract just the array part
		startIdx := strings.Index(data, "[")
		endIdx := strings.LastIndex(data, "]")
		if startIdx >= 0 && endIdx > startIdx {
			data = data[startIdx : endIdx+1]
			if err := json.Unmarshal([]byte(data), &steps); err != nil {
				return models.StreamChunk{Text: data}
			}
		} else {
			return models.StreamChunk{Text: data}
		}
	}

	result := models.StreamChunk{}

	// Process each step
	for _, step := range steps {
		switch step.StepType {
		case "FINAL":
			// Parse final content
			contentMap, ok := step.Content.(map[string]interface{})
			if !ok {
				continue
			}

			answerJSON, ok := contentMap["answer"].(string)
			if !ok {
				continue
			}

			// Parse the nested answer JSON
			var finalAnswer models.FinalAnswer
			if err := json.Unmarshal([]byte(answerJSON), &finalAnswer); err != nil {
				// If parsing fails, try to use the raw answer
				result.Text = answerJSON
				continue
			}

			// Extract the answer text
			result.Text = finalAnswer.Answer
			result.StepType = "FINAL"
			result.Chunks = finalAnswer.Chunks
			result.WebResults = finalAnswer.WebResults

			// Also add extra web results
			if len(finalAnswer.ExtraWebResults) > 0 {
				result.WebResults = append(result.WebResults, finalAnswer.ExtraWebResults...)
			}

		case "SEARCH_RESULTS":
			// Parse search results
			contentMap, ok := step.Content.(map[string]interface{})
			if !ok {
				continue
			}

			webResultsRaw, ok := contentMap["web_results"].([]interface{})
			if !ok {
				continue
			}

			for _, wrRaw := range webResultsRaw {
				wrMap, ok := wrRaw.(map[string]interface{})
				if !ok {
					continue
				}

				wr := models.WebResult{}
				if name, ok := wrMap["name"].(string); ok {
					wr.Name = name
				}
				if url, ok := wrMap["url"].(string); ok {
					wr.URL = url
				}
				if snippet, ok := wrMap["snippet"].(string); ok {
					wr.Snippet = snippet
				}
				if title, ok := wrMap["title"].(string); ok {
					wr.Title = title
				}
				result.WebResults = append(result.WebResults, wr)
			}
			result.StepType = "SEARCH_RESULTS"

		case "INITIAL_QUERY":
			result.StepType = "INITIAL_QUERY"
			// Ignore initial query content

		case "SEARCH_WEB":
			result.StepType = "SEARCH_WEB"
			// Ignore search web step
		}

		// Extract UUID
		if step.UUID != "" {
			result.BackendUUID = step.UUID
		}
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
