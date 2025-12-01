package client

import (
	"fmt"
	"io"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
	"github.com/diogo/perplexity-go/pkg/models"
)

// MockS3Client is a mock implementation of S3HTTPClient for testing.
type MockS3Client struct {
	Response *stdhttp.Response
	Error    error
}

// Do implements S3HTTPClient.Do for testing.
func (m *MockS3Client) Do(req *stdhttp.Request) (*stdhttp.Response, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	if m.Response == nil {
		return &stdhttp.Response{
			StatusCode: 204,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
	return m.Response, nil
}

// createTestResponse creates a test HTTP response with JSON body
func createTestResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
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
		{"file.tar.gz", "application/octet-stream"}, // .gz is unknown
		{".hidden.jpg", "image/jpeg"},
		{".gitignore", "application/octet-stream"},
		{"file.JPG.PNG", "image/png"}, // Multiple extensions - last one counts
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
		{"file.PNG.GIF", true},  // Multiple extensions - only last should count (.GIF is an image)
		{"file.image.txt", false}, // Last extension is not an image
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
		name      string
		filename  string
		content   string
		mockResp  *http.Response
		want      string
		wantErr   bool
	}{
		{
			name:     "successful image upload",
			filename: "test.png",
			content:  "fake image data",
			mockResp: createTestResponse(200, `{
				"url": "https://test-bucket.s3.us-east-1.amazonaws.com",
				"fields": {
					"key": "uploads/test.png",
					"bucket": "test-bucket",
					"region": "us-east-1"
				},
				"final_url": "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/test.png"
			}`),
			want:    "https://s3.us-east-1.amazonaws.com/test-bucket/uploads/test.png",
			wantErr: false,
		},
		{
			name:     "successful PDF upload",
			filename: "document.pdf",
			content:  "PDF content",
			mockResp: createTestResponse(200, `{
				"url": "https://bucket.s3.eu-west-1.amazonaws.com",
				"fields": {
					"key": "docs/document.pdf"
				}
			}`),
			want:    "https://bucket.s3.eu-west-1.amazonaws.com/docs/document.pdf",
			wantErr: false,
		},
		{
			name:     "file not found error",
			filename: "/nonexistent/path/to/file.txt",
			content:  "",
			mockResp: nil,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "upload URL request fails",
			filename: "test.jpg",
			content:  "image",
			mockResp: createTestResponse(401, "unauthorized"),
			want:     "",
			wantErr:  true,
		},
		{
			name:     "large file handling",
			filename: "large.bin",
			content:  string(make([]byte, 10000)), // 10KB file
			mockResp: createTestResponse(200, `{
				"url": "https://bucket.s3.amazonaws.com",
				"fields": {"key": "large.bin"}
			}`),
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
			mockClient := NewMockHTTPClient()
			if tt.mockResp != nil {
				mockClient.SetResponse(tt.mockResp)
			}
			client.http = mockClient
			// Inject mock S3 client for successful uploads
			client.s3Client = &MockS3Client{}

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
		mockResp  *http.Response
		mockErr   error
		want      string
		wantErr   bool
	}{
		{
			name:      "successful upload with image",
			data:      []byte("image data"),
			filename:  "photo.jpg",
			contentTy: "image/jpeg",
			mockResp: createTestResponse(200, `{
				"url": "https://photos.s3.us-west-2.amazonaws.com",
				"fields": {
					"key": "uploads/photo.jpg",
					"bucket": "photos",
					"region": "us-west-2"
				},
				"final_url": "https://photos.s3.us-west-2.amazonaws.com/uploads/photo.jpg"
			}`),
			want:    "https://s3.us-west-2.amazonaws.com/photos/uploads/photo.jpg",
			wantErr: false,
		},
		{
			name:      "successful upload with JSON",
			data:      []byte(`{"key": "value"}`),
			filename:  "data.json",
			contentTy: "application/json",
			mockResp: createTestResponse(200, `{
				"url": "https://data-bucket.s3.amazonaws.com",
				"fields": {"key": "data.json"}
			}`),
			want:    "https://data-bucket.s3.amazonaws.com/data.json",
			wantErr: false,
		},
		{
			name:    "upload URL request network error",
			data:    []byte("test"),
			filename: "test.txt",
			contentTy: "text/plain",
			mockErr: fmt.Errorf("network error"),
			want:    "",
			wantErr: true,
		},
		{
			name:      "upload URL request HTTP error",
			data:      []byte("test"),
			filename:  "test.txt",
			contentTy: "text/plain",
			mockResp: createTestResponse(400, "bad request"),
			want:      "",
			wantErr:   true,
		},
		{
			name:      "invalid JSON response",
			data:      []byte("test"),
			filename:  "test.txt",
			contentTy: "text/plain",
			mockResp: createTestResponse(200, "invalid json"),
			want:      "",
			wantErr:   true,
		},
		{
			name:      "large byte array upload",
			data:      make([]byte, 50000), // 50KB
			filename:  "large.bin",
			contentTy: "application/octet-stream",
			mockResp: createTestResponse(200, `{
				"url": "https://bucket.s3.amazonaws.com",
				"fields": {"key": "large.bin"}
			}`),
			want:    "https://bucket.s3.amazonaws.com/large.bin",
			wantErr: false,
		},
		{
			name:      "empty filename",
			data:      []byte("test"),
			filename:  "",
			contentTy: "application/octet-stream",
			mockResp: createTestResponse(200, `{
				"url": "https://bucket.s3.amazonaws.com",
				"fields": {"key": "unnamed"}
			}`),
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
			mockClient := NewMockHTTPClient()
			if tt.mockResp != nil {
				mockClient.SetResponse(tt.mockResp)
			}
			if tt.mockErr != nil {
				mockClient.SetError(tt.mockErr)
			}
			client.http = mockClient
			// Inject mock S3 client for successful uploads
			client.s3Client = &MockS3Client{}

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

// TestUploadURLConstruction tests building the final URL from upload response.
func TestUploadURLConstruction(t *testing.T) {
	tests := []struct {
		name     string
		upload   models.UploadURLResponse
		expected string
	}{
		{
			name: "with FinalURL provided",
			upload: models.UploadURLResponse{
				URL:      "https://bucket.s3.amazonaws.com",
				Fields:   map[string]string{"key": "test.png"},
				FinalURL: "https://bucket.s3.amazonaws.com/test.png",
			},
			expected: "https://bucket.s3.amazonaws.com/test.png",
		},
		{
			name: "without FinalURL - uses URL and key",
			upload: models.UploadURLResponse{
				URL:      "https://bucket.s3.amazonaws.com",
				Fields:   map[string]string{"key": "docs/document.pdf"},
				FinalURL: "",
			},
			expected: "https://bucket.s3.amazonaws.com/docs/document.pdf",
		},
		{
			name: "empty Fields map with FinalURL",
			upload: models.UploadURLResponse{
				URL:      "https://bucket.s3.amazonaws.com",
				Fields:   map[string]string{},
				FinalURL: "https://custom.url/test.txt",
			},
			expected: "https://custom.url/test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.upload.FinalURL != "" {
				result = tt.upload.FinalURL
			} else {
				result = tt.upload.URL + "/" + tt.upload.Fields["key"]
			}
			if result != tt.expected {
				t.Errorf("URL construction = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestFileUploadsRemaining tests the file upload counting functionality.
func TestFileUploadsRemaining(t *testing.T) {
	cfg := DefaultConfig()
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// Initial state: should have max uploads available
	if remaining := client.FileUploadsRemaining(); remaining != 10 {
		t.Errorf("Initial FileUploadsRemaining() = %d, want 10", remaining)
	}
}
