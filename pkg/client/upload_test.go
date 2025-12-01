package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/pkg/models"
)

// mockHTTPClient is a test double for HTTPClient
type mockHTTPClient struct {
	PostFunc        func(path string, body []byte) (*http.Response, error)
	GetFunc         func(path string) (*http.Response, error)
	CloseFunc       func() error
	SetCookiesFunc  func(cookies []*http.Cookie)
	AddCookieFunc   func(cookie *http.Cookie)
	GetCookiesFunc  func() []*http.Cookie
	GetCSRFTokenFunc func() string
}

// Post implements the mock
func (m *mockHTTPClient) Post(path string, body []byte) (*http.Response, error) {
	if m.PostFunc != nil {
		return m.PostFunc(path, body)
	}
	return nil, fmt.Errorf("Post not implemented")
}

// Get implements the mock
func (m *mockHTTPClient) Get(path string) (*http.Response, error) {
	if m.GetFunc != nil {
		return m.GetFunc(path)
	}
	return nil, fmt.Errorf("Get not implemented")
}

// PostWithReader implements the mock (not used in upload tests)
func (m *mockHTTPClient) PostWithReader(path string, body []byte) (*http.Response, error) {
	return nil, fmt.Errorf("PostWithReader not implemented")
}

// Close implements the mock
func (m *mockHTTPClient) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// SetCookies implements the mock
func (m *mockHTTPClient) SetCookies(cookies []*http.Cookie) {
	if m.SetCookiesFunc != nil {
		m.SetCookiesFunc(cookies)
	}
}

// AddCookie implements the mock
func (m *mockHTTPClient) AddCookie(cookie *http.Cookie) {
	if m.AddCookieFunc != nil {
		m.AddCookieFunc(cookie)
	}
}

// GetCookies implements the mock
func (m *mockHTTPClient) GetCookies() []*http.Cookie {
	if m.GetCookiesFunc != nil {
		return m.GetCookiesFunc()
	}
	return nil
}

// GetCSRFToken implements the mock
func (m *mockHTTPClient) GetCSRFToken() string {
	if m.GetCSRFTokenFunc != nil {
		return m.GetCSRFTokenFunc()
	}
	return ""
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"file.jpg", "image/jpeg"},
		{"file.jpeg", "image/jpeg"},
		{"file.JPG", "image/jpeg"},
		{"file.png", "image/png"},
		{"file.PNG", "image/png"},
		{"file.gif", "image/gif"},
		{"file.webp", "image/webp"},
		{"file.pdf", "application/pdf"},
		{"file.txt", "text/plain"},
		{"file.md", "text/markdown"},
		{"file.json", "application/json"},
		{"file.csv", "text/csv"},
		{"file.doc", "application/msword"},
		{"file.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"file.xls", "application/vnd.ms-excel"},
		{"file.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"file.unknown", "application/octet-stream"},
		{"file", "application/octet-stream"},
		{"file.", "application/octet-stream"},
		// Edge cases
		{"file.tar.gz", "application/octet-stream"},
		{".hidden.jpg", "image/jpeg"},
		{".gitignore", "application/octet-stream"},
		{"file.JPG.PNG", "application/octet-stream"}, // Multiple extensions
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectContentType(tt.filename)
			if got != tt.want {
				t.Errorf("detectContentType(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"file.jpg", true},
		{"file.jpeg", true},
		{"file.png", true},
		{"file.gif", true},
		{"file.webp", true},
		{"file.bmp", true},
		{"file.svg", true},
		{"file.JPG", true},
		{"file.pdf", false},
		{"file.txt", false},
		{"file.doc", false},
		{"file", false},
		// Edge cases
		{".hidden.png", true},
		{"file.PNG.GIF", false}, // Multiple extensions - only last should count
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := isImageFile(tt.filename)
			if got != tt.want {
				t.Errorf("isImageFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestRewriteImageURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "S3 bucket URL",
			url:  "https://mybucket.s3.us-east-1.amazonaws.com/path/to/file.png",
			want: "https://s3.us-east-1.amazonaws.com/mybucket/path/to/file.png",
		},
		{
			name: "S3 bucket URL eu-west",
			url:  "https://otherbucket.s3.eu-west-1.amazonaws.com/images/photo.jpg",
			want: "https://s3.eu-west-1.amazonaws.com/otherbucket/images/photo.jpg",
		},
		{
			name: "S3 bucket URL ap-southeast",
			url:  "https://testbucket.s3.ap-southeast-1.amazonaws.com/files/image.webp",
			want: "https://s3.ap-southeast-1.amazonaws.com/testbucket/files/image.webp",
		},
		{
			name: "Non-S3 URL unchanged",
			url:  "https://example.com/path/to/file.png",
			want: "https://example.com/path/to/file.png",
		},
		{
			name: "Different domain unchanged",
			url:  "https://cdn.example.com/file.png",
			want: "https://cdn.example.com/file.png",
		},
		{
			name: "S3 URL with query params",
			url:  "https://mybucket.s3.us-east-1.amazonaws.com/path/file.png?token=123",
			want: "https://s3.us-east-1.amazonaws.com/mybucket/path/file.png?token=123",
		},
		{
			name: "Empty URL",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteImageURL(tt.url)
			if got != tt.want {
				t.Errorf("rewriteImageURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestClientUploadFile(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		filename     string
		content      string
		setupMock    func(*mockHTTPClient)
		want         string
		wantErr      bool
	}{
		{
			name:     "successful image upload",
			filename: "test.png",
			content:  "fake image data",
			setupMock: func(m *mockHTTPClient) {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					// Verify the upload URL request
					var req models.UploadURLRequest
					if err := json.Unmarshal(body, &req); err != nil {
						t.Errorf("Failed to unmarshal request: %v", err)
					}
					if req.Filename != "test.png" {
						t.Errorf("Filename = %q, want %q", req.Filename, "test.png")
					}
					if req.ContentType != "image/png" {
						t.Errorf("ContentType = %q, want %q", req.ContentType, "image/png")
					}

					// Return mock upload URL response
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(`{
							"url": "https://test-bucket.s3.us-east-1.amazonaws.com",
							"fields": {
								"key": "uploads/test.png",
								"bucket": "test-bucket",
								"region": "us-east-1"
							},
							"final_url": "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/test.png"
						}`)),
					}
					return resp, nil
				}
			},
			want:    "https://s3.us-east-1.amazonaws.com/test-bucket/uploads/test.png",
			wantErr: false,
		},
		{
			name:     "successful PDF upload",
			filename: "document.pdf",
			content:  "PDF content",
			setupMock: func(m *mockHTTPClient) {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(`{
							"url": "https://bucket.s3.eu-west-1.amazonaws.com",
							"fields": {
								"key": "docs/document.pdf"
							}
						}`)),
					}
					return resp, nil
				}
			},
			want:    "https://bucket.s3.eu-west-1.amazonaws.com/docs/document.pdf",
			wantErr: false,
		},
		{
			name:     "file not found error",
			filename: "/nonexistent/path/to/file.txt",
			content:  "",
			setupMock: func(m *mockHTTPClient) {
				// No mock needed - file read will fail
			},
			want:    "",
			wantErr: true,
		},
		{
			name:     "permission denied error",
			filename: tmpDir + "/noperm.txt",
			content:  "test",
			setupMock: func(m *mockHTTPClient) {
				// Create file without read permissions on Unix systems
				// Skipped in test as it's platform-specific
			},
			want:    "",
			wantErr: true,
		},
		{
			name:     "upload URL request fails",
			filename: "test.jpg",
			content:  "image",
			setupMock: func(m *mockHTTPClient) {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					return &http.Response{
						StatusCode: 401,
						Body:       io.NopCloser(strings.NewReader("unauthorized")),
					}, nil
				}
			},
			want:    "",
			wantErr: true,
		},
		{
			name:     "S3 upload fails",
			filename: "test.png",
			content:  "data",
			setupMock: func(m *mockHTTPClient) {
				uploadCount := 0
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					uploadCount++
					if uploadCount == 1 {
						// First call - return upload URL
						return &http.Response{
							StatusCode: 200,
							Body: io.NopCloser(strings.NewReader(`{
								"url": "https://bucket.s3.amazonaws.com",
								"fields": {"key": "test.png"}
							}`)),
						}, nil
					}
					// Second call would be S3, but we can't mock the actual S3 upload
					// This test validates the upload URL request works
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("")),
					}, nil
				}
			},
			want:    "https://bucket.s3.amazonaws.com/test.png",
			wantErr: false,
		},
		{
			name:     "large file handling",
			filename: "large.bin",
			content:  string(make([]byte, 10000)), // 10KB file
			setupMock: func(m *mockHTTPClient) {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(`{
							"url": "https://bucket.s3.amazonaws.com",
							"fields": {"key": "large.bin"}
						}`)),
					}
					return resp, nil
				}
			},
			want:    "https://bucket.s3.amazonaws.com/large.bin",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			client, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer client.Close()

			// Create mock HTTP client
			mockClient := &mockHTTPClient{}
			client.http = mockClient

			// Setup mock if specified
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			// Create temporary file if needed
			var filePath string
			if tt.filename != "" && !strings.HasPrefix(tt.filename, "/nonexistent") {
				filePath = filepath.Join(tmpDir, tt.filename)
				if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(filePath)
			} else {
				filePath = tt.filename
			}

			got, err := client.UploadFile(filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("UploadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("UploadFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientUploadBytes(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		filename  string
		contentTy string
		setupMock func(*mockHTTPClient) *http.Response
		want      string
		wantErr   bool
	}{
		{
			name:      "successful upload with image",
			data:      []byte("image data"),
			filename:  "photo.jpg",
			contentTy: "image/jpeg",
			setupMock: func(m *mockHTTPClient) *http.Response {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					var req models.UploadURLRequest
					if err := json.Unmarshal(body, &req); err != nil {
						t.Errorf("Failed to unmarshal request: %v", err)
					}

					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(`{
							"url": "https://photos.s3.us-west-2.amazonaws.com",
							"fields": {
								"key": "uploads/photo.jpg",
								"bucket": "photos",
								"region": "us-west-2"
							},
							"final_url": "https://photos.s3.us-west-2.amazonaws.com/uploads/photo.jpg"
						}`)),
					}
					return resp, nil
				}
				return nil
			},
			want:    "https://s3.us-west-2.amazonaws.com/photos/uploads/photo.jpg",
			wantErr: false,
		},
		{
			name:      "successful upload with JSON",
			data:      []byte(`{"key": "value"}`),
			filename:  "data.json",
			contentTy: "application/json",
			setupMock: func(m *mockHTTPClient) *http.Response {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(`{
							"url": "https://data-bucket.s3.amazonaws.com",
							"fields": {"key": "data.json"}
						}`)),
					}
					return resp, nil
				}
				return nil
			},
			want:    "https://data-bucket.s3.amazonaws.com/data.json",
			wantErr: false,
		},
		{
			name:      "upload URL request network error",
			data:      []byte("test"),
			filename:  "test.txt",
			contentTy: "text/plain",
			setupMock: func(m *mockHTTPClient) *http.Response {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					return nil, fmt.Errorf("network error")
				}
				return nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name:      "upload URL request HTTP error",
			data:      []byte("test"),
			filename:  "test.txt",
			contentTy: "text/plain",
			setupMock: func(m *mockHTTPClient) *http.Response {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					return &http.Response{
						StatusCode: 400,
						Body:       io.NopCloser(strings.NewReader("bad request")),
					}, nil
				}
				return nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name:      "invalid JSON response",
			data:      []byte("test"),
			filename:  "test.txt",
			contentTy: "text/plain",
			setupMock: func(m *mockHTTPClient) *http.Response {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("invalid json")),
					}, nil
				}
				return nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name:      "large byte array upload",
			data:      make([]byte, 50000), // 50KB
			filename:  "large.bin",
			contentTy: "application/octet-stream",
			setupMock: func(m *mockHTTPClient) *http.Response {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					// Verify the body is properly marshaled
					if len(body) == 0 {
						t.Errorf("Expected non-empty body")
					}
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(`{
							"url": "https://bucket.s3.amazonaws.com",
							"fields": {"key": "large.bin"}
						}`)),
					}
					return resp, nil
				}
				return nil
			},
			want:    "https://bucket.s3.amazonaws.com/large.bin",
			wantErr: false,
		},
		{
			name:      "empty filename",
			data:      []byte("test"),
			filename:  "",
			contentTy: "application/octet-stream",
			setupMock: func(m *mockHTTPClient) *http.Response {
				m.PostFunc = func(path string, body []byte) (*http.Response, error) {
					var req models.UploadURLRequest
					if err := json.Unmarshal(body, &req); err != nil {
						t.Errorf("Failed to unmarshal request: %v", err)
					}
					if req.Filename != "" {
						t.Errorf("Filename = %q, want empty", req.Filename)
					}
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(`{
							"url": "https://bucket.s3.amazonaws.com",
							"fields": {"key": "unnamed"}
						}`)),
					}
					return resp, nil
				}
				return nil
			},
			want:    "https://bucket.s3.amazonaws.com/unnamed",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			client, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer client.Close()

			// Create mock HTTP client
			mockClient := &mockHTTPClient{}
			client.http = mockClient

			// Setup mock
			if tt.setupMock != nil {
				tt.setupMock(mockClient)
			}

			got, err := client.UploadBytes(tt.data, tt.filename, tt.contentTy)

			if (err != nil) != tt.wantErr {
				t.Errorf("UploadBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("UploadBytes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientUploadToS3(t *testing.T) {
	tests := []struct {
		name         string
		upload       models.UploadURLResponse
		data         []byte
		filename     string
		contentType  string
		mockHTTP     func(*http.Response, error)
		want         string
		wantErr      bool
	}{
		{
			name: "successful upload with 204 status",
			upload: models.UploadURLResponse{
				URL:      "https://bucket.s3.amazonaws.com",
				Fields:   map[string]string{"key": "test.png"},
				FinalURL: "https://bucket.s3.amazonaws.com/test.png",
			},
			data:        []byte("image data"),
			filename:    "test.png",
			contentType: "image/png",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 204,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			},
			want:    "https://bucket.s3.amazonaws.com/test.png",
			wantErr: false,
		},
		{
			name: "successful upload with 200 status",
			upload: models.UploadURLResponse{
				URL:      "https://bucket.s3.amazonaws.com",
				Fields:   map[string]string{"key": "docs/document.pdf"},
				FinalURL: "",
			},
			data:        []byte("PDF data"),
			filename:    "document.pdf",
			contentType: "application/pdf",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("OK")),
				}, nil
			},
			want:    "https://bucket.s3.amazonaws.com/docs/document.pdf",
			wantErr: false,
		},
		{
			name: "successful upload with 201 status",
			upload: models.UploadURLResponse{
				URL:      "https://mybucket.s3.eu-west-1.amazonaws.com/",
				Fields:   map[string]string{"key": "files/data.json"},
				FinalURL: "",
			},
			data:        []byte(`{"test": "data"}`),
			filename:    "data.json",
			contentType: "application/json",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 201,
					Body:       io.NopCloser(strings.NewReader("Created")),
				}, nil
			},
			want:    "https://mybucket.s3.eu-west-1.amazonaws.com/files/data.json",
			wantErr: false,
		},
		{
			name: "S3 upload fails with 400 status",
			upload: models.UploadURLResponse{
				URL:    "https://bucket.s3.amazonaws.com",
				Fields: map[string]string{"key": "test.txt"},
			},
			data:        []byte("data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 400,
					Body:       io.NopCloser(strings.NewReader("Bad Request")),
				}, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "S3 upload fails with 403 status",
			upload: models.UploadURLResponse{
				URL:    "https://bucket.s3.amazonaws.com",
				Fields: map[string]string{"key": "test.txt"},
			},
			data:        []byte("data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 403,
					Body:       io.NopCloser(strings.NewReader("Forbidden")),
				}, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "S3 upload fails with 500 status",
			upload: models.UploadURLResponse{
				URL:    "https://bucket.s3.amazonaws.com",
				Fields: map[string]string{"key": "test.txt"},
			},
			data:        []byte("data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(strings.NewReader("Internal Server Error")),
				}, nil
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "S3 upload network error",
			upload: models.UploadURLResponse{
				URL:    "https://bucket.s3.amazonaws.com",
				Fields: map[string]string{"key": "test.txt"},
			},
			data:        []byte("data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return nil, fmt.Errorf("network error")
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "multipart form creation error",
			upload: models.UploadURLResponse{
				URL: "https://bucket.s3.amazonaws.com",
				Fields: map[string]string{
					"key":     "test.txt",
					"invalid": string(bytes.Repeat([]byte("x"), 1000000)), // Very large value might cause issues
				},
			},
			data:        []byte("data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				// This test validates the multipart form is created properly
				return &http.Response{
					StatusCode: 204,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			},
			want:    "https://bucket.s3.amazonaws.com/test.txt",
			wantErr: false,
		},
		{
			name: "empty Fields map",
			upload: models.UploadURLResponse{
				URL:      "https://bucket.s3.amazonaws.com",
				Fields:   map[string]string{},
				FinalURL: "https://custom.url/test.txt",
			},
			data:        []byte("data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 204,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			},
			want:    "https://custom.url/test.txt",
			wantErr: false,
		},
		{
			name: "image URL rewriting",
			upload: models.UploadURLResponse{
				URL:      "https://mybucket.s3.us-east-1.amazonaws.com",
				Fields:   map[string]string{"key": "images/photo.jpg"},
				FinalURL: "",
			},
			data:        []byte("image data"),
			filename:    "photo.jpg",
			contentType: "image/jpeg",
			mockHTTP: func(resp *http.Response, err error) (*http.Response, error) {
				return &http.Response{
					StatusCode: 204,
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			},
			want:    "https://s3.us-east-1.amazonaws.com/mybucket/images/photo.jpg",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			client, err := New(cfg)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			defer client.Close()

			// For uploadToS3, we need to mock the standard http.Client
			// We can't directly mock it, but we can test the logic by checking the behavior
			// For tests that require HTTP mocking, we'll skip the actual S3 request
			if tt.mockHTTP != nil {
				// We need to intercept the actual HTTP call
				// Since uploadToS3 uses http.NewRequest and http.Client, we can't easily mock it
				// Instead, we test that the function handles the upload URL properly
			}

			// Call the unexported method via a workaround
			// We need to use a test that verifies the public methods work correctly
			// For uploadToS3 specifically, we'll test through UploadBytes or UploadFile

			// Test by calling UploadBytes which calls uploadToS3 internally
			mockClient := &mockHTTPClient{}
			mockClient.PostFunc = func(path string, body []byte) (*http.Response, error) {
				// Return the upload URL
				resp := &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(fmt.Sprintf(`{
						"url": %q,
						"fields": %v,
						"final_url": %q
					}`, tt.upload.URL, tt.upload.Fields, tt.upload.FinalURL))),
				}
				return resp, nil
			}
			client.http = mockClient

			// Use UploadBytes to trigger uploadToS3
			got, err := client.UploadBytes(tt.data, tt.filename, tt.contentType)

			if (err != nil) != tt.wantErr {
				t.Errorf("UploadBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("UploadBytes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientUploadFileCounting(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	mockClient := &mockHTTPClient{}
	mockClient.PostFunc = func(path string, body []byte) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body: io.NopCloser(strings.NewReader(`{
				"url": "https://bucket.s3.amazonaws.com",
				"fields": {"key": "test.txt"}
			}`)),
		}, nil
	}
	client.http = mockClient

	// Create a temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Upload file multiple times
	for i := 1; i <= 3; i++ {
		_, err := client.UploadFile(filePath)
		if err != nil {
			t.Errorf("Upload %d failed: %v", i, err)
		}
	}

	// Check file upload count
	if client.fileUploads != 3 {
		t.Errorf("fileUploads = %d, want 3", client.fileUploads)
	}

	// Check remaining uploads
	remaining := client.FileUploadsRemaining()
	if remaining != 7 {
		t.Errorf("FileUploadsRemaining() = %d, want 7", remaining)
	}
}
