// Package client provides HTTP client functionality with TLS fingerprint spoofing.
package client

import (
	"bytes"
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

// SetCookies sets cookies for the client.
func (c *HTTPClient) SetCookies(cookies []*http.Cookie) {
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

// buildHeaders returns common headers for Perplexity API requests.
func (c *HTTPClient) buildHeaders() http.Header {
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
	return headers
}

// Get performs a GET request.
func (c *HTTPClient) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = c.buildHeaders()
	return c.client.Do(req)
}

// Post performs a POST request with body.
func (c *HTTPClient) Post(path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest("POST", baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = c.buildHeaders()

	if body != nil {
		req.ContentLength = int64(len(body))
	}

	return c.client.Do(req)
}

// PostWithReader performs a POST request with io.ReadCloser body.
func (c *HTTPClient) PostWithReader(path string, body []byte) (*http.Response, error) {
	fullURL := baseURL + path

	req, err := http.NewRequest("POST", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = c.buildHeaders()

	return c.client.Do(req)
}

// GetCSRFToken extracts CSRF token from cookies.
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
func (c *HTTPClient) Close() error {
	// tls-client doesn't have explicit close
	return nil
}
