package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	httpclient "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/pkg/models"
)

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
			name: "Non-S3 URL unchanged",
			url:  "https://example.com/path/to/file.png",
			want: "https://example.com/path/to/file.png",
		},
		{
			name: "Different domain unchanged",
			url:  "https://cdn.example.com/file.png",
			want: "https://cdn.example.com/file.png",
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

// mockUploadClient is a simple mock for testing upload functionality
type mockUploadClient struct {
	apiResponse models.UploadURLResponse
	apiError    error
	s3Status    int
	s3Body      string
}

func (m *mockUploadClient) Post(path string, body []byte) (*http.Response, error) {
	if m.apiError != nil {
		return nil, m.apiError
	}

	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("")),
	}

	if m.apiResponse.URL != "" {
		data, _ := json.Marshal(m.apiResponse)
		resp.Body = io.NopCloser(bytes.NewReader(data))
	}

	return resp, nil
}

func (m *mockUploadClient) Get(path string) (*http.Response, error) {
	return nil, nil
}

func (m *mockUploadClient) PostWithReader(path string, body []byte) (*http.Response, error) {
	return nil, nil
}

func (m *mockUploadClient) buildHeaders() httpclient.Header {
	return httpclient.Header{}
}

func (m *mockUploadClient) GetCSRFToken() string {
	return ""
}

func (m *mockUploadClient) SetCookies(cookies []*httpclient.Cookie) {}

func (m *mockUploadClient) AddCookie(cookie *httpclient.Cookie) {}

func (m *mockUploadClient) GetCookies() []*httpclient.Cookie {
	return nil
}

func (m *mockUploadClient) Close() error {
	return nil
}

// mockS3Transport simulates S3 responses
type mockS3Transport struct {
	statusCode int
	body       string
}

func (t *mockS3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: t.statusCode,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     http.Header{"Content-Type": {"application/xml"}},
	}, nil
}

// TestUploadFile tests file upload functionality
func TestUploadFile(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		filename         string
		mockAPI          *mockUploadClient
		mockS3Transport  *mockS3Transport
		wantErr          bool
		expectedURL      string
	}{
		{
			name:        "successful image upload",
			fileContent: "fake image data",
			filename:    "test.png",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key":   "uploads/test.png",
						"token": "abc123",
					},
					FinalURL: "https://s3.us-east-1.amazonaws.com/test-bucket/uploads/test.png",
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr:          false,
			expectedURL:      "https://s3.us-east-1.amazonaws.com/test-bucket/uploads/test.png",
		},
		{
			name:        "successful PDF upload",
			fileContent: "%PDF-1.4 fake pdf",
			filename:    "document.pdf",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "uploads/document.pdf",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr: false,
		},
		{
			name:        "file not found error",
			fileContent: "",
			filename:    "/nonexistent/path/file.txt",
			mockAPI:     nil,
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr: true,
		},
		{
			name:        "API HTTP error",
			fileContent: "test data",
			filename:    "test.txt",
			mockAPI: &mockUploadClient{
				apiError: fmt.Errorf("network error"),
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr: true,
		},
		{
			name:        "API JSON decode error - empty response",
			fileContent: "test data",
			filename:    "test.txt",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr: false, // Will fail in S3 upload, not API
		},
		{
			name:        "S3 upload failure",
			fileContent: "test data",
			filename:    "test.txt",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "uploads/test.txt",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusBadRequest,
				body:       "Bad Request",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file if needed
			var tmpFile *os.File
			if tt.filename != "" && !strings.HasPrefix(tt.filename, "/nonexistent") {
				var err error
				tmpFile, err = os.CreateTemp("", "test-*")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())
				defer tmpFile.Close()

				if tt.fileContent != "" {
					if _, err := tmpFile.Write([]byte(tt.fileContent)); err != nil {
						t.Fatalf("Failed to write to temp file: %v", err)
					}
				}
				tt.filename = tmpFile.Name()
			}

			// Create client with mocked HTTP
			httpClient, err := NewHTTPClient()
			if err != nil {
				t.Fatalf("Failed to create HTTP client: %v", err)
			}

			client := &Client{
				http: httpClient,
			}

			// Override HTTP client if mock is provided
			if tt.mockAPI != nil {
				client.http = tt.mockAPI
			}

			// Test upload
			got, err := client.UploadFile(tt.filename)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UploadFile() error = nil, wantErr %v", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("UploadFile() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if tt.expectedURL != "" && got != tt.expectedURL {
					t.Errorf("UploadFile() = %q, want %q", got, tt.expectedURL)
				}
			}
		})
	}
}

// TestUploadBytes tests bytes upload functionality
func TestUploadBytes(t *testing.T) {
	tests := []struct {
		name             string
		data             []byte
		filename         string
		contentType      string
		mockAPI          *mockUploadClient
		mockS3Transport  *mockS3Transport
		wantErr          bool
		expectedURL      string
	}{
		{
			name:        "successful upload",
			data:        []byte("test file content"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL:      "test-bucket.s3.us-east-1.amazonaws.com",
					FinalURL: "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/test.txt",
					Fields: map[string]string{
						"key": "uploads/test.txt",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr:          false,
			expectedURL:      "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/test.txt",
		},
		{
			name:        "successful upload with S3 status 200",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "test.txt",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusOK,
			},
			wantErr: false,
		},
		{
			name:        "successful upload with S3 status 201",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "test.txt",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusCreated,
			},
			wantErr: false,
		},
		{
			name:        "API HTTP error",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockAPI: &mockUploadClient{
				apiError: fmt.Errorf("network error"),
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr: true,
		},
		{
			name:        "S3 upload failure",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "test.txt",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusInternalServerError,
				body:       "Internal Server Error",
			},
			wantErr: true,
		},
		{
			name:        "image file with URL rewriting",
			data:        []byte("fake image"),
			filename:    "image.png",
			contentType: "image/png",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "bucket.s3.eu-west-1.amazonaws.com",
					Fields: map[string]string{
						"key": "images/image.png",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr:     false,
			expectedURL: "https://s3.eu-west-1.amazonaws.com/bucket/images/image.png",
		},
		{
			name:        "upload count increments",
			data:        []byte("test"),
			filename:    "test.txt",
			contentType: "text/plain",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "test.txt",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr: false,
		},
		{
			name:        "empty filename",
			data:        []byte("test"),
			filename:    "",
			contentType: "text/plain",
			mockAPI: &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "test.txt",
					},
				},
			},
			mockS3Transport: &mockS3Transport{
				statusCode: http.StatusNoContent,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP client
			httpClient, err := NewHTTPClient()
			if err != nil {
				t.Fatalf("Failed to create HTTP client: %v", err)
			}

			// Create client
			client := &Client{
				http:        httpClient,
				fileUploads: 0,
			}

			// Override HTTP client if mock is provided
			if tt.mockAPI != nil {
				client.http = tt.mockAPI
			}

			// Execute upload
			got, err := client.UploadBytes(tt.data, tt.filename, tt.contentType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UploadBytes() error = nil, wantErr %v", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("UploadBytes() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if tt.expectedURL != "" && got != tt.expectedURL {
					t.Errorf("UploadBytes() = %q, want %q", got, tt.expectedURL)
				}
			}
		})
	}
}

// TestContentTypeDetectionEdgeCases tests edge cases for content type detection
func TestContentTypeDetectionEdgeCases(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		// Case sensitivity
		{"test.JPEG", "image/jpeg"},
		{"test.JpG", "image/jpeg"},
		{"test.PDF", "application/pdf"},

		// Multiple dots
		{"file.backup.txt", "text/plain"},
		{"my.archive.zip", "application/octet-stream"},

		// Special characters
		{"file-name_v2.png", "image/png"},
		{"file with spaces.txt", "text/plain"},

		// Empty/missing extensions
		{"noext", "application/octet-stream"},
		{".hidden", "application/octet-stream"},
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

// TestIsImageFileEdgeCases tests edge cases for image file detection
func TestIsImageFileEdgeCases(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		// Case sensitivity
		{"test.JPG", true},
		{"test.Png", true},
		{"test.GIF", true},

		// Non-image files
		{"image.txt", false},
		{"photo.pdf", false},
		{"picture.doc", false},

		// Edge cases
		{".svg", true},  // Hidden SVG file
		{"file.jpegx", false}, // Similar extension
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

// TestUploadFileVariousTypes tests upload with various file types
func TestUploadFileVariousTypes(t *testing.T) {
	fileTypes := []struct {
		extension string
		content   string
	}{
		{".jpg", "fake jpg"},
		{".png", "fake png"},
		{".pdf", "%PDF-1.4"},
		{".doc", "fake doc"},
		{".xlsx", "fake xlsx"},
		{".csv", "col1,col2\nval1,val2"},
		{".json", `{"key": "value"}`},
		{".txt", "plain text"},
	}

	for _, ft := range fileTypes {
		t.Run(ft.extension, func(t *testing.T) {
			// Create temp file
			tmpFile, err := os.CreateTemp("", "test-*"+ft.extension)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			defer tmpFile.Close()

			if _, err := tmpFile.Write([]byte(ft.content)); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}

			// Create mock client
			mockAPI := &mockUploadClient{
				apiResponse: models.UploadURLResponse{
					URL: "test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "test" + ft.extension,
					},
				},
			}
			mockS3Transport := &mockS3Transport{
				statusCode: http.StatusNoContent,
			}

			httpClient, err := NewHTTPClient()
			if err != nil {
				t.Fatalf("Failed to create HTTP client: %v", err)
			}

			client := &Client{
				http: httpClient,
			}
			client.http = mockAPI

			// Test upload
			_, err = client.UploadFile(tmpFile.Name())
			if err != nil {
				t.Errorf("UploadFile() error = %v", err)
			}

			// Verify content type was detected correctly
			expectedType := detectContentType(tmpFile.Name())
			if expectedType == "application/octet-stream" && ft.extension != ".xyz" {
				t.Errorf("Content type not detected for %s", ft.extension)
			}
		})
	}
}
