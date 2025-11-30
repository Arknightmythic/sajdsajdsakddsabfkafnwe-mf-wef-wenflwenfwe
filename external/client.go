package external

import (
	"bytes"
	"dokuprime-be/config"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		httpClient:  &http.Client{},
	}
}

type ExtractRequest struct {
	ID       string
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
	StartTimestamp   string `json:"start_timestamp"`
}

type FlexibleStringArray []string

func (f *FlexibleStringArray) UnmarshalJSON(data []byte) error {

	var arr [][]interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = make([]string, 0, len(arr))
		for _, item := range arr {
			if len(item) >= 2 {

				filename := fmt.Sprintf("%v", item[1])
				*f = append(*f, filename)
			}
		}
		return nil
	}

	var strArr []string
	if err := json.Unmarshal(data, &strArr); err == nil {
		*f = strArr
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
	QuestionID       int                 `json:"question_id"`
	AnswerID         int                 `json:"answer_id"`
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

	if err := writer.WriteField("id", req.ID); err != nil {
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

func (c *Client) DeleteDocument(req DeleteRequest) error {
	url := fmt.Sprintf("%s/api/delete?id=%d&category=%s", c.baseURL, req.ID, strings.ToLower(req.Category))

	httpReq, err := http.NewRequest("DELETE", url, nil)
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

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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
	url := c.messagesURL + "/api/send/reply"

	requestBody := MessageAPIRequest{
		Status:  "success",
		Message: "Message sent successfully",
		Data:    data,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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
