// Package client provides the main Perplexity API client.
package client

import (
	"context"
	"fmt"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/internal/auth"
	"github.com/diogo/perplexity-go/pkg/models"
)

// Client is the main Perplexity API client.
type Client struct {
	http          *HTTPClient
	cookies       []*http.Cookie
	csrfToken     string
	defaultModel  models.Model
	defaultMode   models.Mode
	defaultLang   string
	defaultSrcs   []models.Source
	proQueries    int
	fileUploads   int
	maxProQueries int
	maxFileUploads int
}

// Config holds client configuration options.
type Config struct {
	Cookies      []*http.Cookie
	CookieFile   string
	DefaultModel models.Model
	DefaultMode  models.Mode
	Language     string
	Sources      []models.Source
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		DefaultModel: models.ModelPplxPro,
		DefaultMode:  models.ModeDefault,
		Language:     "en-US",
		Sources:      []models.Source{models.SourceWeb},
	}
}

// New creates a new Perplexity client.
func New(cfg Config) (*Client, error) {
	httpClient, err := NewHTTPClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	client := &Client{
		http:           httpClient,
		defaultModel:   cfg.DefaultModel,
		defaultMode:    cfg.DefaultMode,
		defaultLang:    cfg.Language,
		defaultSrcs:    cfg.Sources,
		proQueries:     0,
		fileUploads:    0,
		maxProQueries:  5,
		maxFileUploads: 10,
	}

	// Load cookies from file if specified
	if cfg.CookieFile != "" {
		cookies, err := auth.LoadCookiesFromFile(cfg.CookieFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load cookies: %w", err)
		}
		client.SetCookies(cookies)
	} else if cfg.Cookies != nil {
		client.SetCookies(cfg.Cookies)
	}

	return client, nil
}

// NewWithCookieFile creates a client loading cookies from file.
func NewWithCookieFile(cookieFile string) (*Client, error) {
	cfg := DefaultConfig()
	cfg.CookieFile = cookieFile
	return New(cfg)
}

// NewWithCookies creates a client with provided cookies.
func NewWithCookies(cookies []*http.Cookie) (*Client, error) {
	cfg := DefaultConfig()
	cfg.Cookies = cookies
	return New(cfg)
}

// SetCookies sets the client cookies.
func (c *Client) SetCookies(cookies []*http.Cookie) {
	c.cookies = cookies
	c.http.SetCookies(cookies)
	c.csrfToken = auth.ExtractCSRFToken(cookies)
}

// GetCookies returns current cookies.
func (c *Client) GetCookies() []*http.Cookie {
	return c.http.GetCookies()
}

// HasValidSession checks if the client has valid authentication.
func (c *Client) HasValidSession() bool {
	return c.csrfToken != ""
}

// ProQueriesRemaining returns remaining pro queries.
func (c *Client) ProQueriesRemaining() int {
	return c.maxProQueries - c.proQueries
}

// FileUploadsRemaining returns remaining file uploads.
func (c *Client) FileUploadsRemaining() int {
	return c.maxFileUploads - c.fileUploads
}

// Search performs a search query.
func (c *Client) Search(ctx context.Context, opts models.SearchOptions) (*models.SearchResponse, error) {
	// Check if streaming is requested
	if opts.Stream {
		return c.searchStream(ctx, opts)
	}
	return c.searchNonStream(ctx, opts)
}

// SearchStream performs a streaming search query.
func (c *Client) SearchStream(ctx context.Context, opts models.SearchOptions) (<-chan models.StreamChunk, error) {
	opts.Stream = true
	return c.searchStreamChannel(ctx, opts)
}

// Close closes the client and releases resources.
func (c *Client) Close() error {
	return c.http.Close()
}

// SetDefaultModel sets the default model.
func (c *Client) SetDefaultModel(model models.Model) {
	c.defaultModel = model
}

// SetDefaultMode sets the default mode.
func (c *Client) SetDefaultMode(mode models.Mode) {
	c.defaultMode = mode
}

// SetDefaultLanguage sets the default language.
func (c *Client) SetDefaultLanguage(lang string) {
	c.defaultLang = lang
}

// SetDefaultSources sets the default sources.
func (c *Client) SetDefaultSources(sources []models.Source) {
	c.defaultSrcs = sources
}

// applyDefaults fills in missing options with defaults.
func (c *Client) applyDefaults(opts *models.SearchOptions) {
	if opts.Mode == "" {
		opts.Mode = c.defaultMode
	}
	if opts.Model == "" {
		opts.Model = c.defaultModel
	}
	if opts.Language == "" {
		opts.Language = c.defaultLang
	}
	if len(opts.Sources) == 0 {
		opts.Sources = c.defaultSrcs
	}
}
