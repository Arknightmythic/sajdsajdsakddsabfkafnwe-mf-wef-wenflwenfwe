package config

import (
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

func (wsc *WebSocketClient) Connect() error {
	wsc.reconnectMu.Lock()
	defer wsc.reconnectMu.Unlock()

	if wsc.connected {
		return nil
	}

	urlWithToken := fmt.Sprintf("%s?token=%s", wsc.url, wsc.token)
	conn, _, err := websocket.DefaultDialer.Dial(urlWithToken, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	wsc.conn = conn
	wsc.connected = true
	log.Println("✅ Connected to WebSocket server")

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
	go wsc.reconnect()
}

func (wsc *WebSocketClient) processResponse(response WebSocketResponse) {
	switch response.Event {
	case "message":
		wsc.handleMessageEvent(response.Channel, response.Data)
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

	return wsc.conn.WriteJSON(msg)
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

	return wsc.conn.WriteJSON(msg)
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
		return wsc.conn.Close()
	}
	return nil
}

func (wsc *WebSocketClient) IsConnected() bool {
	wsc.mu.Lock()
	defer wsc.mu.Unlock()
	return wsc.connected
}
