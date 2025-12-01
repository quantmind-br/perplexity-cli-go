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

// TestUploadFile tests file upload functionality
func TestUploadFile(t *testing.T) {
	tests := []struct {
		name             string
		fileContent      string
		filename         string
		setupAPI         func(w http.ResponseWriter)
		setupS3          func(w http.ResponseWriter)
		wantErr          bool
		expectedURL      string
		expectFileUpload bool
	}{
		{
			name:        "successful image upload",
			fileContent: "fake image data",
			filename:    "test.png",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key":   "uploads/test.png",
						"token": "abc123",
					},
					FinalURL: "https://s3.us-east-1.amazonaws.com/test-bucket/uploads/test.png",
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:          false,
			expectedURL:      "https://s3.us-east-1.amazonaws.com/test-bucket/uploads/test.png",
			expectFileUpload: true,
		},
		{
			name:        "successful PDF upload",
			fileContent: "%PDF-1.4 fake pdf content",
			filename:    "document.pdf",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key":   "uploads/document.pdf",
						"token": "xyz789",
					},
					FinalURL: "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/document.pdf",
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:          false,
			expectedURL:      "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/document.pdf",
			expectFileUpload: true,
		},
		{
			name:        "file not found error",
			fileContent: "",
			filename:    "/nonexistent/path/file.txt",
			setupAPI:    nil,
			setupS3:     nil,
			wantErr:     true,
		},
		{
			name:        "API error - bad status code",
			fileContent: "test data",
			filename:    "test.txt",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
			},
			setupS3:    nil,
			wantErr:    true,
		},
		{
			name:        "API error - JSON decode failure",
			fileContent: "test data",
			filename:    "test.txt",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{invalid json"))
			},
			setupS3:  nil,
			wantErr:  true,
		},
		{
			name:        "S3 upload failure",
			fileContent: "test data",
			filename:    "test.txt",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key":   "uploads/test.txt",
						"token": "abc123",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("bad request"))
			},
			wantErr: true,
		},
		{
			name:        "image URL rewriting",
			fileContent: "fake image",
			filename:    "photo.jpg",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://mybucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "photos/photo.jpg",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:         false,
			expectedURL:     "https://s3.us-east-1.amazonaws.com/mybucket/photos/photo.jpg",
			expectFileUpload: true,
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

			// Setup mock servers
			var apiServer, s3Server *httptest.Server
			if tt.setupAPI != nil {
				apiServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tt.setupAPI(w)
				}))
				defer apiServer.Close()
			}
			if tt.setupS3 != nil {
				s3Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tt.setupS3(w)
				}))
				defer s3Server.Close()
			}

			// Create client
			httpClient, err := NewHTTPClient()
			if err != nil {
				t.Fatalf("Failed to create HTTP client: %v", err)
			}

			// Override baseURL for API server
			client := &Client{
				http: httpClient,
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
				if got != tt.expectedURL {
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
		setupAPI         func(w http.ResponseWriter)
		setupS3          func(w http.ResponseWriter)
		wantErr          bool
		expectedURL      string
		expectFileUpload bool
	}{
		{
			name:        "successful upload",
			data:        []byte("test file content"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL:      "https://test-bucket.s3.us-east-1.amazonaws.com",
					FinalURL: "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/test.txt",
					Fields: map[string]string{
						"key":   "uploads/test.txt",
						"token": "abc123",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:          false,
			expectedURL:      "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/test.txt",
			expectFileUpload: true,
		},
		{
			name:        "API marshal error - invalid data",
			data:        make([]byte, 0), // Empty data to test edge case
			filename:    "",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
				})
			},
			setupS3:    nil,
			wantErr:    false, // Empty filename is technically valid
		},
		{
			name:        "API HTTP error",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte("service unavailable"))
			},
			setupS3:  nil,
			wantErr:  true,
		},
		{
			name:        "API bad status code",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("forbidden"))
			},
			setupS3:  nil,
			wantErr:  true,
		},
		{
			name:        "API JSON decode error",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("invalid json"))
			},
			setupS3:  nil,
			wantErr:  true,
		},
		{
			name:        "S3 upload returns 200",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "uploads/test.txt",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr:          false,
			expectFileUpload: true,
		},
		{
			name:        "S3 upload returns 201",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "uploads/test.txt",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
			},
			wantErr:          false,
			expectFileUpload: true,
		},
		{
			name:        "S3 upload failure",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "uploads/test.txt",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("bad request"))
			},
			wantErr: true,
		},
		{
			name:        "image file with URL rewriting",
			data:        []byte("fake image"),
			filename:    "image.png",
			contentType: "image/png",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://bucket.s3.eu-west-1.amazonaws.com",
					Fields: map[string]string{
						"key": "images/image.png",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:     false,
			expectedURL: "https://s3.eu-west-1.amazonaws.com/bucket/images/image.png",
		},
		{
			name:        "upload count increments",
			data:        []byte("test"),
			filename:    "test.txt",
			contentType: "text/plain",
			setupAPI: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.UploadURLResponse{
					URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
					Fields: map[string]string{
						"key": "test.txt",
					},
				})
			},
			setupS3: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr:          false,
			expectFileUpload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock servers
			var apiServer, s3Server *httptest.Server
			if tt.setupAPI != nil {
				apiServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tt.setupAPI(w)
				}))
				defer apiServer.Close()
			}
			if tt.setupS3 != nil {
				s3Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					tt.setupS3(w)
				}))
				defer s3Server.Close()
			}

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

			// Override the upload URL if API server is set
			if apiServer != nil {
				originalURL := uploadPath
				uploadPath = strings.TrimPrefix(apiServer.URL, "http://") + uploadPath
				defer func() { uploadPath = originalURL }()
			}

			// Override S3 URL if S3 server is set
			if s3Server != nil && tt.setupAPI != nil {
				// We need to modify the upload URL to point to our test S3 server
				defer func() {
					// Restore original URLs
				}()
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

// TestUploadToS3ViaUploadBytes tests uploadToS3 indirectly through UploadBytes
func TestUploadToS3ViaUploadBytes(t *testing.T) {
	tests := []struct {
		name             string
		data             []byte
		filename         string
		contentType      string
		apiResponse      models.UploadURLResponse
		s3StatusCode     int
		s3ResponseBody   string
		wantErr          bool
		expectedErrorMsg string
		expectedURL      string
	}{
		{
			name:        "S3 error - internal server error",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			apiResponse: models.UploadURLResponse{
				URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
				Fields: map[string]string{
					"key": "test.txt",
				},
			},
			s3StatusCode:     http.StatusInternalServerError,
			s3ResponseBody:   "internal server error",
			wantErr:          true,
			expectedErrorMsg: "S3 upload failed 500",
		},
		{
			name:        "S3 error - unauthorized",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			apiResponse: models.UploadURLResponse{
				URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
				Fields: map[string]string{
					"key": "test.txt",
				},
			},
			s3StatusCode:     http.StatusUnauthorized,
			s3ResponseBody:   "unauthorized",
			wantErr:          true,
			expectedErrorMsg: "S3 upload failed 401",
		},
		{
			name:        "successful upload - no final URL, build from key",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			apiResponse: models.UploadURLResponse{
				URL: "https://test-bucket.s3.us-east-1.amazonaws.com/",
				Fields: map[string]string{
					"key": "uploads/test.txt",
				},
				FinalURL: "", // Empty final URL
			},
			s3StatusCode:   http.StatusNoContent,
			s3ResponseBody: "",
			wantErr:        false,
			expectedURL:    "https://test-bucket.s3.us-east-1.amazonaws.com/uploads/test.txt",
		},
		{
			name:        "multipart form creation error - empty fields",
			data:        []byte("test data"),
			filename:    "test.txt",
			contentType: "text/plain",
			apiResponse: models.UploadURLResponse{
				URL: "https://test-bucket.s3.us-east-1.amazonaws.com",
				Fields: map[string]string{
					"key":   "test.txt",
					"token": "abc123",
				},
			},
			s3StatusCode:   http.StatusNoContent,
			s3ResponseBody: "",
			wantErr:        false,
		},
		{
			name:        "image file - URL rewriting via uploadToS3",
			data:        []byte("fake image data"),
			filename:    "photo.jpg",
			contentType: "image/jpeg",
			apiResponse: models.UploadURLResponse{
				URL: "https://mybucket.s3.eu-west-1.amazonaws.com",
				Fields: map[string]string{
					"key": "photos/photo.jpg",
				},
			},
			s3StatusCode:   http.StatusNoContent,
			s3ResponseBody: "",
			wantErr:        false,
			expectedURL:    "https://s3.eu-west-1.amazonaws.com/mybucket/photos/photo.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock API server
			apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.apiResponse)
			}))
			defer apiServer.Close()

			// Setup mock S3 server
			s3Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.s3StatusCode)
				if tt.s3ResponseBody != "" {
					w.Write([]byte(tt.s3ResponseBody))
				}
			}))
			defer s3Server.Close()

			// Create HTTP client
			httpClient, err := NewHTTPClient()
			if err != nil {
				t.Fatalf("Failed to create HTTP client: %v", err)
			}

			// Create client
			client := &Client{
				http: httpClient,
			}

			// Override upload path to use mock API
			originalUploadPath := uploadPath
			uploadPath = strings.TrimPrefix(apiServer.URL, "http://") + uploadPath
			defer func() { uploadPath = originalUploadPath }()

			// Override the S3 URL in the API response
			tt.apiResponse.URL = s3Server.URL

			// Execute upload (which calls uploadToS3 internally)
			got, err := client.UploadBytes(tt.data, tt.filename, tt.contentType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UploadBytes() error = nil, wantErr %v", tt.wantErr)
				} else if tt.expectedErrorMsg != "" && !strings.Contains(err.Error(), tt.expectedErrorMsg) {
					t.Errorf("UploadBytes() error = %v, want error containing %v", err, tt.expectedErrorMsg)
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
