package client

import (
	"encoding/json"
	"testing"

	"github.com/diogo/perplexity-go/pkg/models"
)

func TestBuildSearchPayload(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	opts := models.SearchOptions{
		Query:    "test query",
		Mode:     models.ModeDefault,
		Model:    models.ModelPplxPro,
		Language: "en-US",
		Sources:  []models.Source{models.SourceWeb},
	}

	payload, err := client.buildSearchPayload(opts)
	if err != nil {
		t.Fatalf("buildSearchPayload() error = %v", err)
	}

	var req models.SearchRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if req.Query != "test query" {
		t.Errorf("Query = %q, want %q", req.Query, "test query")
	}
	if req.Language != "en-US" {
		t.Errorf("Language = %q, want %q", req.Language, "en-US")
	}
	if req.Mode != "copilot" {
		t.Errorf("Mode = %q, want %q", req.Mode, "copilot")
	}
}

func TestBuildSearchPayloadFastMode(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	opts := models.SearchOptions{
		Query: "test query",
		Mode:  models.ModeFast,
	}

	payload, err := client.buildSearchPayload(opts)
	if err != nil {
		t.Fatalf("buildSearchPayload() error = %v", err)
	}

	var req models.SearchRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if req.Mode != "concise" {
		t.Errorf("Mode = %q, want %q for fast mode", req.Mode, "concise")
	}
	if req.ModelPreference == nil || *req.ModelPreference != "turbo" {
		t.Errorf("ModelPreference should be turbo for fast mode")
	}
}

func TestBuildSearchPayloadGPT5Thinking(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	opts := models.SearchOptions{
		Query: "test query",
		Mode:  models.ModeDefault,
		Model: models.ModelGPT5Thinking,
	}

	payload, err := client.buildSearchPayload(opts)
	if err != nil {
		t.Fatalf("buildSearchPayload() error = %v", err)
	}

	var req models.SearchRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	// GPT5Thinking should force concise mode with turbo
	if req.Mode != "concise" {
		t.Errorf("Mode = %q, want %q for gpt5_thinking", req.Mode, "concise")
	}
	if req.ModelPreference == nil || *req.ModelPreference != "turbo" {
		t.Errorf("ModelPreference should be turbo for gpt5_thinking")
	}
}

func TestBuildSearchPayloadReasoningMode(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	opts := models.SearchOptions{
		Query: "test query",
		Mode:  models.ModeReasoning,
	}

	payload, err := client.buildSearchPayload(opts)
	if err != nil {
		t.Fatalf("buildSearchPayload() error = %v", err)
	}

	var req models.SearchRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if req.Mode != "copilot" {
		t.Errorf("Mode = %q, want %q", req.Mode, "copilot")
	}
	if !req.IsProReasoningMode {
		t.Error("IsProReasoningMode should be true for reasoning mode")
	}
}

func TestBuildSearchPayloadDeepResearch(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	opts := models.SearchOptions{
		Query: "test query",
		Mode:  models.ModeDeepResearch,
	}

	payload, err := client.buildSearchPayload(opts)
	if err != nil {
		t.Fatalf("buildSearchPayload() error = %v", err)
	}

	var req models.SearchRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if req.Mode != "copilot" {
		t.Errorf("Mode = %q, want %q", req.Mode, "copilot")
	}
	if req.ModelPreference == nil || *req.ModelPreference != "pplx_alpha" {
		t.Errorf("ModelPreference should be pplx_alpha for deep-research")
	}
}

func TestBuildSearchPayloadWithFollowUp(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	opts := models.SearchOptions{
		Query: "follow up query",
		Mode:  models.ModeDefault,
		FollowUp: &models.FollowUpContext{
			BackendUUID: "test-uuid-123",
			Attachments: []models.Attachment{
				{URL: "https://example.com/file.pdf"},
			},
		},
	}

	payload, err := client.buildSearchPayload(opts)
	if err != nil {
		t.Fatalf("buildSearchPayload() error = %v", err)
	}

	var req models.SearchRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	if req.BackendUUID != "test-uuid-123" {
		t.Errorf("BackendUUID = %q, want %q", req.BackendUUID, "test-uuid-123")
	}
	if len(req.Attachments) != 1 {
		t.Errorf("len(Attachments) = %d, want 1", len(req.Attachments))
	}
}

func TestParseSSEChunk(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	tests := []struct {
		name  string
		chunk string
		want  models.StreamChunk
	}{
		{
			name:  "done signal",
			chunk: "data: [DONE]",
			want:  models.StreamChunk{Done: true},
		},
		{
			name:  "empty data",
			chunk: "",
			want:  models.StreamChunk{Done: true},
		},
		{
			name:  "plain text",
			chunk: "Hello world",
			want:  models.StreamChunk{Text: "Hello world"},
		},
		{
			name:  "json with backend_uuid",
			chunk: `data: {"backend_uuid": "test-uuid"}`,
			want:  models.StreamChunk{BackendUUID: "test-uuid"},
		},
		{
			name:  "json with delta",
			chunk: `data: {"delta": "new text"}`,
			want:  models.StreamChunk{Delta: "new text"},
		},
		{
			name:  "json with finished",
			chunk: `data: {"finished": true}`,
			want:  models.StreamChunk{Done: true},
		},
		{
			name:  "json with finish_reason",
			chunk: `data: {"finish_reason": "stop"}`,
			want:  models.StreamChunk{Done: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.parseSSEChunk(tt.chunk)
			if got.Done != tt.want.Done {
				t.Errorf("Done = %v, want %v", got.Done, tt.want.Done)
			}
			if got.BackendUUID != tt.want.BackendUUID {
				t.Errorf("BackendUUID = %q, want %q", got.BackendUUID, tt.want.BackendUUID)
			}
			if got.Delta != tt.want.Delta {
				t.Errorf("Delta = %q, want %q", got.Delta, tt.want.Delta)
			}
			if got.Text != tt.want.Text {
				t.Errorf("Text = %q, want %q", got.Text, tt.want.Text)
			}
		})
	}
}

func TestParseBlocks(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	blocks := []interface{}{
		map[string]interface{}{
			"markdown_block": map[string]interface{}{
				"answer": "This is the answer",
				"citations": []interface{}{
					map[string]interface{}{
						"url":   "https://example.com",
						"title": "Example",
					},
				},
			},
		},
		map[string]interface{}{
			"web_search_results": map[string]interface{}{
				"results": []interface{}{
					map[string]interface{}{
						"url":     "https://example.com/result",
						"title":   "Result Title",
						"snippet": "Result snippet",
					},
				},
			},
		},
	}

	result := client.parseBlocks(blocks)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Check markdown block
	if result[0].MarkdownBlock == nil {
		t.Fatal("MarkdownBlock should not be nil")
	}
	if result[0].MarkdownBlock.Answer != "This is the answer" {
		t.Errorf("Answer = %q, want %q", result[0].MarkdownBlock.Answer, "This is the answer")
	}
	if len(result[0].MarkdownBlock.Citations) != 1 {
		t.Errorf("len(Citations) = %d, want 1", len(result[0].MarkdownBlock.Citations))
	}

	// Check web search results
	if result[1].WebSearchResults == nil {
		t.Fatal("WebSearchResults should not be nil")
	}
	if len(result[1].WebSearchResults.Results) != 1 {
		t.Errorf("len(Results) = %d, want 1", len(result[1].WebSearchResults.Results))
	}
	if result[1].WebSearchResults.Results[0].URL != "https://example.com/result" {
		t.Errorf("URL = %q, want %q", result[1].WebSearchResults.Results[0].URL, "https://example.com/result")
	}
}

func TestParseStepBasedResponse(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with a complete step-based response
	data := `[{"step_type": "INITIAL_QUERY", "content": {"query": "test"}, "uuid": ""}, {"step_type": "FINAL", "content": {"answer": "{\"answer\": \"This is the answer.\", \"web_results\": [], \"chunks\": [\"This\", \" is\", \" the\", \" answer.\"], \"extra_web_results\": [], \"structured_answer\": []}"}, "uuid": "test-uuid"}]`

	result := client.parseStepBasedResponse(data)

	if result.StepType != "FINAL" {
		t.Errorf("StepType = %q, want %q", result.StepType, "FINAL")
	}
	if result.Text != "This is the answer." {
		t.Errorf("Text = %q, want %q", result.Text, "This is the answer.")
	}
	if len(result.Chunks) != 4 {
		t.Errorf("len(Chunks) = %d, want 4", len(result.Chunks))
	}
}

func TestParseStepBasedResponse_WithWebResults(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with web results in SEARCH_RESULTS step
	data := `[{"step_type": "SEARCH_RESULTS", "content": {"goal_id": "0", "web_results": [{"name": "Test", "url": "https://example.com", "snippet": "Test snippet"}]}, "uuid": "test-uuid"}]`

	result := client.parseStepBasedResponse(data)

	if result.StepType != "SEARCH_RESULTS" {
		t.Errorf("StepType = %q, want %q", result.StepType, "SEARCH_RESULTS")
	}
	if len(result.WebResults) != 1 {
		t.Fatalf("len(WebResults) = %d, want 1", len(result.WebResults))
	}
	if result.WebResults[0].URL != "https://example.com" {
		t.Errorf("WebResults[0].URL = %q, want %q", result.WebResults[0].URL, "https://example.com")
	}
}

func TestParseStepBasedResponse_WithTrailingSSEMarker(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with trailing SSE marker
	data := `[{"step_type": "FINAL", "content": {"answer": "{\"answer\": \"Answer text.\", \"web_results\": [], \"chunks\": [], \"extra_web_results\": [], \"structured_answer\": []}"}, "uuid": ""}]event: end_of_stream
data: {}`

	result := client.parseStepBasedResponse(data)

	if result.StepType != "FINAL" {
		t.Errorf("StepType = %q, want %q", result.StepType, "FINAL")
	}
	if result.Text != "Answer text." {
		t.Errorf("Text = %q, want %q", result.Text, "Answer text.")
	}
}

func TestParseSSEChunk_NewFormat(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with new format where text field contains step array
	chunk := `{"backend_uuid": "test-uuid", "text": "[{\"step_type\": \"FINAL\", \"content\": {\"answer\": \"{\\\"answer\\\": \\\"The answer.\\\", \\\"web_results\\\": [], \\\"chunks\\\": [], \\\"extra_web_results\\\": [], \\\"structured_answer\\\": []}\"}, \"uuid\": \"\"}]"}`

	result := client.parseSSEChunk(chunk)

	if result.StepType != "FINAL" {
		t.Errorf("StepType = %q, want %q", result.StepType, "FINAL")
	}
	if result.Text != "The answer." {
		t.Errorf("Text = %q, want %q", result.Text, "The answer.")
	}
	if !result.Done {
		t.Error("Done should be true for FINAL step")
	}
}

func TestParseSSEChunk_EndOfStream(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	tests := []struct {
		name  string
		chunk string
	}{
		{"event: end_of_stream", "event: end_of_stream\ndata: {}"},
		{"empty object", "{}"},
		{"data: empty object", "data: {}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.parseSSEChunk(tt.chunk)
			if !result.Done {
				t.Errorf("Done = false, want true for %q", tt.chunk)
			}
		})
	}
}
