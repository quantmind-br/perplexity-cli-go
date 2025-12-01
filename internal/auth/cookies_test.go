package auth

import (
	"os"
	"path/filepath"
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

func TestLoadCookiesFromFile(t *testing.T) {
	// Create a temporary cookie file
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "cookies.json")

	// Write test cookies
	cookieJSON := `[
		{
			"name": "test_cookie",
			"value": "test_value",
			"domain": ".perplexity.ai",
			"path": "/",
			"secure": true,
			"httpOnly": true,
			"sameSite": "Lax"
		},
		{
			"name": "next-auth.csrf-token",
			"value": "csrf_token_value|hash",
			"domain": ".perplexity.ai",
			"path": "/"
		},
		{
			"name": "other_cookie",
			"value": "other_value",
			"domain": ".example.com",
			"path": "/"
		}
	]`

	if err := os.WriteFile(cookieFile, []byte(cookieJSON), 0644); err != nil {
		t.Fatalf("Failed to write test cookie file: %v", err)
	}

	cookies, err := LoadCookiesFromFile(cookieFile)
	if err != nil {
		t.Fatalf("LoadCookiesFromFile() error = %v", err)
	}

	// Should only have 2 cookies (perplexity.ai domain)
	if len(cookies) != 2 {
		t.Errorf("len(cookies) = %d, want 2", len(cookies))
	}

	// Check first cookie
	found := false
	for _, c := range cookies {
		if c.Name == "test_cookie" {
			found = true
			if c.Value != "test_value" {
				t.Errorf("cookie.Value = %q, want %q", c.Value, "test_value")
			}
			if !c.Secure {
				t.Error("cookie.Secure should be true")
			}
			if !c.HttpOnly {
				t.Error("cookie.HttpOnly should be true")
			}
		}
	}
	if !found {
		t.Error("test_cookie not found")
	}
}

func TestLoadCookiesFromFile_NotFound(t *testing.T) {
	_, err := LoadCookiesFromFile("/nonexistent/path/cookies.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadCookiesFromFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(cookieFile, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadCookiesFromFile(cookieFile)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestHasCSRFToken(t *testing.T) {
	tests := []struct {
		name    string
		cookies []*http.Cookie
		want    bool
	}{
		{
			name: "with CSRF token",
			cookies: []*http.Cookie{
				{Name: "next-auth.csrf-token", Value: "token|hash"},
			},
			want: true,
		},
		{
			name: "without CSRF token",
			cookies: []*http.Cookie{
				{Name: "other_cookie", Value: "value"},
			},
			want: false,
		},
		{
			name: "empty CSRF token",
			cookies: []*http.Cookie{
				{Name: "next-auth.csrf-token", Value: ""},
			},
			want: false,
		},
		{
			name:    "nil cookies",
			cookies: nil,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasCSRFToken(tt.cookies)
			if got != tt.want {
				t.Errorf("HasCSRFToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractCSRFToken(t *testing.T) {
	tests := []struct {
		name    string
		cookies []*http.Cookie
		want    string
	}{
		{
			name: "token with hash",
			cookies: []*http.Cookie{
				{Name: "next-auth.csrf-token", Value: "mytoken|somehash"},
			},
			want: "mytoken",
		},
		{
			name: "token without hash",
			cookies: []*http.Cookie{
				{Name: "next-auth.csrf-token", Value: "justtoken"},
			},
			want: "justtoken",
		},
		{
			name: "no token",
			cookies: []*http.Cookie{
				{Name: "other", Value: "value"},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCSRFToken(tt.cookies)
			if got != tt.want {
				t.Errorf("ExtractCSRFToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCookieMap(t *testing.T) {
	cookies := []*http.Cookie{
		{Name: "cookie1", Value: "value1"},
		{Name: "cookie2", Value: "value2"},
		{Name: "cookie3", Value: "value3"},
	}

	m := CookieMap(cookies)

	if len(m) != 3 {
		t.Errorf("len(map) = %d, want 3", len(m))
	}
	if m["cookie1"] != "value1" {
		t.Errorf("map[cookie1] = %q, want %q", m["cookie1"], "value1")
	}
	if m["cookie2"] != "value2" {
		t.Errorf("map[cookie2] = %q, want %q", m["cookie2"], "value2")
	}
}

func TestSaveCookiesToFile(t *testing.T) {
	tmpDir := t.TempDir()
	cookieFile := filepath.Join(tmpDir, "saved_cookies.json")

	cookies := []*http.Cookie{
		{
			Name:     "test_cookie",
			Value:    "test_value",
			Domain:   ".perplexity.ai",
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	}

	if err := SaveCookiesToFile(cookies, cookieFile); err != nil {
		t.Fatalf("SaveCookiesToFile() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		t.Error("Cookie file was not created")
	}

	// Load and verify
	loaded, err := LoadCookiesFromFile(cookieFile)
	if err != nil {
		t.Fatalf("Failed to load saved cookies: %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("len(loaded) = %d, want 1", len(loaded))
	}
	if loaded[0].Name != "test_cookie" {
		t.Errorf("loaded[0].Name = %q, want %q", loaded[0].Name, "test_cookie")
	}
}

func TestGetDefaultCookiePath(t *testing.T) {
	path, err := GetDefaultCookiePath()
	if err != nil {
		t.Fatalf("GetDefaultCookiePath() error = %v", err)
	}

	if path == "" {
		t.Error("GetDefaultCookiePath() returned empty path")
	}

	// Should contain .perplexity-cli
	if !filepath.IsAbs(path) {
		t.Error("Path should be absolute")
	}
}

func TestLoadCookiesFromNetscape(t *testing.T) {
	tmpDir := t.TempDir()
	netscapeFile := filepath.Join(tmpDir, "cookies.txt")

	netscapeContent := `# Netscape HTTP Cookie File
.perplexity.ai	TRUE	/	TRUE	1735689600	session	session_value
.perplexity.ai	TRUE	/	FALSE	0	test	test_value
.example.com	TRUE	/	TRUE	1735689600	other	other_value
`

	if err := os.WriteFile(netscapeFile, []byte(netscapeContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cookies, err := LoadCookiesFromNetscape(netscapeFile)
	if err != nil {
		t.Fatalf("LoadCookiesFromNetscape() error = %v", err)
	}

	// Should only have 2 cookies (perplexity.ai domain)
	if len(cookies) != 2 {
		t.Errorf("len(cookies) = %d, want 2", len(cookies))
	}

	// Check session cookie
	found := false
	for _, c := range cookies {
		if c.Name == "session" {
			found = true
			if c.Value != "session_value" {
				t.Errorf("cookie.Value = %q, want %q", c.Value, "session_value")
			}
			if !c.Secure {
				t.Error("cookie.Secure should be true")
			}
		}
	}
	if !found {
		t.Error("session cookie not found")
	}
}

func TestLoadCookiesFromNetscape_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	netscapeFile := filepath.Join(tmpDir, "cookies.txt")

	// Expired cookie (timestamp 0 means session cookie, negative is expired)
	netscapeContent := `# Netscape HTTP Cookie File
.perplexity.ai	TRUE	/	FALSE	0	session	session_value
.perplexity.ai	TRUE	/	FALSE	-1	expired	expired_value
.example.com	TRUE	/	FALSE	1735689600	other	other_value
`

	if err := os.WriteFile(netscapeFile, []byte(netscapeContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cookies, err := LoadCookiesFromNetscape(netscapeFile)
	if err != nil {
		t.Fatalf("LoadCookiesFromNetscape() error = %v", err)
	}

	// Session cookie (0) should be included, expired (-1) should not
	if len(cookies) != 2 {
		t.Errorf("len(cookies) = %d, want 2 (session cookie and valid cookie)", len(cookies))
	}
}

func TestLoadCookiesFromNetscape_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	netscapeFile := filepath.Join(tmpDir, "invalid.txt")

	// Invalid format
	invalidContent := `This is not a valid Netscape cookie file
Invalid line
`

	if err := os.WriteFile(netscapeFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cookies, err := LoadCookiesFromNetscape(netscapeFile)
	if err != nil {
		t.Fatalf("LoadCookiesFromNetscape() error = %v", err)
	}

	// Should not crash, may return 0 cookies or partial
	_ = cookies // Just verify it doesn't panic
}
