package config

import (
	"dokuprime-be/audit"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Action        string      `json:"action"`
	Channel       string      `json:"channel"`
	Data          interface{} `json:"data,omitempty"`
	MessageID     string      `json:"messageId,omitempty"`
	LastMessageID string      `json:"lastMessageId,omitempty"`
}

type WebSocketResponse struct {
	Event    string          `json:"event"`
	Channel  string          `json:"channel"`
	StreamID string          `json:"streamId"`
	Data     json.RawMessage `json:"data"`
	Status   string          `json:"status"`
	Message  string          `json:"message"`
	Error    string          `json:"error"`
}

type WebSocketClient struct {
	conn            *websocket.Conn
	url             string
	token           string
	mu              sync.Mutex
	reconnectMu     sync.Mutex
	messageHandlers map[string][]func(json.RawMessage)
	connected       bool
	reconnecting    bool
	auditService    *audit.AuditService
}

func NewWebSocketClient(url, token string) *WebSocketClient {
	return &WebSocketClient{
		url:             url,
		token:           token,
		messageHandlers: make(map[string][]func(json.RawMessage)),
		connected:       false,
		reconnecting:    false,
	}
}

func (wsc *WebSocketClient) SetAuditService(auditService *audit.AuditService) {
	wsc.auditService = auditService
}

func (wsc *WebSocketClient) Connect() error {
	wsc.reconnectMu.Lock()
	defer wsc.reconnectMu.Unlock()

	if wsc.connected {
		return nil
	}

	urlWithToken := fmt.Sprintf("%s?token=%s", wsc.url, wsc.token)
	conn, _, err := websocket.DefaultDialer.Dial(urlWithToken, nil)
	if err != nil {
		wsc.logWebSocketAction("CONNECT", "websocket", urlWithToken, "", err)
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	wsc.conn = conn
	wsc.connected = true
	log.Println("✅ Connected to WebSocket server")
	wsc.logWebSocketAction("CONNECT", "websocket", urlWithToken, "Connected successfully", nil)

	go wsc.readMessages()

	return nil
}

func (wsc *WebSocketClient) readMessages() {
	defer wsc.handleDisconnect()

	for {
		var response WebSocketResponse
		err := wsc.conn.ReadJSON(&response)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		wsc.processResponse(response)
	}
}

func (wsc *WebSocketClient) handleDisconnect() {
	wsc.mu.Lock()
	wsc.connected = false
	wsc.mu.Unlock()
	wsc.logWebSocketAction("DISCONNECT", "websocket", wsc.url, "Connection lost", nil)
	go wsc.reconnect()
}

func (wsc *WebSocketClient) processResponse(response WebSocketResponse) {
	switch response.Event {
	case "message":
		wsc.handleMessageEvent(response.Channel, response.Data)

		dataStr := string(response.Data)
		wsc.logWebSocketAction("RECEIVE", response.Channel, response.Channel, dataStr, nil)
	default:
		if response.Status != "" {
			log.Printf("WebSocket status: %s - %s", response.Status, response.Message)
		}
	}
}

func (wsc *WebSocketClient) handleMessageEvent(channel string, data json.RawMessage) {
	wsc.mu.Lock()
	handlers, exists := wsc.messageHandlers[channel]
	wsc.mu.Unlock()

	if exists {
		for _, handler := range handlers {
			go handler(data)
		}
	}
}

func (wsc *WebSocketClient) reconnect() {
	wsc.reconnectMu.Lock()
	if wsc.reconnecting {
		wsc.reconnectMu.Unlock()
		return
	}
	wsc.reconnecting = true
	wsc.reconnectMu.Unlock()

	defer func() {
		wsc.reconnectMu.Lock()
		wsc.reconnecting = false
		wsc.reconnectMu.Unlock()
	}()

	for i := 0; i < 5; i++ {
		log.Printf("Attempting to reconnect to WebSocket (attempt %d/5)...", i+1)
		time.Sleep(time.Second * time.Duration(i+1))

		if err := wsc.Connect(); err == nil {
			log.Println("✅ Reconnected to WebSocket server")

			wsc.mu.Lock()
			channels := make([]string, 0, len(wsc.messageHandlers))
			for channel := range wsc.messageHandlers {
				channels = append(channels, channel)
			}
			wsc.mu.Unlock()

			for _, channel := range channels {
				if err := wsc.Subscribe(channel, "$"); err != nil {
					log.Printf("Failed to resubscribe to channel %s: %v", channel, err)
				}
			}
			return
		}
	}

	log.Println("❌ Failed to reconnect to WebSocket after 5 attempts")
	wsc.logWebSocketAction("RECONNECT_FAILED", "websocket", wsc.url, "Failed after 5 attempts", nil)
}

func (wsc *WebSocketClient) Subscribe(channel, lastMessageID string) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if !wsc.connected {
		return fmt.Errorf("WebSocket not connected")
	}

	msg := WebSocketMessage{
		Action:        "subscribe",
		Channel:       channel,
		LastMessageID: lastMessageID,
	}

	err := wsc.conn.WriteJSON(msg)
	if err != nil {
		wsc.logWebSocketAction("SUBSCRIBE", channel, channel, "", err)
	} else {
		msgJSON, _ := json.Marshal(msg)
		wsc.logWebSocketAction("SUBSCRIBE", channel, channel, string(msgJSON), nil)
	}
	return err
}

func (wsc *WebSocketClient) Publish(channel string, data interface{}) error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if !wsc.connected {
		return fmt.Errorf("WebSocket not connected")
	}

	msg := WebSocketMessage{
		Action:    "publish",
		Channel:   channel,
		Data:      data,
		MessageID: uuid.New().String(),
	}

	err := wsc.conn.WriteJSON(msg)

	msgJSON, _ := json.Marshal(msg)
	if err != nil {
		wsc.logWebSocketAction("PUBLISH", channel, channel, string(msgJSON), err)
	} else {
		wsc.logWebSocketAction("PUBLISH", channel, channel, string(msgJSON), nil)
	}

	return err
}

func (wsc *WebSocketClient) OnMessage(channel string, handler func(json.RawMessage)) {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if _, exists := wsc.messageHandlers[channel]; !exists {
		wsc.messageHandlers[channel] = make([]func(json.RawMessage), 0)
	}

	wsc.messageHandlers[channel] = append(wsc.messageHandlers[channel], handler)
}

func (wsc *WebSocketClient) Close() error {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()

	if wsc.conn != nil {
		wsc.connected = false
		wsc.logWebSocketAction("CLOSE", "websocket", wsc.url, "Connection closed", nil)
		return wsc.conn.Close()
	}
	return nil
}

func (wsc *WebSocketClient) IsConnected() bool {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	return wsc.connected
}

func (wsc *WebSocketClient) logWebSocketAction(action, channel, path, message string, err error) {
	if wsc.auditService == nil {
		return
	}

	var errorMessage *string
	if err != nil {
		errMsg := err.Error()
		errorMessage = &errMsg
	}

	var statusCode *int
	if err == nil {
		status := 200
		statusCode = &status
	} else {
		status := 500
		statusCode = &status
	}

	wsc.auditService.Log(audit.CreateAuditLogRequest{
		UserID:       nil,
		Action:       action,
		Resource:     "websocket",
		Method:       "WS",
		Path:         path,
		StatusCode:   statusCode,
		RequestBody:  stringPtr(message),
		ResponseBody: nil,
		IPAddress:    "system",
		UserAgent:    "websocket-client",
		ErrorMessage: errorMessage,
		Duration:     nil,
	})
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
