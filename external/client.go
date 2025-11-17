package external

import (
	"bytes"
	"context"
	"dokuprime-be/config"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL     string
	messagesURL string
	httpClient  *http.Client
}

func NewClient(cfg *config.ExternalAPIConfig) *Client {
	return &Client{
		baseURL:     cfg.BaseURL,
		messagesURL: cfg.MessagesAPIURL,
		httpClient: &http.Client{
			Timeout: 1000 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				DisableKeepAlives:   false,
			},
		},
	}
}

type ExtractRequest struct {
	ID       int
	Category string
	Filename string
	FilePath string
}

type DeleteRequest struct {
	ID       int
	Category string
}

type ChatRequest struct {
	PlatformUniqueID string `json:"platform_unique_id"`
	Query            string `json:"query"`
	ConversationID   string `json:"conversation_id"`
	Platform         string `json:"platform"`
}

type FlexibleStringArray []string

func (f *FlexibleStringArray) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	if str != "" {
		*f = []string{str}
	} else {
		*f = []string{}
	}
	return nil
}

type ChatResponse struct {
	User             string              `json:"user"`
	ConversationID   string              `json:"conversation_id"`
	Query            string              `json:"query"`
	RewrittenQuery   string              `json:"rewritten_query"`
	Category         string              `json:"category"`
	QuestionCategory []string            `json:"question_category"`
	Answer           string              `json:"answer"`
	Citations        FlexibleStringArray `json:"citations"`
	IsHelpdesk       bool                `json:"is_helpdesk"`
	IsAnswered       *bool               `json:"is_answered"`
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

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	url := c.baseURL + endpoint

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", contentType)
	httpReq.Header.Set("X-API-Key", os.Getenv("X_API_KEY"))
	httpReq.Header.Set("Content-Length", strconv.FormatInt(int64(body.Len()), 10))

	maxRetries := 3
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {

			waitTime := time.Duration(attempt) * time.Second
			time.Sleep(waitTime)

			body.Reset()
			file.Seek(0, 0)
			writer = multipart.NewWriter(body)
			writer.WriteField("id", strconv.Itoa(req.ID))
			writer.WriteField("category", req.Category)
			writer.WriteField("filename", req.Filename)
			part, _ = writer.CreateFormFile("file", req.Filename)
			io.Copy(part, file)
			contentType = writer.FormDataContentType()
			writer.Close()

			httpReq, _ = http.NewRequestWithContext(ctx, "POST", url, body)
			httpReq.Header.Set("Content-Type", contentType)
			httpReq.Header.Set("X-API-Key", os.Getenv("X_API_KEY"))
			httpReq.Header.Set("Content-Length", strconv.FormatInt(int64(body.Len()), 10))
		}

		resp, err = c.httpClient.Do(httpReq)
		if err == nil {
			break
		}
		lastErr = err
	}

	if err != nil {
		return fmt.Errorf("failed to send request after %d attempts: %w", maxRetries, lastErr)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("external API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) DeleteDocument(req DeleteRequest) error {
	url := fmt.Sprintf("%s/api/delete?id=%d&category=%s", c.baseURL, req.ID, strings.ToLower(req.Category))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	httpReq.Header.Set("X-API-Key", os.Getenv("X_API_KEY"))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("external API delete returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) SendChatMessage(req ChatRequest) (*ChatResponse, error) {
	url := c.baseURL + "/api/chat/"

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", os.Getenv("X_API_KEY"))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("external API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chatResp, nil
}

type MessageAPIRequest struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (c *Client) SendMessageToAPI(data interface{}) error {
	url := c.messagesURL + "/api/messages"

	requestBody := MessageAPIRequest{
		Status:  "success",
		Message: "Message sent successfully",
		Data:    data,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", os.Getenv("MESSAGES_API_KEY"))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("messages API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
