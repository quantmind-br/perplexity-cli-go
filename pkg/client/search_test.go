package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/pkg/models"
)

// Helper functions for tests
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func removeFile(path string) {
	os.Remove(path)
}

// mockableHTTPClient is an interface that matches HTTPClient for testing
type mockableHTTPClient interface {
	Post(path string, body []byte) (*http.Response, error)
	buildHeaders() http.Header
	GetCSRFToken() string
	SetCookies(cookies []*http.Cookie)
	AddCookie(cookie *http.Cookie)
	GetCookies() []*http.Cookie
	Close() error
}

// setHTTPClient replaces the HTTP client (for testing only)
func (c *Client) setHTTPClient(mockHTTP mockableHTTPClient) {
	// This is a test helper that uses type assertion to replace the internal client
	// In tests, we can access the private http field
	c.http = &mockHTTPWrapper{client: mockHTTP}
}

// mockHTTPWrapper wraps a mockableHTTPClient to implement *HTTPClient interface
type mockHTTPWrapper struct {
	client mockableHTTPClient
}

func (m *mockHTTPWrapper) Post(path string, body []byte) (*http.Response, error) {
	return m.client.Post(path, body)
}

func (m *mockHTTPWrapper) Get(path string) (*http.Response, error) {
	// Not used in search tests
	return nil, nil
}

func (m *mockHTTPWrapper) PostWithReader(path string, body []byte) (*http.Response, error) {
	return m.client.PostWithReader(path, body)
}

func (m *mockHTTPWrapper) buildHeaders() http.Header {
	return m.client.buildHeaders()
}

func (m *mockHTTPWrapper) GetCSRFToken() string {
	return m.client.GetCSRFToken()
}

func (m *mockHTTPWrapper) SetCookies(cookies []*http.Cookie) {
	m.client.SetCookies(cookies)
}

func (m *mockHTTPWrapper) AddCookie(cookie *http.Cookie) {
	m.client.AddCookie(cookie)
}

func (m *mockHTTPWrapper) GetCookies() []*http.Cookie {
	return m.client.GetCookies()
}

func (m *mockHTTPWrapper) Close() error {
	return m.client.Close()
}

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

	if req.QueryStr != "test query" {
		t.Errorf("Query = %q, want %q", req.QueryStr, "test query")
	}
	if req.Params.Language != "en-US" {
		t.Errorf("Language = %q, want %q", req.Params.Language, "en-US")
	}
	if req.Params.Mode != "copilot" {
		t.Errorf("Mode = %q, want %q", req.Params.Mode, "copilot")
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

	if req.Params.Mode != "concise" {
		t.Errorf("Mode = %q, want %q for fast mode", req.Params.Mode, "concise")
	}
	if req.Params.ModelPreference == nil || *req.Params.ModelPreference != "turbo" {
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
		Model: models.ModelGPT51Thinking,
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
	if req.Params.Mode != "concise" {
		t.Errorf("Mode = %q, want %q for gpt5_thinking", req.Params.Mode, "concise")
	}
	if req.Params.ModelPreference == nil || *req.Params.ModelPreference != "turbo" {
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

	if req.Params.Mode != "copilot" {
		t.Errorf("Mode = %q, want %q", req.Params.Mode, "copilot")
	}
	if !req.Params.IsProReasoningMode {
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

	if req.Params.Mode != "copilot" {
		t.Errorf("Mode = %q, want %q", req.Params.Mode, "copilot")
	}
	if req.Params.ModelPreference == nil || *req.Params.ModelPreference != "pplx_alpha" {
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

	if req.Params.BackendUUID != "test-uuid-123" {
		t.Errorf("BackendUUID = %q, want %q", req.Params.BackendUUID, "test-uuid-123")
	}
	if len(req.Params.Attachments) != 1 {
		t.Errorf("len(Attachments) = %d, want 1", len(req.Params.Attachments))
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

// MockHTTPClient is a mock implementation of HTTPClient for testing
type MockHTTPClient struct {
	Response *http.Response
	Error    error
}

func (m *MockHTTPClient) Post(path string, body []byte) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Response, nil
}

func (m *MockHTTPClient) Get(path string) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Response, nil
}

func (m *MockHTTPClient) PostWithReader(path string, body []byte) (*http.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Response, nil
}

func (m *MockHTTPClient) buildHeaders() http.Header {
	return http.Header{}
}

func (m *MockHTTPClient) GetCSRFToken() string {
	return ""
}

func (m *MockHTTPClient) SetCookies(cookies []*http.Cookie) {}

func (m *MockHTTPClient) AddCookie(cookie *http.Cookie) {}

func (m *MockHTTPClient) GetCookies() []*http.Cookie {
	return nil
}

func (m *MockHTTPClient) Close() error {
	return nil
}

func TestSearchStreamChannel(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Create SSE response
	sseData := `event: message
data: {"text": "Hello", "backend_uuid": "test-uuid-123"}

event: message
data: {"delta": " World"}

event: message
data: [DONE]`

	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sseData)),
		Header:     http.Header{"Content-Type": {"text/event-stream"}},
	}

	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	// Test streaming channel
	ctx := context.Background()
	ch, err := client.searchStreamChannel(ctx, models.SearchOptions{
		Query:    "test query",
		Mode:     models.ModeDefault,
		Model:    models.ModelPplxPro,
		Language: "en-US",
		Sources:  []models.Source{models.SourceWeb},
	})
	if err != nil {
		t.Fatalf("searchStreamChannel() error = %v", err)
	}

	// Collect chunks
	chunksReceived := 0
	for chunk := range ch {
		chunksReceived++
		if chunk.Error != nil {
			t.Errorf("Unexpected error in chunk: %v", chunk.Error)
		}
		if chunk.Done {
			break
		}
	}

	if chunksReceived == 0 {
		t.Error("Expected to receive at least one chunk")
	}
}

func TestSearchStreamChannel_ErrorResponse(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	mockClient := &MockHTTPClient{Error: fmt.Errorf("network error")}
	client.http = &mockHTTPWrapper{client: mockClient}

	ctx := context.Background()
	ch, err := client.searchStreamChannel(ctx, models.SearchOptions{
		Query: "test query",
	})
	if err != nil {
		t.Fatalf("searchStreamChannel() error = %v", err)
	}

	// Should receive error chunk
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error chunk")
	}
}

func TestSearchStreamChannel_Non200Status(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Simulate non-200 status
	response := &http.Response{
		StatusCode: 401,
		Body:       io.NopCloser(strings.NewReader("Unauthorized")),
	}
	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	ctx := context.Background()
	ch, err := client.searchStreamChannel(ctx, models.SearchOptions{
		Query: "test query",
	})
	if err != nil {
		t.Fatalf("searchStreamChannel() error = %v", err)
	}

	// Should receive error chunk
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error chunk for non-200 status")
	}
}

func TestSearchStreamChannel_ContextCancelled(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Create SSE response with delay
	sseData := `event: message
data: {"text": "Hello"}

event: message
data: [DONE]`

	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sseData)),
	}
	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ch, err := client.searchStreamChannel(ctx, models.SearchOptions{
		Query: "test query",
	})
	if err != nil {
		t.Fatalf("searchStreamChannel() error = %v", err)
	}

	// Should receive error chunk for cancelled context
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error chunk for cancelled context")
	}
}

func TestSearchStream(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Create step-based SSE response
	sseData := `event: message
data: [{"step_type": "INITIAL_QUERY", "content": {"query": "test"}, "uuid": ""}]
event: message
data: [{"step_type": "SEARCH_RESULTS", "content": {"web_results": []}, "uuid": ""}]
event: message
data: [{"step_type": "FINAL", "content": {"answer": "{\"answer\": \"Streaming answer\", \"web_results\": [], \"chunks\": [], \"extra_web_results\": [], \"structured_answer\": []}"}, "uuid": "test-uuid"}]
event: message
data: [DONE]`

	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sseData)),
	}
	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	ctx := context.Background()
	result, err := client.searchStream(ctx, models.SearchOptions{
		Query: "test query",
	})
	if err != nil {
		t.Fatalf("searchStream() error = %v", err)
	}

	if result == nil {
		t.Fatal("searchStream() returned nil response")
	}

	if result.Text != "Streaming answer" {
		t.Errorf("Text = %q, want %q", result.Text, "Streaming answer")
	}

	if result.BackendUUID != "test-uuid" {
		t.Errorf("BackendUUID = %q, want %q", result.BackendUUID, "test-uuid")
	}
}

func TestSearchNonStream(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Create legacy format SSE response
	sseData := `event: message
data: {"backend_uuid": "legacy-uuid", "text": "{\"blocks\": [{\"markdown_block\": {\"answer\": \"Legacy answer\", \"citations\": []}}]}"}

event: message
data: [DONE]`

	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sseData)),
	}
	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	ctx := context.Background()
	result, err := client.searchNonStream(ctx, models.SearchOptions{
		Query: "test query",
	})
	if err != nil {
		t.Fatalf("searchNonStream() error = %v", err)
	}

	if result == nil {
		t.Fatal("searchNonStream() returned nil response")
	}

	if result.Text != "Legacy answer" {
		t.Errorf("Text = %q, want %q", result.Text, "Legacy answer")
	}

	if result.BackendUUID != "legacy-uuid" {
		t.Errorf("BackendUUID = %q, want %q", result.BackendUUID, "legacy-uuid")
	}

	if len(result.Blocks) == 0 {
		t.Error("Expected blocks in response")
	}
}

func TestSearchNonStream_WithErrorInChunk(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Create SSE response with error
	sseData := `event: message
data: {"text": "Some text"}

event: message
data: {"error": "Something went wrong"}

event: message
data: [DONE]`

	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sseData)),
	}
	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	ctx := context.Background()
	result, err := client.searchNonStream(ctx, models.SearchOptions{
		Query: "test query",
	})
	if err == nil {
		t.Error("searchNonStream() expected error but got nil")
	}

	if result != nil {
		t.Error("searchNonStream() should return nil on error")
	}
}

func TestSearch(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	sseData := `event: message
data: [{"step_type": "FINAL", "content": {"answer": "{\"answer\": \"Test answer\", \"web_results\": [], \"chunks\": [], \"extra_web_results\": [], \"structured_answer\": []}"}, "uuid": "search-uuid"}]
event: message
data: [DONE]`

	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sseData)),
	}
	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	ctx := context.Background()
	result, err := client.Search(ctx, models.SearchOptions{
		Query:  "test query",
		Stream: false,
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if result == nil {
		t.Fatal("Search() returned nil response")
	}

	if result.Text != "Test answer" {
		t.Errorf("Text = %q, want %q", result.Text, "Test answer")
	}
}

func TestSearchStreamIntegration(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	sseData := `event: message
data: {"delta": "Hello"}

event: message
data: {"delta": " World"}

event: message
data: [DONE]`

	response := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sseData)),
	}
	mockClient := &MockHTTPClient{Response: response}
	client.http = &mockHTTPWrapper{client: mockClient}

	ctx := context.Background()
	ch, err := client.SearchStream(ctx, models.SearchOptions{
		Query: "test query",
	})
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}

	// Collect chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
		if chunk.Done {
			break
		}
	}

	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	// Check that we got the deltas
	if chunks[0].Delta != "Hello" {
		t.Errorf("First chunk Delta = %q, want %q", chunks[0].Delta, "Hello")
	}
	if chunks[1].Delta != " World" {
		t.Errorf("Second chunk Delta = %q, want %q", chunks[1].Delta, " World")
	}
}

func TestParseSSEStream(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with SSE format using \r\n\r\n delimiter
	sseData := "event: message\ndata: {\"text\": \"First chunk\"}\r\n\r\nevent: message\ndata: {\"text\": \"Second chunk\"}\r\n\rnevent: message\ndata: [DONE]\r\n\r\n"

	ch := make(chan models.StreamChunk, 10)
	go func() {
		client.parseSSEStream(ctx, strings.NewReader(sseData), ch)
		close(ch)
	}()

	// Collect chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should have text
	if len(chunks) > 0 && chunks[0].Text != "First chunk" {
		t.Errorf("First chunk Text = %q, want %q", chunks[0].Text, "First chunk")
	}

	// Second chunk should have text
	if len(chunks) > 1 && chunks[1].Text != "Second chunk" {
		t.Errorf("Second chunk Text = %q, want %q", chunks[1].Text, "Second chunk")
	}
}

func TestParseSSEStream_WithLFOnly(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with SSE format using \n\n delimiter (alternative format)
	sseData := "event: message\ndata: {\"text\": \"Test with LF\"}\n\nevent: message\ndata: [DONE]\n\n"

	ch := make(chan models.StreamChunk, 10)
	go func() {
		client.parseSSEStream(ctx, strings.NewReader(sseData), ch)
		close(ch)
	}()

	// Collect chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least 1 chunk")
	}

	if len(chunks) > 0 && chunks[0].Text != "Test with LF" {
		t.Errorf("Chunk Text = %q, want %q", chunks[0].Text, "Test with LF")
	}
}

func TestParseSSEStream_WithEmptyChunks(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with empty chunks between data
	sseData := "event: message\ndata: {\"text\": \"First\"}\r\n\r\n\r\n\r\nevent: message\ndata: {\"text\": \"Second\"}\r\n\r\n\r\n\r\nevent: message\ndata: [DONE]\r\n\r\n"

	ch := make(chan models.StreamChunk, 10)
	go func() {
		client.parseSSEStream(ctx, strings.NewReader(sseData), ch)
		close(ch)
	}()

	// Collect chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should handle empty lines gracefully
	if len(chunks) < 2 {
		t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
	}
}

func TestParseSSEStream_WithEventPrefix(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Test with explicit "event: message" prefix
	sseData := "event: message\ndata: {\"text\": \"Prefixed\"}\r\n\r\nevent: message\ndata: [DONE]\r\n\r\n"

	ch := make(chan models.StreamChunk, 10)
	go func() {
		client.parseSSEStream(ctx, strings.NewReader(sseData), ch)
		close(ch)
	}()

	// Collect chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least 1 chunk")
	}

	if len(chunks) > 0 && chunks[0].Text != "Prefixed" {
		t.Errorf("Chunk Text = %q, want %q", chunks[0].Text, "Prefixed")
	}
}

func TestParseSSEStream_WithLargeBuffer(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a large SSE response to test buffer handling
	var buf strings.Builder
	buf.WriteString("event: message\ndata: {\"text\": \"")
	// Write large text (60KB+)
	for i := 0; i < 60000; i++ {
		buf.WriteRune('a')
	}
	buf.WriteString("\"}\r\n\r\nevent: message\ndata: [DONE]\r\n\r\n")

	ch := make(chan models.StreamChunk, 10)
	go func() {
		client.parseSSEStream(ctx, strings.NewReader(buf.String()), ch)
		close(ch)
	}()

	// Collect chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least 1 chunk")
	}
}

func TestParseSSEStream_WithContextCancellation(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// SSE data with delay
	sseData := "event: message\ndata: {\"text\": \"Should not parse\"}\r\n\r\nevent: message\ndata: [DONE]\r\n\r\n"

	ch := make(chan models.StreamChunk, 10)
	go func() {
		client.parseSSEStream(ctx, strings.NewReader(sseData), ch)
		close(ch)
	}()

	// Should receive error due to context cancellation
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error due to context cancellation")
	}
}

func TestParseSSEStream_WithScannerError(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Create a reader that will cause scanner error
	invalidData := strings.Repeat("a", 2000) // Larger than default buffer

	ch := make(chan models.StreamChunk, 10)
	go func() {
		client.parseSSEStream(ctx, strings.NewReader(invalidData), ch)
		close(ch)
	}()

	// Should handle scanner error gracefully
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error for invalid SSE data")
	}
}

func TestParseSSEChunk_WithLegacyInnerJSON(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test legacy format where text field contains inner JSON with blocks
	chunk := `{"backend_uuid": "legacy-uuid", "text": "{\"blocks\": [{\"markdown_block\": {\"answer\": \"Legacy block answer\", \"citations\": [{\"url\": \"https://example.com\", \"title\": \"Example\", \"snippet\": \"Example snippet\"}]}}, {\"web_search_results\": {\"results\": [{\"url\": \"https://example.com/result\", \"title\": \"Result\", \"snippet\": \"Result snippet\"}]}}]}"}`

	result := client.parseSSEChunk(chunk)

	if result.BackendUUID != "legacy-uuid" {
		t.Errorf("BackendUUID = %q, want %q", result.BackendUUID, "legacy-uuid")
	}

	if len(result.Blocks) != 2 {
		t.Errorf("len(Blocks) = %d, want 2", len(result.Blocks))
	}

	if result.Blocks[0].MarkdownBlock == nil {
		t.Error("MarkdownBlock should not be nil")
	} else if result.Blocks[0].MarkdownBlock.Answer != "Legacy block answer" {
		t.Errorf("Block Answer = %q, want %q", result.Blocks[0].MarkdownBlock.Answer, "Legacy block answer")
	}

	if len(result.Blocks[0].MarkdownBlock.Citations) != 1 {
		t.Errorf("len(Citations) = %d, want 1", len(result.Blocks[0].MarkdownBlock.Citations))
	} else if result.Blocks[0].MarkdownBlock.Citations[0].URL != "https://example.com" {
		t.Errorf("Citation URL = %q, want %q", result.Blocks[0].MarkdownBlock.Citations[0].URL, "https://example.com")
	}

	if result.Blocks[1].WebSearchResults == nil {
		t.Error("WebSearchResults should not be nil")
	} else if len(result.Blocks[1].WebSearchResults.Results) != 1 {
		t.Errorf("len(WebSearchResults) = %d, want 1", len(result.Blocks[1].WebSearchResults.Results))
	}
}

func TestParseSSEChunk_WithNestedStepBasedFormat(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test where text field contains step-based format that's already FINAL
	chunk := `{"backend_uuid": "nested-uuid", "text": "[{\"step_type\": \"FINAL\", \"content\": {\"answer\": \"{\\\"answer\\\": \\\"Nested FINAL answer\\\", \\\"web_results\\\": [{\\\"url\\\": \\\"https://nested.com\\\", \\\"title\\\": \\\"Nested\\\", \\\"snippet\\\": \\\"Nested snippet\\\"}], \\\"chunks\\\": [\\\"Nested\\\", \\\" FINAL\\\", \\\" answer\\\"], \\\"extra_web_results\\\": [], \\\"structured_answer\\\": []}\"}, \"uuid\": \"nested-final-uuid\"}]"}`

	result := client.parseSSEChunk(chunk)

	if result.StepType != "FINAL" {
		t.Errorf("StepType = %q, want %q", result.StepType, "FINAL")
	}

	if result.Text != "Nested FINAL answer" {
		t.Errorf("Text = %q, want %q", result.Text, "Nested FINAL answer")
	}

	if !result.Done {
		t.Error("Done should be true for FINAL step")
	}

	if len(result.WebResults) != 1 {
		t.Errorf("len(WebResults) = %d, want 1", len(result.WebResults))
	} else if result.WebResults[0].URL != "https://nested.com" {
		t.Errorf("WebResults[0].URL = %q, want %q", result.WebResults[0].URL, "https://nested.com")
	}

	if len(result.Chunks) != 3 {
		t.Errorf("len(Chunks) = %d, want 3", len(result.Chunks))
	}
}

func TestParseSSEChunk_WithDirectStepBasedFormat(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test direct step-based format (not nested in text field)
	chunk := `[{"step_type": "INITIAL_QUERY", "content": {"query": "direct step query"}, "uuid": "direct-query-uuid"}, {"step_type": "SEARCH_WEB", "content": {}, "uuid": ""}, {"step_type": "FINAL", "content": {"answer": "{\"answer\": \"Direct FINAL answer\", \"web_results\": [], \"chunks\": [], \"extra_web_results\": [], \"structured_answer\": []}"}, "uuid": "direct-final-uuid"}]`

	result := client.parseSSEChunk(chunk)

	if result.StepType != "FINAL" {
		t.Errorf("StepType = %q, want %q", result.StepType, "FINAL")
	}

	if result.Text != "Direct FINAL answer" {
		t.Errorf("Text = %q, want %q", result.Text, "Direct FINAL answer")
	}
}

func TestParseSSEChunk_WithInvalidJSON(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with invalid JSON - should return as plain text
	chunk := `{"text": "not valid json`

	result := client.parseSSEChunk(chunk)

	if result.Text != `{"text": "not valid json` {
		t.Errorf("Text = %q, want %q", result.Text, `{"text": "not valid json`)
	}
}

func TestParseSSEChunk_WithStepBasedNonFinal(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with non-FINAL step type
	chunk := `[{"step_type": "SEARCH_WEB", "content": {"url": "https://search.com"}, "uuid": "search-uuid"}]`

	result := client.parseSSEChunk(chunk)

	if result.StepType != "SEARCH_WEB" {
		t.Errorf("StepType = %q, want %q", result.StepType, "SEARCH_WEB")
	}

	if result.BackendUUID != "search-uuid" {
		t.Errorf("BackendUUID = %q, want %q", result.BackendUUID, "search-uuid")
	}

	// Non-FINAL steps should not have Done set
	if result.Done {
		t.Error("Done should be false for non-FINAL step")
	}
}

func TestParseStepBasedResponse_WithMultipleSteps(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with multiple steps including search results and final
	data := `[{"step_type": "SEARCH_RESULTS", "content": {"goal_id": "0", "web_results": [{"name": "Result 1", "url": "https://result1.com", "snippet": "Snippet 1"}, {"name": "Result 2", "url": "https://result2.com", "snippet": "Snippet 2"}]}, "uuid": "search-results-uuid"}, {"step_type": "FINAL", "content": {"answer": "{\"answer\": \"Final answer with multiple results\", \"web_results\": [], \"chunks\": [], \"extra_web_results\": [{\"name\": \"Extra\", \"url\": \"https://extra.com\", \"snippet\": \"Extra snippet\"}], \"structured_answer\": []}"}, "uuid": "final-uuid"}]`

	result := client.parseStepBasedResponse(data)

	if result.StepType != "FINAL" {
		t.Errorf("StepType = %q, want %q", result.StepType, "FINAL")
	}

	if result.Text != "Final answer with multiple results" {
		t.Errorf("Text = %q, want %q", result.Text, "Final answer with multiple results")
	}

	// Should have web results from SEARCH_RESULTS step
	if len(result.WebResults) < 3 {
		t.Errorf("Expected at least 3 WebResults, got %d", len(result.WebResults))
	}

	// Should have extra web results from FINAL step
	if len(result.WebResults) >= 3 {
		if result.WebResults[2].URL != "https://extra.com" {
			t.Errorf("Extra WebResults[2].URL = %q, want %q", result.WebResults[2].URL, "https://extra.com")
		}
	}
}

func TestParseStepBasedResponse_WithMalformedJSON(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test with malformed JSON - should return as plain text
	data := `[{"step_type": "FINAL", "content": {"answer": "not valid json`

	result := client.parseStepBasedResponse(data)

	if result.Text == "" {
		t.Error("Expected Text to contain raw data for malformed JSON")
	}
}

func TestParseStepBasedResponse_WithExtractionLogic(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Test the array extraction logic
	data := `some prefix [{"step_type": "FINAL", "content": {"answer": "{\"answer\": \"Extracted answer\", \"web_results\": [], \"chunks\": [], \"extra_web_results\": [], \"structured_answer\": []}"}, "uuid": ""}] some suffix`

	result := client.parseStepBasedResponse(data)

	if result.StepType != "FINAL" {
		t.Errorf("StepType = %q, want %q", result.StepType, "FINAL")
	}

	if result.Text != "Extracted answer" {
		t.Errorf("Text = %q, want %q", result.Text, "Extracted answer")
	}
}

func TestNewClientWithCookieFile(t *testing.T) {
	// Create a temp cookie file
	tmpFile := "/tmp/test-cookies.json"
	cookieData := `[{"name": "test", "value": "value", "domain": ".perplexity.ai", "path": "/"}]`
	err := writeFile(tmpFile, []byte(cookieData))
	if err != nil {
		t.Fatalf("Failed to create temp cookie file: %v", err)
	}
	defer removeFile(tmpFile)

	client, err := NewWithCookieFile(tmpFile)
	if err != nil {
		t.Fatalf("NewWithCookieFile() error = %v", err)
	}
	defer client.Close()

	if !client.HasValidSession() {
		// The cookie doesn't have CSRF token, so this is expected
		// But the client should have been created
	}
}

func TestNewClientWithCookies(t *testing.T) {
	cookies := []*http.Cookie{
		{Name: "test-cookie", Value: "test-value", Domain: ".perplexity.ai"},
		{Name: "next-auth.csrf-token", Value: "csrf-token", Domain: ".perplexity.ai"},
	}

	client, err := NewWithCookies(cookies)
	if err != nil {
		t.Fatalf("NewWithCookies() error = %v", err)
	}
	defer client.Close()

	if !client.HasValidSession() {
		t.Error("Client should have valid session with CSRF token")
	}

	retrievedCookies := client.GetCookies()
	if len(retrievedCookies) == 0 {
		t.Error("GetCookies() should return cookies")
	}
}

func TestClientGetCookies(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	cookies := []*http.Cookie{
		{Name: "cookie1", Value: "value1", Domain: ".perplexity.ai"},
		{Name: "cookie2", Value: "value2", Domain: ".perplexity.ai"},
	}

	client.SetCookies(cookies)

	retrieved := client.GetCookies()
	if len(retrieved) != 2 {
		t.Errorf("len(retrieved cookies) = %d, want 2", len(retrieved))
	}
}
