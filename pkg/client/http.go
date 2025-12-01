// Package client provides HTTP client functionality with TLS fingerprint spoofing.
package client

import (
	"fmt"
	"io"
	"net/url"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const (
	baseURL      = "https://www.perplexity.ai"
	searchPath   = "/rest/sse/perplexity_ask"
	sessionPath  = "/api/auth/session"
	uploadPath   = "/rest/uploads/create_upload_url"
	userAgent    = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// HTTPClientInterface defines the contract for HTTP client operations.
// This interface enables dependency injection and mocking for testing.
// The interface is designed to be test-friendly while maintaining backward compatibility.
type HTTPClientInterface interface {
	// Get performs a GET request to the given URL.
	// The URL can be a full URL or a path (will be prefixed with baseURL).
	Get(url string, headers map[string]string) (*http.Response, error)

	// Post performs a POST request with the given body.
	// The URL can be a full URL or a path (will be prefixed with baseURL).
	Post(url string, body io.Reader, headers map[string]string) (*http.Response, error)

	// PostWithReader performs a POST request with a reader and content type.
	// The URL can be a full URL or a path (will be prefixed with baseURL).
	PostWithReader(url string, reader io.Reader, contentType string, headers map[string]string) (*http.Response, error)

	// SetCookies sets cookies for the client using a map of name-value pairs.
	// This signature uses map[string]string to enable easy mocking in tests.
	SetCookies(cookies map[string]string)

	// GetCSRFToken retrieves the CSRF token from cookies.
	GetCSRFToken() string

	// GetCookies returns current cookies.
	GetCookies() []*http.Cookie

	// Close closes the HTTP client and releases resources.
	Close() error
}

// HTTPClient wraps tls-client to provide Chrome-impersonating HTTP requests.
type HTTPClient struct {
	client  tls_client.HttpClient
	cookies []*http.Cookie
}

// NewHTTPClient creates a new HTTP client with Chrome TLS fingerprint.
func NewHTTPClient() (*HTTPClient, error) {
	jar := tls_client.NewCookieJar()

	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(jar),
		tls_client.WithRandomTLSExtensionOrder(),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS client: %w", err)
	}

	return &HTTPClient{
		client:  client,
		cookies: make([]*http.Cookie, 0),
	}, nil
}

// cookiesMapToSlice converts a map of cookies to a slice of http.Cookie.
func cookiesMapToSlice(cookies map[string]string) []*http.Cookie {
	cookieSlice := make([]*http.Cookie, 0, len(cookies))
	for name, value := range cookies {
		cookieSlice = append(cookieSlice, &http.Cookie{
			Name:  name,
			Value: value,
		})
	}
	return cookieSlice
}

// cookiesSliceToMap converts a slice of http.Cookie to a map.
func cookiesSliceToMap(cookies []*http.Cookie) map[string]string {
	cookieMap := make(map[string]string, len(cookies))
	for _, cookie := range cookies {
		cookieMap[cookie.Name] = cookie.Value
	}
	return cookieMap
}

// buildHeaders returns common headers for Perplexity API requests.
// It merges custom headers with default headers.
func (c *HTTPClient) buildHeaders(customHeaders map[string]string) http.Header {
	headers := http.Header{
		"Accept":             {"*/*"},
		"Accept-Encoding":    {"gzip, deflate, br, zstd"},
		"Accept-Language":    {"en-US,en;q=0.9"},
		"Content-Type":       {"application/json"},
		"Origin":             {baseURL},
		"Referer":            {baseURL + "/"},
		"User-Agent":         {userAgent},
		"sec-ch-ua":          {`"Chromium";v="133", "Not(A:Brand";v="99", "Google Chrome";v="133"`},
		"sec-ch-ua-mobile":   {"?0"},
		"sec-ch-ua-platform": {`"Linux"`},
		"sec-fetch-dest":     {"empty"},
		"sec-fetch-mode":     {"cors"},
		"sec-fetch-site":     {"same-origin"},
		"priority":           {"u=1, i"},
	}

	// Merge custom headers
	for key, value := range customHeaders {
		headers.Set(key, value)
	}

	return headers
}

// normalizeURL converts a path to a full URL if needed.
func (c *HTTPClient) normalizeURL(urlStr string) string {
	// If it's already a full URL (starts with http:// or https://), return as-is
	if len(urlStr) > 7 && (urlStr[:7] == "http://" || urlStr[:8] == "https://") {
		return urlStr
	}
	// Otherwise, prepend baseURL
	return baseURL + urlStr
}

// Get performs a GET request.
// Implements HTTPClientInterface.
func (c *HTTPClient) Get(urlStr string, headers map[string]string) (*http.Response, error) {
	fullURL := c.normalizeURL(urlStr)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = c.buildHeaders(headers)
	return c.client.Do(req)
}

// Post performs a POST request with body.
// Implements HTTPClientInterface.
func (c *HTTPClient) Post(urlStr string, body io.Reader, headers map[string]string) (*http.Response, error) {
	fullURL := c.normalizeURL(urlStr)
	req, err := http.NewRequest("POST", fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = c.buildHeaders(headers)
	return c.client.Do(req)
}

// PostWithReader performs a POST request with a reader and content type.
// Implements HTTPClientInterface.
func (c *HTTPClient) PostWithReader(urlStr string, reader io.Reader, contentType string, headers map[string]string) (*http.Response, error) {
	// Create a new reader that wraps the provided reader with proper content type
	headersWithCT := make(map[string]string)
	for k, v := range headers {
		headersWithCT[k] = v
	}
	headersWithCT["Content-Type"] = contentType

	return c.Post(urlStr, reader, headersWithCT)
}

// SetCookies sets cookies for the client.
// Implements HTTPClientInterface.
func (c *HTTPClient) SetCookies(cookies map[string]string) {
	c.cookies = cookiesMapToSlice(cookies)
	u, _ := url.Parse(baseURL)
	c.client.SetCookies(u, c.cookies)
}

// GetCSRFToken extracts CSRF token from cookies.
// Implements HTTPClientInterface.
func (c *HTTPClient) GetCSRFToken() string {
	for _, cookie := range c.GetCookies() {
		if cookie.Name == "next-auth.csrf-token" {
			// Token format: "value|hash"
			value := cookie.Value
			for i, ch := range value {
				if ch == '|' {
					return value[:i]
				}
			}
			return value
		}
	}
	return ""
}

// Close closes the HTTP client.
// Implements HTTPClientInterface.
func (c *HTTPClient) Close() error {
	// tls-client doesn't have explicit close
	return nil
}

// SetCookiesLegacy sets cookies for the client using []*http.Cookie.
// This is kept for backward compatibility with existing code.
func (c *HTTPClient) SetCookiesLegacy(cookies []*http.Cookie) {
	c.cookies = cookies
	u, _ := url.Parse(baseURL)
	c.client.SetCookies(u, cookies)
}

// AddCookie adds a single cookie.
func (c *HTTPClient) AddCookie(cookie *http.Cookie) {
	c.cookies = append(c.cookies, cookie)
	u, _ := url.Parse(baseURL)
	c.client.SetCookies(u, c.cookies)
}

// GetCookies returns current cookies.
func (c *HTTPClient) GetCookies() []*http.Cookie {
	u, _ := url.Parse(baseURL)
	return c.client.GetCookies(u)
}

// MockHTTPClient is a mock implementation of HTTPClientInterface for testing.
// It allows tests to simulate HTTP responses without making real network calls.
type MockHTTPClient struct {
	// Responses to return (indexed by call number)
	Responses []*http.Response
	Errors    []error

	// Cookies state
	Cookies map[string]string

	// Configuration
	defaultResponse *http.Response
	defaultError    error

	// Request tracking
	LastRequestURL   string
	LastRequestBody  []byte
	RequestCount     int
}

// NewMockHTTPClient creates a new MockHTTPClient with default settings.
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		Cookies:         make(map[string]string),
		defaultResponse: nil,
		defaultError:    nil,
	}
}

// SetResponse sets the default response for all future calls.
func (m *MockHTTPClient) SetResponse(resp *http.Response) {
	m.defaultResponse = resp
}

// SetError sets the default error for all future calls.
func (m *MockHTTPClient) SetError(err error) {
	m.defaultError = err
}

// Get simulates a GET request for testing.
// Implements HTTPClientInterface.
func (m *MockHTTPClient) Get(url string, headers map[string]string) (*http.Response, error) {
	m.RequestCount++
	m.LastRequestURL = url
	return m.defaultResponse, m.defaultError
}

// Post simulates a POST request for testing.
// Implements HTTPClientInterface.
func (m *MockHTTPClient) Post(url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	m.RequestCount++
	m.LastRequestURL = url
	if body != nil {
		m.LastRequestBody, _ = io.ReadAll(body)
	}
	return m.defaultResponse, m.defaultError
}

// PostWithReader simulates a POST request with reader for testing.
// Implements HTTPClientInterface.
func (m *MockHTTPClient) PostWithReader(url string, reader io.Reader, contentType string, headers map[string]string) (*http.Response, error) {
	return m.Post(url, reader, headers)
}

// SetCookies sets cookies on the mock client for testing.
// Implements HTTPClientInterface.
func (m *MockHTTPClient) SetCookies(cookies map[string]string) {
	m.Cookies = cookies
}

// GetCSRFToken returns CSRF token from cookies for testing.
// Implements HTTPClientInterface.
func (m *MockHTTPClient) GetCSRFToken() string {
	if m.Cookies == nil {
		return ""
	}
	cookieName := "next-auth.csrf-token"
	token, ok := m.Cookies[cookieName]
	if !ok {
		return ""
	}
	return token
}

// GetCookies returns cookies as http.Cookie slice for testing.
// Implements HTTPClientInterface.
func (m *MockHTTPClient) GetCookies() []*http.Cookie {
	cookies := make([]*http.Cookie, 0, len(m.Cookies))
	for name, value := range m.Cookies {
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
	}
	return cookies
}

// Close closes the mock client for testing.
// Implements HTTPClientInterface.
func (m *MockHTTPClient) Close() error {
	return nil
}

// Ensure HTTPClient implements the interface
var _ HTTPClientInterface = &HTTPClient{}

// Ensure MockHTTPClient implements the interface
var _ HTTPClientInterface = &MockHTTPClient{}
