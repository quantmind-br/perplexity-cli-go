package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/diogo/perplexity-go/pkg/models"
)

// S3HTTPClient defines the interface for S3 upload requests.
// This allows injection of mock clients for testing.
type S3HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// UploadFile uploads a file and returns its URL for use in queries.
func (c *Client) UploadFile(filePath string) (string, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	filename := filepath.Base(filePath)
	contentType := detectContentType(filename)

	return c.UploadBytes(data, filename, contentType)
}

// UploadBytes uploads file bytes and returns the URL.
func (c *Client) UploadBytes(data []byte, filename, contentType string) (string, error) {
	// Step 1: Request upload URL
	uploadReq := models.UploadURLRequest{
		Filename:    filename,
		ContentType: contentType,
	}

	reqBody, err := json.Marshal(uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal upload request: %w", err)
	}

	resp, err := c.http.Post(uploadPath, bytes.NewReader(reqBody), nil)
	if err != nil {
		return "", fmt.Errorf("failed to request upload URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload URL request failed %d: %s", resp.StatusCode, string(body))
	}

	var uploadResp models.UploadURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("failed to decode upload response: %w", err)
	}

	// Step 2: Upload to S3
	finalURL, err := c.uploadToS3(uploadResp, data, filename, contentType)
	if err != nil {
		return "", fmt.Errorf("S3 upload failed: %w", err)
	}

	// Track upload count
	c.fileUploads++

	return finalURL, nil
}

// uploadToS3 uploads the file to the S3 bucket.
func (c *Client) uploadToS3(upload models.UploadURLResponse, data []byte, filename, contentType string) (string, error) {
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add fields from upload response
	for key, value := range upload.Fields {
		if err := writer.WriteField(key, value); err != nil {
			return "", fmt.Errorf("failed to write field %s: %w", key, err)
		}
	}

	// Add Content-Type field
	if err := writer.WriteField("Content-Type", contentType); err != nil {
		return "", fmt.Errorf("failed to write Content-Type: %w", err)
	}

	// Add file
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Send to S3 using standard http client (no TLS spoofing needed for S3)
	req, err := http.NewRequest("POST", upload.URL, &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create S3 request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Use injected S3 client if available, otherwise use default http.Client
	var s3Client S3HTTPClient
	if c.s3Client != nil {
		s3Client = c.s3Client
	} else {
		s3Client = &http.Client{}
	}
	resp, err := s3Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("S3 request failed: %w", err)
	}
	defer resp.Body.Close()

	// S3 returns 204 No Content on success
	if resp.StatusCode != 204 && resp.StatusCode != 200 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("S3 upload failed %d: %s", resp.StatusCode, string(body))
	}

	// Construct final URL
	finalURL := upload.FinalURL
	if finalURL == "" {
		// Build URL from upload.Fields["key"]
		if key, ok := upload.Fields["key"]; ok {
			finalURL = strings.TrimSuffix(upload.URL, "/") + "/" + key
		}
	}

	// Handle image URL rewriting
	if isImageFile(filename) {
		finalURL = rewriteImageURL(finalURL)
	}

	return finalURL, nil
}

// detectContentType detects MIME type from filename.
func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}

// isImageFile checks if the filename represents an image.
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return true
	}
	return false
}

// rewriteImageURL rewrites S3 URLs for image display.
// This matches the Python implementation's regex URL rewriting.
func rewriteImageURL(url string) string {
	// Pattern to match S3 bucket URLs
	re := regexp.MustCompile(`https://([^.]+)\.s3\.([^.]+)\.amazonaws\.com/(.+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) == 4 {
		bucket := matches[1]
		region := matches[2]
		key := matches[3]
		return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", region, bucket, key)
	}
	return url
}
