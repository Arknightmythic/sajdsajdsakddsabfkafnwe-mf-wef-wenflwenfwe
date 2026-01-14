package external

import (
	"bytes"
	"dokuprime-be/audit"
	"dokuprime-be/config"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	isFailedToRequest = "failed to create request: %w"
	isContentType     = "Content-Type"
	isXAPI            = "X-API-Key"
	isFailedToSend    = "failed to send request: %w"
)

type Client struct {
	baseURL      string
	messagesURL  string
	httpClient   *http.Client
	auditService *audit.AuditService
}

func NewClient(cfg *config.ExternalAPIConfig, auditService *audit.AuditService) *Client {
	return &Client{
		baseURL:      cfg.BaseURL,
		messagesURL:  cfg.MessagesAPIURL,
		httpClient:   &http.Client{},
		auditService: auditService,
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

type Citation struct {
	ID       string
	Filename string
}

type CitationArray [2]string

type FlexibleCitationArray []CitationArray

func (f *FlexibleCitationArray) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*f = []CitationArray{}
		return nil
	}

	if string(data) == `""` || string(data) == `''` {
		*f = []CitationArray{}
		return nil
	}

	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*f = []CitationArray{}
		return nil
	}

	var arr [][]interface{}
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = make([]CitationArray, 0, len(arr))
		for _, item := range arr {
			if len(item) >= 2 {
				citation := CitationArray{
					fmt.Sprintf("%v", item[0]),
					fmt.Sprintf("%v", item[1]),
				}
				*f = append(*f, citation)
			}
		}
		return nil
	}

	var emptyArr []interface{}
	if err := json.Unmarshal(data, &emptyArr); err == nil {
		*f = []CitationArray{}
		return nil
	}

	return fmt.Errorf("unable to unmarshal citations: unexpected format")
}

type FlexibleStringArray []string

func (f *FlexibleStringArray) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*f = []string{}
		return nil
	}

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
	User             string                `json:"user"`
	ConversationID   string                `json:"conversation_id"`
	Query            string                `json:"query"`
	RewrittenQuery   string                `json:"rewritten_query"`
	Category         string                `json:"category"`
	QuestionCategory []string              `json:"question_category"`
	Answer           string                `json:"answer"`
	Citations        FlexibleCitationArray `json:"citations"`
	IsHelpdesk       bool                  `json:"is_helpdesk"`
	IsAnswered       *bool                 `json:"is_answered"`
	QuestionID       int                   `json:"question_id"`
	AnswerID         int                   `json:"answer_id"`
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
		return fmt.Errorf(isFailedToRequest, err)
	}

	httpReq.Header.Set(isContentType, writer.FormDataContentType())
	httpReq.Header.Set(isXAPI, config.AppConfig.XAPIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf(isFailedToSend, err)
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

	httpReq.Header.Set(isXAPI, config.AppConfig.XAPIKey)

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
	startTime := time.Now()
	url := c.baseURL + "/api/chat/"

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf(isFailedToRequest, err)
	}

	httpReq.Header.Set(isContentType, "application/json")
	httpReq.Header.Set(isXAPI, config.AppConfig.XAPIKey)

	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		c.logExternalAPICall("SendChatMessage", "POST", url, string(jsonData), "", nil, err, duration)
		return nil, fmt.Errorf(isFailedToSend, err)
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logExternalAPICall("SendChatMessage", "POST", url, string(jsonData), "", &resp.StatusCode, err, duration)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusGatewayTimeout || resp.StatusCode == http.StatusServiceUnavailable {
		c.logExternalAPICall("SendChatMessage", "POST", url, string(jsonData), string(bodyBytes), &resp.StatusCode, nil, duration)
		var jsonResp map[string]interface{}
		if json.Unmarshal(bodyBytes, &jsonResp) == nil {
			jsonBytes, _ := json.Marshal(jsonResp)
			return nil, fmt.Errorf("external API returned status %d: %s", resp.StatusCode, string(jsonBytes))
		}
		return nil, fmt.Errorf("external API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		c.logExternalAPICall("SendChatMessage", "POST", url, string(jsonData), string(bodyBytes), &resp.StatusCode, nil, duration)
		return nil, fmt.Errorf("external API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		c.logExternalAPICall("SendChatMessage", "POST", url, string(jsonData), string(bodyBytes), &resp.StatusCode, err, duration)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logExternalAPICall("SendChatMessage", "POST", url, string(jsonData), string(bodyBytes), &resp.StatusCode, nil, duration)
	return &chatResp, nil
}

type MessageAPIRequest struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func (c *Client) SendMessageToAPI(data interface{}) error {
	startTime := time.Now()
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
		return fmt.Errorf(isFailedToRequest, err)
	}

	httpReq.Header.Set(isContentType, "application/json")
	httpReq.Header.Set(isXAPI, config.AppConfig.MessagesAPIKey)

	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		c.logExternalAPICall("SendMessageToAPI", "POST", url, string(jsonData), "", nil, err, duration)
		return fmt.Errorf(isFailedToSend, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		c.logExternalAPICall("SendMessageToAPI", "POST", url, string(jsonData), string(bodyBytes), &resp.StatusCode, nil, duration)
		return fmt.Errorf("messages API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	c.logExternalAPICall("SendMessageToAPI", "POST", url, string(jsonData), string(bodyBytes), &resp.StatusCode, nil, duration)
	return nil
}

func (c *Client) logExternalAPICall(action, method, path, reqBody, respBody string, statusCode *int, err error, duration int64) {
	if c.auditService == nil {
		return
	}

	var errorMessage *string
	if err != nil {
		errMsg := err.Error()
		errorMessage = &errMsg
	}

	// Extract platform_unique_id from request body JSON
	var userID *string
	var userType *string
	if reqBody != "" {
		platformUniqueID := extractPlatformUniqueID(reqBody)
		if platformUniqueID != "" {
			userID = &platformUniqueID
			uType := "external_api"
			userType = &uType
		}
	}

	c.auditService.Log(audit.CreateAuditLogRequest{
		UserID:       userID,
		UserType:     userType,
		Action:       action,
		Resource:     "external_api",
		Method:       method,
		Path:         path,
		StatusCode:   statusCode,
		RequestBody:  stringPtr(reqBody),
		ResponseBody: stringPtr(respBody),
		IPAddress:    "system",
		UserAgent:    "dokuprime-backend",
		ErrorMessage: errorMessage,
		Duration:     &duration,
	})
}

// extractPlatformUniqueID extracts platform_unique_id from JSON request body
func extractPlatformUniqueID(jsonBody string) string {
	var data map[string]interface{}

	if err := json.Unmarshal([]byte(jsonBody), &data); err != nil {
		return ""
	}

	// Try to get platform_unique_id directly
	if platformID, ok := data["platform_unique_id"].(string); ok && platformID != "" {
		return platformID
	}

	// If the request body has nested "data" field (like in MessageAPIRequest)
	if nestedData, ok := data["data"].(map[string]interface{}); ok {
		if platformID, ok := nestedData["platform_unique_id"].(string); ok && platformID != "" {
			return platformID
		}
	}

	return ""
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}

	if len(s) > 5000 {
		truncated := s[:5000] + "... (truncated)"
		return &truncated
	}
	return &s
}
