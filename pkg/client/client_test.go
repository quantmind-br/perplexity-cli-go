package client

import (
	"testing"

	fhttp "github.com/bogdanfinn/fhttp"
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

	cookies := []*fhttp.Cookie{
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
	cookies := []*fhttp.Cookie{
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
