package models

import "time"

// SearchResponse represents a complete response from Perplexity.
type SearchResponse struct {
	Text         string          `json:"text"`
	BackendUUID  string          `json:"backend_uuid,omitempty"`
	Blocks       []ResponseBlock `json:"blocks,omitempty"`
	Attachments  []Attachment    `json:"attachments,omitempty"`
	FinishReason string          `json:"finish_reason,omitempty"`
	WebResults   []WebResult     `json:"web_results,omitempty"`
}

// SSEStep represents a step in the SSE stream response.
type SSEStep struct {
	StepType string      `json:"step_type"`
	Content  interface{} `json:"content"`
	UUID     string      `json:"uuid,omitempty"`
}

// FinalContent represents the content of a FINAL step.
type FinalContent struct {
	Answer string `json:"answer"`
}

// FinalAnswer represents the parsed answer JSON from a FINAL step.
type FinalAnswer struct {
	Answer           string      `json:"answer"`
	WebResults       []WebResult `json:"web_results,omitempty"`
	Chunks           []string    `json:"chunks,omitempty"`
	ExtraWebResults  []WebResult `json:"extra_web_results,omitempty"`
	StructuredAnswer []string    `json:"structured_answer,omitempty"`
}

// WebResult represents a web search result from the API.
type WebResult struct {
	Name      string                 `json:"name,omitempty"`
	URL       string                 `json:"url"`
	Snippet   string                 `json:"snippet,omitempty"`
	Title     string                 `json:"title,omitempty"`
	MetaData  map[string]interface{} `json:"meta_data,omitempty"`
	SiteLinks []interface{}          `json:"sitelinks,omitempty"`
}

// SearchResultsContent represents SEARCH_RESULTS step content.
type SearchResultsContent struct {
	GoalID     string      `json:"goal_id,omitempty"`
	WebResults []WebResult `json:"web_results,omitempty"`
}

// ResponseBlock represents a block in the response.
type ResponseBlock struct {
	MarkdownBlock      *MarkdownBlock      `json:"markdown_block,omitempty"`
	PlanBlock          *PlanBlock          `json:"plan_block,omitempty"`
	ReasoningPlanBlock *ReasoningPlanBlock `json:"reasoning_plan_block,omitempty"`
	WebSearchResults   *WebSearchResults   `json:"web_search_results,omitempty"`
}

// MarkdownBlock contains the main answer with citations.
type MarkdownBlock struct {
	Answer    string     `json:"answer"`
	Citations []Citation `json:"citations,omitempty"`
}

// Citation represents a source citation.
type Citation struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
	SiteName    string `json:"site_name,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

// PlanBlock represents an execution plan (mode='default').
type PlanBlock struct {
	Steps []PlanStep `json:"steps,omitempty"`
}

// PlanStep represents a step in the execution plan.
type PlanStep struct {
	Description string `json:"description"`
	Status      string `json:"status,omitempty"`
}

// ReasoningPlanBlock represents step-by-step reasoning (deep-research).
type ReasoningPlanBlock struct {
	Reasoning []ReasoningStep `json:"reasoning,omitempty"`
}

// ReasoningStep represents a reasoning step.
type ReasoningStep struct {
	Thought string `json:"thought"`
	Action  string `json:"action,omitempty"`
	Result  string `json:"result,omitempty"`
}

// WebSearchResults contains search results from various sources.
type WebSearchResults struct {
	Results []WebSearchResult `json:"results,omitempty"`
}

// WebSearchResult represents a single search result.
type WebSearchResult struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Snippet     string `json:"snippet,omitempty"`
	Favicon     string `json:"favicon,omitempty"`
	Source      string `json:"source,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
}

// StreamChunk represents a chunk of streaming response.
type StreamChunk struct {
	Text        string          `json:"text,omitempty"`
	Delta       string          `json:"delta,omitempty"`
	BackendUUID string          `json:"backend_uuid,omitempty"`
	Blocks      []ResponseBlock `json:"blocks,omitempty"`
	Done        bool            `json:"done,omitempty"`
	Error       error           `json:"-"`
	// New step-based fields
	StepType   string      `json:"step_type,omitempty"`
	WebResults []WebResult `json:"web_results,omitempty"`
	Chunks     []string    `json:"chunks,omitempty"`
}

// HistoryEntry represents a query in the history file.
type HistoryEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Query       string    `json:"query"`
	Mode        string    `json:"mode"`
	Model       string    `json:"model,omitempty"`
	Response    string    `json:"response,omitempty"`
	BackendUUID string    `json:"backend_uuid,omitempty"`
}
