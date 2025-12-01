// Package auth handles authentication and cookie management.
package auth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
)

// JSONCookie represents a cookie in JSON format (browser export).
type JSONCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expirationDate,omitempty"`
	Secure   bool    `json:"secure"`
	HTTPOnly bool    `json:"httpOnly"`
	SameSite string  `json:"sameSite,omitempty"`
}

// LoadCookiesFromFile loads cookies from a JSON file.
func LoadCookiesFromFile(path string) ([]*http.Cookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cookie file: %w", err)
	}

	var jsonCookies []JSONCookie
	if err := json.Unmarshal(data, &jsonCookies); err != nil {
		return nil, fmt.Errorf("failed to parse cookie JSON: %w", err)
	}

	cookies := make([]*http.Cookie, 0, len(jsonCookies))
	for _, jc := range jsonCookies {
		// Only include Perplexity cookies
		if !strings.Contains(jc.Domain, "perplexity.ai") {
			continue
		}

		cookie := &http.Cookie{
			Name:     jc.Name,
			Value:    jc.Value,
			Domain:   jc.Domain,
			Path:     jc.Path,
			Secure:   jc.Secure,
			HttpOnly: jc.HTTPOnly,
		}

		// Convert expiration timestamp
		if jc.Expires > 0 {
			cookie.Expires = time.Unix(int64(jc.Expires), 0)
		}

		// Parse SameSite
		switch strings.ToLower(jc.SameSite) {
		case "strict":
			cookie.SameSite = http.SameSiteStrictMode
		case "lax":
			cookie.SameSite = http.SameSiteLaxMode
		case "none":
			cookie.SameSite = http.SameSiteNoneMode
		default:
			cookie.SameSite = http.SameSiteDefaultMode
		}

		cookies = append(cookies, cookie)
	}

	return cookies, nil
}

// LoadCookiesFromNetscape loads cookies from Netscape format file.
func LoadCookiesFromNetscape(path string) ([]*http.Cookie, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie file: %w", err)
	}
	defer file.Close()

	var cookies []*http.Cookie
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Netscape format: domain, tailmatch, path, secure, expiration, name, value
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}

		domain := fields[0]
		// tailmatch := fields[1] == "TRUE"
		path := fields[2]
		secure := fields[3] == "TRUE"
		expiration := fields[4]
		name := fields[5]
		value := fields[6]

		// Only include Perplexity cookies
		if !strings.Contains(domain, "perplexity.ai") {
			continue
		}

		cookie := &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   path,
			Secure: secure,
		}

		// Parse expiration
		if expiration != "0" {
			var exp int64
			fmt.Sscanf(expiration, "%d", &exp)
			if exp > 0 {
				cookie.Expires = time.Unix(exp, 0)
			}
		}

		cookies = append(cookies, cookie)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading cookie file: %w", err)
	}

	return cookies, nil
}

// GetDefaultCookiePath returns the default cookie file path.
func GetDefaultCookiePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".perplexity-cli", "cookies.json"), nil
}

// SaveCookiesToFile saves cookies to a JSON file.
func SaveCookiesToFile(cookies []*http.Cookie, path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	jsonCookies := make([]JSONCookie, 0, len(cookies))
	for _, c := range cookies {
		jc := JSONCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
		}

		if !c.Expires.IsZero() {
			jc.Expires = float64(c.Expires.Unix())
		}

		switch c.SameSite {
		case http.SameSiteStrictMode:
			jc.SameSite = "Strict"
		case http.SameSiteLaxMode:
			jc.SameSite = "Lax"
		case http.SameSiteNoneMode:
			jc.SameSite = "None"
		}

		jsonCookies = append(jsonCookies, jc)
	}

	data, err := json.MarshalIndent(jsonCookies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write cookie file: %w", err)
	}

	return nil
}

// HasCSRFToken checks if cookies contain a valid CSRF token.
func HasCSRFToken(cookies []*http.Cookie) bool {
	for _, c := range cookies {
		if c.Name == "next-auth.csrf-token" && c.Value != "" {
			return true
		}
	}
	return false
}

// ExtractCSRFToken extracts the CSRF token value from cookies.
func ExtractCSRFToken(cookies []*http.Cookie) string {
	for _, c := range cookies {
		if c.Name == "next-auth.csrf-token" {
			// Token format: "value|hash"
			value := c.Value
			if idx := strings.Index(value, "|"); idx != -1 {
				return value[:idx]
			}
			return value
		}
	}
	return ""
}

// CookieMap converts cookie slice to map for easier access.
func CookieMap(cookies []*http.Cookie) map[string]string {
	m := make(map[string]string, len(cookies))
	for _, c := range cookies {
		m[c.Name] = c.Value
	}
	return m
}
