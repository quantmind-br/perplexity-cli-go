package client

import (
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

func TestNewHTTPClient(t *testing.T) {
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	if client == nil {
		t.Fatal("NewHTTPClient() returned nil")
	}

	if client.client == nil {
		t.Error("client.client should not be nil")
	}

	if client.cookies == nil {
		t.Error("client.cookies should be initialized")
	}
}

func TestHTTPClientSetCookies(t *testing.T) {
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	cookies := []*http.Cookie{
		{Name: "test", Value: "value", Domain: ".perplexity.ai"},
	}

	client.SetCookies(cookies)

	if len(client.cookies) != 1 {
		t.Errorf("len(cookies) = %d, want 1", len(client.cookies))
	}
}

func TestHTTPClientAddCookie(t *testing.T) {
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	client.AddCookie(&http.Cookie{Name: "cookie1", Value: "value1", Domain: ".perplexity.ai"})
	client.AddCookie(&http.Cookie{Name: "cookie2", Value: "value2", Domain: ".perplexity.ai"})

	if len(client.cookies) != 2 {
		t.Errorf("len(cookies) = %d, want 2", len(client.cookies))
	}
}

func TestHTTPClientBuildHeaders(t *testing.T) {
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	headers := client.buildHeaders()

	// Check required headers (using direct access since Get() canonicalizes)
	if v := headers["Accept"]; len(v) == 0 || v[0] != "*/*" {
		t.Errorf("Header Accept = %v, want [*/*]", v)
	}
	if v := headers["Accept-Language"]; len(v) == 0 || v[0] != "en-US,en;q=0.9" {
		t.Errorf("Header Accept-Language = %v, want [en-US,en;q=0.9]", v)
	}
	if v := headers["Content-Type"]; len(v) == 0 || v[0] != "application/json" {
		t.Errorf("Header Content-Type = %v, want [application/json]", v)
	}
	if v := headers["Origin"]; len(v) == 0 || v[0] != baseURL {
		t.Errorf("Header Origin = %v, want [%s]", v, baseURL)
	}
	if v := headers["User-Agent"]; len(v) == 0 || v[0] != userAgent {
		t.Errorf("Header User-Agent = %v, want [%s]", v, userAgent)
	}
	// Check sec-* headers
	if v := headers["sec-ch-ua-mobile"]; len(v) == 0 || v[0] != "?0" {
		t.Errorf("Header sec-ch-ua-mobile = %v, want [?0]", v)
	}
	if v := headers["sec-fetch-dest"]; len(v) == 0 || v[0] != "empty" {
		t.Errorf("Header sec-fetch-dest = %v, want [empty]", v)
	}
}

func TestHTTPClientGetCSRFToken(t *testing.T) {
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	// No cookies, no token
	token := client.GetCSRFToken()
	if token != "" {
		t.Errorf("GetCSRFToken() = %q, want empty", token)
	}

	// Add CSRF cookie with hash
	client.SetCookies([]*http.Cookie{
		{Name: "next-auth.csrf-token", Value: "mytoken|somehash", Domain: ".perplexity.ai", Path: "/"},
	})

	token = client.GetCSRFToken()
	if token != "mytoken" {
		t.Errorf("GetCSRFToken() = %q, want %q", token, "mytoken")
	}
}

func TestHTTPClientGetCSRFTokenNoHash(t *testing.T) {
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	// Add CSRF cookie without hash
	client.SetCookies([]*http.Cookie{
		{Name: "next-auth.csrf-token", Value: "justtoken", Domain: ".perplexity.ai", Path: "/"},
	})

	token := client.GetCSRFToken()
	if token != "justtoken" {
		t.Errorf("GetCSRFToken() = %q, want %q", token, "justtoken")
	}
}

func TestHTTPClientClose(t *testing.T) {
	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient() error = %v", err)
	}

	// Close should not error
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestConstants(t *testing.T) {
	if baseURL != "https://www.perplexity.ai" {
		t.Errorf("baseURL = %q, want %q", baseURL, "https://www.perplexity.ai")
	}
	if searchPath != "/rest/sse/perplexity_ask" {
		t.Errorf("searchPath = %q, want %q", searchPath, "/rest/sse/perplexity_ask")
	}
	if sessionPath != "/api/auth/session" {
		t.Errorf("sessionPath = %q, want %q", sessionPath, "/api/auth/session")
	}
	if uploadPath != "/rest/uploads/create_upload_url" {
		t.Errorf("uploadPath = %q, want %q", uploadPath, "/rest/uploads/create_upload_url")
	}
	if userAgent == "" {
		t.Error("userAgent should not be empty")
	}
}
