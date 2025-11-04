package external

import (
	"bytes"
	"dokuprime-be/config"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg *config.ExternalAPIConfig) *Client {
	return &Client{
		baseURL:    cfg.BaseURL,
		httpClient: &http.Client{},
	}
}

type ExtractRequest struct {
	ID       int
	Category string
	Filename string
	FilePath string
}

func (c *Client) ExtractDocument(req ExtractRequest) error {

	ext := strings.ToLower(filepath.Ext(req.Filename))
	var endpoint string

	switch ext {
	case ".pdf":
		endpoint = "/extract/pdf"
	case ".txt":
		endpoint = "/extract/txt"
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	file, err := os.Open(req.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("id", strconv.Itoa(req.ID)); err != nil {
		return fmt.Errorf("failed to write id field: %w", err)
	}

	if err := writer.WriteField("category", req.Category); err != nil {
		return fmt.Errorf("failed to write category field: %w", err)
	}

	if err := writer.WriteField("filename", req.Filename); err != nil {
		return fmt.Errorf("failed to write filename field: %w", err)
	}

	part, err := writer.CreateFormFile("file", req.Filename)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	url := c.baseURL + endpoint
	httpReq, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-API-Key", os.Getenv("X_API_KEY"))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("external API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
