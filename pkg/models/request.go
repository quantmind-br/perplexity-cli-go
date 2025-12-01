package models

// SearchRequest represents the payload for a Perplexity search query.
type SearchRequest struct {
	Version              string        `json:"version"`
	Source               string        `json:"source"`
	FrontendSessionID    string        `json:"frontend_session_id,omitempty"`
	Language             string        `json:"language"`
	Timezone             string        `json:"timezone"`
	SearchFocus          string        `json:"search_focus"`
	FrontendUUID         string        `json:"frontend_uuid,omitempty"`
	GptUUID              string        `json:"gpt_uuid,omitempty"`
	ConversationMode     string        `json:"conversation_mode,omitempty"`
	Mode                 string        `json:"mode"`
	IsIncognito          bool          `json:"is_incognito"`
	InPage               string        `json:"in_page,omitempty"`
	Query                string        `json:"query_str"`
	ModelPreference      *string       `json:"model_preference,omitempty"`
	IsProReasoningMode   bool          `json:"is_pro_reasoning_mode,omitempty"`
	Sources              []string      `json:"sources,omitempty"`
	Attachments          []Attachment  `json:"attachments,omitempty"`
	BackendUUID          string        `json:"backend_uuid,omitempty"`
	ReadWriteToken       *string       `json:"read_write_token,omitempty"`
	FunctioningMode      string        `json:"functioning_mode,omitempty"`
	UseInhouseModel      bool          `json:"use_inhouse_model,omitempty"`
}

// Attachment represents a file attachment in the request.
type Attachment struct {
	URL          string `json:"url"`
	ContentType  string `json:"content_type,omitempty"`
	Name         string `json:"name,omitempty"`
	Size         int64  `json:"size,omitempty"`
}

// SearchOptions contains user-configurable search parameters.
type SearchOptions struct {
	Query       string
	Mode        Mode
	Model       Model
	Sources     []Source
	Language    string
	Incognito   bool
	Stream      bool
	Attachments []string
	FollowUp    *FollowUpContext
}

// FollowUpContext contains context for follow-up queries.
type FollowUpContext struct {
	BackendUUID  string
	Attachments  []Attachment
}

// DefaultSearchOptions returns options with sensible defaults.
func DefaultSearchOptions(query string) SearchOptions {
	return SearchOptions{
		Query:     query,
		Mode:      ModeDefault,
		Model:     ModelPplxPro,
		Sources:   []Source{SourceWeb},
		Language:  "en-US",
		Incognito: false,
		Stream:    true,
	}
}

// UploadURLRequest represents a request to get an upload URL.
type UploadURLRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

// UploadURLResponse contains the S3 upload URL and fields.
type UploadURLResponse struct {
	URL        string            `json:"url"`
	Fields     map[string]string `json:"fields"`
	FinalURL   string            `json:"final_url,omitempty"`
}
