package models

import "testing"

func TestDefaultSearchOptions(t *testing.T) {
	query := "test query"
	opts := DefaultSearchOptions(query)

	if opts.Query != query {
		t.Errorf("Query = %q, want %q", opts.Query, query)
	}
	if opts.Mode != ModeDefault {
		t.Errorf("Mode = %q, want %q", opts.Mode, ModeDefault)
	}
	if opts.Model != ModelPplxPro {
		t.Errorf("Model = %q, want %q", opts.Model, ModelPplxPro)
	}
	if len(opts.Sources) != 1 || opts.Sources[0] != SourceWeb {
		t.Errorf("Sources = %v, want [web]", opts.Sources)
	}
	if opts.Language != "en-US" {
		t.Errorf("Language = %q, want %q", opts.Language, "en-US")
	}
	if opts.Incognito {
		t.Error("Incognito should be false by default")
	}
	if !opts.Stream {
		t.Error("Stream should be true by default")
	}
}

func TestSearchRequestJSON(t *testing.T) {
	req := SearchRequest{
		Version:  "2.18",
		Source:   "default",
		Language: "en-US",
		Mode:     "copilot",
		Query:    "test",
	}

	if req.Version != "2.18" {
		t.Errorf("Version = %q, want %q", req.Version, "2.18")
	}
	if req.Query != "test" {
		t.Errorf("Query = %q, want %q", req.Query, "test")
	}
}

func TestAttachment(t *testing.T) {
	att := Attachment{
		URL:         "https://example.com/file.pdf",
		ContentType: "application/pdf",
		Name:        "file.pdf",
		Size:        1024,
	}

	if att.URL != "https://example.com/file.pdf" {
		t.Errorf("URL = %q, want %q", att.URL, "https://example.com/file.pdf")
	}
	if att.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want %q", att.ContentType, "application/pdf")
	}
	if att.Size != 1024 {
		t.Errorf("Size = %d, want %d", att.Size, 1024)
	}
}

func TestFollowUpContext(t *testing.T) {
	ctx := FollowUpContext{
		BackendUUID: "test-uuid-123",
		Attachments: []Attachment{
			{URL: "https://example.com/file.pdf"},
		},
	}

	if ctx.BackendUUID != "test-uuid-123" {
		t.Errorf("BackendUUID = %q, want %q", ctx.BackendUUID, "test-uuid-123")
	}
	if len(ctx.Attachments) != 1 {
		t.Errorf("len(Attachments) = %d, want 1", len(ctx.Attachments))
	}
}

func TestUploadURLRequest(t *testing.T) {
	req := UploadURLRequest{
		Filename:    "test.pdf",
		ContentType: "application/pdf",
	}

	if req.Filename != "test.pdf" {
		t.Errorf("Filename = %q, want %q", req.Filename, "test.pdf")
	}
	if req.ContentType != "application/pdf" {
		t.Errorf("ContentType = %q, want %q", req.ContentType, "application/pdf")
	}
}
