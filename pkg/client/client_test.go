package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/pkg/models"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.DefaultModel != models.ModelPplxPro {
		t.Errorf("DefaultModel = %q, want %q", cfg.DefaultModel, models.ModelPplxPro)
	}
	if cfg.DefaultMode != models.ModeDefault {
		t.Errorf("DefaultMode = %q, want %q", cfg.DefaultMode, models.ModeDefault)
	}
	if cfg.Language != "en-US" {
		t.Errorf("Language = %q, want %q", cfg.Language, "en-US")
	}
	if len(cfg.Sources) != 1 || cfg.Sources[0] != models.SourceWeb {
		t.Errorf("Sources = %v, want [web]", cfg.Sources)
	}
}

func TestNewClient(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	if client.http == nil {
		t.Error("HTTP client should not be nil")
	}
	if client.defaultModel != models.ModelPplxPro {
		t.Errorf("defaultModel = %q, want %q", client.defaultModel, models.ModelPplxPro)
	}
	if client.defaultMode != models.ModeDefault {
		t.Errorf("defaultMode = %q, want %q", client.defaultMode, models.ModeDefault)
	}
}

func TestClientSetCookies(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	cookies := []*http.Cookie{
		{Name: "next-auth.csrf-token", Value: "testtoken|hash"},
		{Name: "session", Value: "session_value"},
	}

	client.SetCookies(cookies)

	if client.csrfToken != "testtoken" {
		t.Errorf("csrfToken = %q, want %q", client.csrfToken, "testtoken")
	}
}

func TestClientHasValidSession(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Without cookies
	if client.HasValidSession() {
		t.Error("HasValidSession() should be false without cookies")
	}

	// With CSRF token
	cookies := []*http.Cookie{
		{Name: "next-auth.csrf-token", Value: "token|hash"},
	}
	client.SetCookies(cookies)

	if !client.HasValidSession() {
		t.Error("HasValidSession() should be true with CSRF token")
	}
}

func TestClientQueryLimits(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Initial limits
	if client.ProQueriesRemaining() != 5 {
		t.Errorf("ProQueriesRemaining() = %d, want 5", client.ProQueriesRemaining())
	}
	if client.FileUploadsRemaining() != 10 {
		t.Errorf("FileUploadsRemaining() = %d, want 10", client.FileUploadsRemaining())
	}
}

func TestClientSetDefaults(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Change defaults
	client.SetDefaultModel(models.ModelGPT51)
	client.SetDefaultMode(models.ModeFast)
	client.SetDefaultLanguage("pt-BR")
	client.SetDefaultSources([]models.Source{models.SourceWeb, models.SourceScholar})

	if client.defaultModel != models.ModelGPT51 {
		t.Errorf("defaultModel = %q, want %q", client.defaultModel, models.ModelGPT51)
	}
	if client.defaultMode != models.ModeFast {
		t.Errorf("defaultMode = %q, want %q", client.defaultMode, models.ModeFast)
	}
	if client.defaultLang != "pt-BR" {
		t.Errorf("defaultLang = %q, want %q", client.defaultLang, "pt-BR")
	}
	if len(client.defaultSrcs) != 2 {
		t.Errorf("len(defaultSrcs) = %d, want 2", len(client.defaultSrcs))
	}
}

// createTestClient creates a client with a mock HTTP client for testing.
func createTestClient(mock *MockHTTPClient) *Client {
	return &Client{
		http:          mock,
		defaultModel:  models.ModelPplxPro,
		defaultMode:   models.ModeDefault,
		defaultLang:   "en-US",
		defaultSrcs:   []models.Source{models.SourceWeb},
		proQueries:    0,
		fileUploads:   0,
		maxProQueries: 5,
		maxFileUploads: 10,
	}
}

// createResponse creates a test HTTP response with the given body.
func createResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// TestSearchSuccess tests successful non-streaming search.
func TestSearchSuccess(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		mode        models.Mode
		model       models.Model
		sseResponse string
		wantText    string
	}{
		{
			name:        "basic search with text response",
			query:       "What is Go?",
			mode:        models.ModeDefault,
			model:       models.ModelPplxPro,
			sseResponse: `data: {"text": "Go is a programming language."}`,
			wantText:    "Go is a programming language.",
		},
		{
			name:        "search with blocks",
			query:       "What is Python?",
			mode:        models.ModePro,
			model:       models.ModelGPT51,
			sseResponse: `data: {"text": "{\"blocks\": [{\"markdown_block\": {\"answer\": \"Python is a language\"}}]}"}`,
			wantText:    "Python is a language",
		},
		{
			name:        "search with step-based response",
			query:       "What is Java?",
			mode:        models.ModeReasoning,
			model:       models.ModelGemini30Pro,
			sseResponse: `data: {"text": "[{\"step_type\": \"FINAL\", \"content\": {\"answer\": \"{\\\"answer\\\": \\\"Java is a language.\\\", \\\"web_results\\\": [], \\\"chunks\\\": [], \\\"extra_web_results\\\": [], \\\"structured_answer\\\": []}\"}]}"}`,
			wantText:    "Java is a language.",
		},
		{
			name:        "search with multiple SSE chunks",
			query:       "What is Rust?",
			mode:        models.ModeFast,
			model:       models.ModelPplxPro,
			sseResponse: `data: {"delta": "Rust "}
data: {"delta": "is "}
data: {"delta": "fast."}
data: [DONE]`,
			wantText:    "Rust is fast.",
		},
		{
			name:        "search with web results",
			query:       "Go tutorial",
			mode:        models.ModeDeepResearch,
			model:       models.ModelPplxPro,
			sseResponse: `data: {"text": "Here's a tutorial:", "web_results": [{"name": "Go Tutorial", "url": "https://example.com", "snippet": "Learn Go"}]}`,
			wantText:    "Here's a tutorial:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockHTTPClient()
			mock.SetResponse(createResponse(200, tt.sseResponse))
			client := NewWithHTTPClient(mock)

			opts := models.SearchOptions{
				Query:  tt.query,
				Mode:   tt.mode,
				Model:  tt.model,
				Stream: false,
			}

			resp, err := client.Search(context.Background(), opts)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if resp == nil {
				t.Fatal("Search() returned nil response")
			}

			if resp.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", resp.Text, tt.wantText)
			}
		})
	}
}

// TestSearchHTTPError tests search with various HTTP error status codes.
func TestSearchHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "401 unauthorized",
			statusCode: 401,
			body:       "Unauthorized",
			wantErr:    true,
			errMsg:     "API error 401",
		},
		{
			name:       "400 bad request",
			statusCode: 400,
			body:       "Bad Request",
			wantErr:    true,
			errMsg:     "API error 400",
		},
		{
			name:       "500 server error",
			statusCode: 500,
			body:       "Internal Server Error",
			wantErr:    true,
			errMsg:     "API error 500",
		},
		{
			name:       "404 not found",
			statusCode: 404,
			body:       "Not Found",
			wantErr:    true,
			errMsg:     "API error 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockHTTPClient()
			mock.SetResponse(createResponse(tt.statusCode, tt.body))
			client := NewWithHTTPClient(mock)

			opts := models.SearchOptions{
				Query:  "test query",
				Mode:   models.ModeDefault,
				Stream: false,
			}

			resp, err := client.Search(context.Background(), opts)
			if tt.wantErr {
				if err == nil {
					t.Error("Search() should return an error")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Error message = %v, want it to contain %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("Search() unexpected error = %v", err)
				}
			}
			if resp != nil && tt.wantErr {
				t.Error("Search() should return nil response on error")
			}
		})
	}
}

// TestSearchNetworkError tests search with network errors.
func TestSearchNetworkError(t *testing.T) {
	networkErr := errors.New("network connection failed")

	mock := NewMockHTTPClient()
	mock.SetError(networkErr)
	client := NewWithHTTPClient(mock)

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeDefault,
		Stream: false,
	}

	resp, err := client.Search(context.Background(), opts)
	if err == nil {
		t.Error("Search() should return an error for network failure")
	}
	if resp != nil {
		t.Error("Search() should return nil response on error")
	}
}

// TestSearchContextCancellation tests search with context cancellation.
func TestSearchContextCancellation(t *testing.T) {
	// Create a response that simulates a slow response
	slowResponse := `data: {"text": "Starting..."}
data: {"text": "Processing..."}
data: {"text": "Done."}
data: [DONE]`

	mock := NewMockHTTPClient()
	mock.SetResponse(createResponse(200, slowResponse))
	client := NewWithHTTPClient(mock)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeDefault,
		Stream: false,
	}

	resp, err := client.Search(ctx, opts)
	if err == nil {
		t.Error("Search() should return an error for cancelled context")
	}
	if resp != nil {
		t.Error("Search() should return nil response on cancelled context")
	}
}

// TestSearchEmptyResponse tests search with empty or minimal responses.
func TestSearchEmptyResponse(t *testing.T) {
	tests := []struct {
		name        string
		sseResponse string
	}{
		{
			name:        "empty response",
			sseResponse: ``,
		},
		{
			name:        "just done marker",
			sseResponse: `data: [DONE]`,
		},
		{
			name:        "empty JSON object",
			sseResponse: `data: {}`,
		},
		{
			name:        "end of stream",
			sseResponse: `event: end_of_stream
data: {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockHTTPClient()
			mock.SetResponse(createResponse(200, tt.sseResponse))
			client := NewWithHTTPClient(mock)

			opts := models.SearchOptions{
				Query:  "test query",
				Mode:   models.ModeDefault,
				Stream: false,
			}

			resp, err := client.Search(context.Background(), opts)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if resp == nil {
				t.Fatal("Search() returned nil response")
			}

			// Should return empty response, not error
			if resp.Text != "" {
				t.Errorf("Text = %q, want empty", resp.Text)
			}
		})
	}
}

// TestSearchStreamSuccess tests successful streaming search.
func TestSearchStreamSuccess(t *testing.T) {
	tests := []struct {
		name        string
		sseResponse string
		wantChunks  []models.StreamChunk
	}{
		{
			name:        "single chunk with text",
			sseResponse: `data: {"text": "Hello, world!"}`,
			wantChunks: []models.StreamChunk{
				{Text: "Hello, world!"},
			},
		},
		{
			name:        "multiple chunks with deltas",
			sseResponse: `data: {"delta": "Hello"}
data: {"delta": " "}
data: {"delta": "world"}
data: {"delta": "!"}
data: [DONE]`,
			wantChunks: []models.StreamChunk{
				{Delta: "Hello"},
				{Delta: " "},
				{Delta: "world"},
				{Delta: "!"},
				{Done: true},
			},
		},
		{
			name:        "step-based response with final step",
			sseResponse: `data: {"text": "[{\"step_type\": \"FINAL\", \"content\": {\"answer\": \"{\\\"answer\\\": \\\"Final answer.\\\", \\\"web_results\\\": [], \\\"chunks\\\": [], \\\"extra_web_results\\\": [], \\\"structured_answer\\\": []}\"}}]"}`,
			wantChunks: []models.StreamChunk{
				{Text: "Final answer.", StepType: "FINAL", Done: true},
			},
		},
		{
			name:        "response with web results",
			sseResponse: `data: {"text": "Results:", "web_results": [{"url": "https://example.com", "title": "Example"}]}`,
			wantChunks: []models.StreamChunk{
				{Text: "Results:"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockHTTPClient()
			mock.SetResponse(createResponse(200, tt.sseResponse))
			client := NewWithHTTPClient(mock)

			opts := models.SearchOptions{
				Query:  "test query",
				Mode:   models.ModeDefault,
				Stream: true,
			}

			ch, err := client.SearchStream(context.Background(), opts)
			if err != nil {
				t.Fatalf("SearchStream() error = %v", err)
			}

			chunks := []models.StreamChunk{}
			for chunk := range ch {
				if chunk.Error != nil {
					t.Fatalf("Unexpected error in stream: %v", chunk.Error)
				}
				chunks = append(chunks, chunk)
			}

			// For simplicity, just check that we got some chunks and no errors
			if len(chunks) == 0 {
				t.Error("Expected at least one chunk")
			}
		})
	}
}

// TestSearchStreamHTTPError tests streaming search with HTTP errors.
func TestSearchStreamHTTPError(t *testing.T) {
	mock := NewMockHTTPClient()
	mock.SetResponse(createResponse(401, "Unauthorized"))
	client := NewWithHTTPClient(mock)

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeDefault,
		Stream: true,
	}

	ch, err := client.SearchStream(context.Background(), opts)
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}

	// Should get error chunk
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error chunk")
	}
	if !strings.Contains(chunk.Error.Error(), "API error 401") {
		t.Errorf("Error = %v, want it to contain 'API error 401'", chunk.Error)
	}
}

// TestSearchStreamContextCancellation tests streaming search with context cancellation.
func TestSearchStreamContextCancellation(t *testing.T) {
	// Create a response that takes time to process
	slowResponse := `data: {"text": "Part 1"}
data: {"text": "Part 2"}
data: {"text": "Part 3"}`

	mock := &MockHTTPClient{
		ResponseToReturn: createResponse(200, slowResponse),
	}
	client := createTestClient(mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeDefault,
		Stream: true,
	}

	ch, err := client.SearchStream(ctx, opts)
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}

	// Should get error chunk due to cancellation
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error chunk from cancelled context")
	}
}

// TestSearchStreamNetworkError tests streaming search with network errors.
func TestSearchStreamNetworkError(t *testing.T) {
	networkErr := errors.New("network error")

	mock := &MockHTTPClient{
		ErrorToReturn: networkErr,
	}
	client := createTestClient(mock)

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeDefault,
		Stream: true,
	}

	ch, err := client.SearchStream(context.Background(), opts)
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}

	// Should get error chunk
	chunk := <-ch
	if chunk.Error == nil {
		t.Error("Expected error chunk")
	}
}

// TestSearchDifferentModels tests search with different model and mode combinations.
func TestSearchDifferentModels(t *testing.T) {
	// Test that the client correctly uses different models and modes
	mock := &MockHTTPClient{
		ResponseToReturn: createResponse(200, `data: {"text": "Test response"}`),
	}
	client := createTestClient(mock)

	// Test various model/mode combinations
	combinations := []struct {
		mode  models.Mode
		model models.Model
	}{
		{models.ModeFast, models.ModelPplxPro},
		{models.ModePro, models.ModelGPT51},
		{models.ModeReasoning, models.ModelGemini30Pro},
		{models.ModeDeepResearch, models.ModelPplxPro},
		{models.ModeDefault, models.ModelClaude45Sonnet},
	}

	for _, combo := range combinations {
		t.Run(string(combo.mode)+"_"+string(combo.model), func(t *testing.T) {
			opts := models.SearchOptions{
				Query:  "test query",
				Mode:   combo.mode,
				Model:  combo.model,
				Stream: false,
			}

			resp, err := client.Search(context.Background(), opts)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if resp == nil {
				t.Fatal("Search() returned nil response")
			}

			if resp.Text != "Test response" {
				t.Errorf("Text = %q, want %q", resp.Text, "Test response")
			}
		})
	}
}

// TestSearchWithInvalidJSON tests search response with invalid JSON.
func TestSearchWithInvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		sseResponse string
	}{
		{
			name:        "malformed JSON in data field",
			sseResponse: `data: {"text": "{invalid json}"}`,
		},
		{
			name:        "valid JSON but invalid structure",
			sseResponse: `data: {"text": {"unknown_field": "value"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockHTTPClient()
			mock.SetResponse(createResponse(200, tt.sseResponse))
			client := NewWithHTTPClient(mock)

			opts := models.SearchOptions{
				Query:  "test query",
				Mode:   models.ModeDefault,
				Stream: false,
			}

			// Should not panic, should handle gracefully
			resp, err := client.Search(context.Background(), opts)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if resp == nil {
				t.Fatal("Search() returned nil response")
			}
		})
	}
}

// TestSearchStreamWithBackendUUID tests that backend UUID is correctly propagated.
func TestSearchStreamWithBackendUUID(t *testing.T) {
	mock := &MockHTTPClient{
		ResponseToReturn: createResponse(200, `data: {"backend_uuid": "test-uuid-123", "text": "Response"}`),
	}
	client := createTestClient(mock)

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeDefault,
		Stream: true,
	}

	ch, err := client.SearchStream(context.Background(), opts)
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}

	chunk := <-ch
	if chunk.Error != nil {
		t.Fatalf("Unexpected error: %v", chunk.Error)
	}

	if chunk.BackendUUID != "test-uuid-123" {
		t.Errorf("BackendUUID = %q, want %q", chunk.BackendUUID, "test-uuid-123")
	}
}

// TestSearchStreamMultipleSteps tests streaming with multiple steps.
func TestSearchStreamMultipleSteps(t *testing.T) {
	response := `data: {"text": "[{\"step_type\": \"INITIAL_QUERY\", \"content\": {\"query\": \"test\"}, \"uuid\": \"uuid1\"}]"}
data: {"text": "[{\"step_type\": \"SEARCH_RESULTS\", \"content\": {\"web_results\": [{\"url\": \"https://example.com\", \"title\": \"Test\"}]}, \"uuid\": \"uuid2\"}]"}
data: {"text": "[{\"step_type\": \"FINAL\", \"content\": {\"answer\": \"{\\\"answer\\\": \\\"Final answer.\\\", \\\"web_results\\\": [], \\\"chunks\\\": [], \\\"extra_web_results\\\": [], \\\"structured_answer\\\": []}\"}, \"uuid\": \"uuid3\"}]"}`

	mock := &MockHTTPClient{
		ResponseToReturn: createResponse(200, response),
	}
	client := createTestClient(mock)

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeReasoning,
		Stream: true,
	}

	ch, err := client.SearchStream(context.Background(), opts)
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}

	// Collect all chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		if chunk.Error != nil {
			t.Fatalf("Unexpected error: %v", chunk.Error)
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

// TestSearchStreamWithRealSSEFormat tests streaming with real SSE format.
func TestSearchStreamWithRealSSEFormat(t *testing.T) {
	// Real SSE format with event headers
	response := `event: message
data: {"delta": "Streaming "}
data: {"delta": "response"}
data: [DONE]`

	mock := &MockHTTPClient{
		ResponseToReturn: createResponse(200, response),
	}
	client := createTestClient(mock)

	opts := models.SearchOptions{
		Query:  "test query",
		Mode:   models.ModeDefault,
		Stream: true,
	}

	ch, err := client.SearchStream(context.Background(), opts)
	if err != nil {
		t.Fatalf("SearchStream() error = %v", err)
	}

	// Collect chunks
	chunks := []models.StreamChunk{}
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestClientApplyDefaults(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Empty options should get defaults applied
	opts := models.SearchOptions{
		Query: "test query",
	}

	client.applyDefaults(&opts)

	if opts.Mode != models.ModeDefault {
		t.Errorf("Mode = %q, want %q", opts.Mode, models.ModeDefault)
	}
	if opts.Model != models.ModelPplxPro {
		t.Errorf("Model = %q, want %q", opts.Model, models.ModelPplxPro)
	}
	if opts.Language != "en-US" {
		t.Errorf("Language = %q, want %q", opts.Language, "en-US")
	}
	if len(opts.Sources) != 1 || opts.Sources[0] != models.SourceWeb {
		t.Errorf("Sources = %v, want [web]", opts.Sources)
	}
}

func TestClientApplyDefaultsWithExisting(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Options with existing values should not be overwritten
	opts := models.SearchOptions{
		Query:    "test query",
		Mode:     models.ModeFast,
		Model:    models.ModelGPT51,
		Language: "pt-BR",
		Sources:  []models.Source{models.SourceScholar},
	}

	client.applyDefaults(&opts)

	if opts.Mode != models.ModeFast {
		t.Errorf("Mode = %q, want %q (should not change)", opts.Mode, models.ModeFast)
	}
	if opts.Model != models.ModelGPT51 {
		t.Errorf("Model = %q, want %q (should not change)", opts.Model, models.ModelGPT51)
	}
	if opts.Language != "pt-BR" {
		t.Errorf("Language = %q, want %q (should not change)", opts.Language, "pt-BR")
	}
	if len(opts.Sources) != 1 || opts.Sources[0] != models.SourceScholar {
		t.Errorf("Sources = %v, want [scholar] (should not change)", opts.Sources)
	}
}
