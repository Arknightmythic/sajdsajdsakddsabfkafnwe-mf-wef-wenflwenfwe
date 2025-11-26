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

type WebSocketClient struct {
	conn              *websocket.Conn
	url               string
	token             string
	mu                sync.Mutex
	isConnected       bool
	reconnectInterval time.Duration
	subscriptions     map[string][]MessageHandler
	subsMu            sync.RWMutex
	pendingAcks       map[string]chan AckResponse
	acksMu            sync.RWMutex
	stopChan          chan struct{}
}

type MessageHandler func(msg IncomingMessage)

type IncomingMessage struct {
	Event    string          `json:"event"`
	Channel  string          `json:"channel"`
	StreamID string          `json:"streamId"`
	Data     json.RawMessage `json:"data"`
}

type OutgoingMessage struct {
	Action        string      `json:"action"`
	Channel       string      `json:"channel"`
	Data          interface{} `json:"data,omitempty"`
	MessageID     string      `json:"messageId,omitempty"`
	LastMessageID string      `json:"lastMessageId,omitempty"`
}

type AckResponse struct {
	Status    string `json:"status"`
	MessageID string `json:"messageId,omitempty"`
	Error     string `json:"error,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Message   string `json:"message,omitempty"`
}

type Config struct {
	URL               string
	Token             string
	ReconnectInterval time.Duration
}

var (
	defaultClient *WebSocketClient
	once          sync.Once
)

func InitWebSocketClient(config Config) error {
	var err error
	once.Do(func() {
		if config.ReconnectInterval == 0 {
			config.ReconnectInterval = 5 * time.Second
		}

		defaultClient = &WebSocketClient{
			url:               config.URL,
			token:             config.Token,
			reconnectInterval: config.ReconnectInterval,
			subscriptions:     make(map[string][]MessageHandler),
			pendingAcks:       make(map[string]chan AckResponse),
			stopChan:          make(chan struct{}),
		}

		err = defaultClient.Connect()
		if err != nil {
			log.Printf("❌ Gagal koneksi WebSocket: %v", err)

			go defaultClient.autoReconnect()
		} else {
			log.Println("✅ WebSocket client terhubung")
		}
	})
	return err
}

func GetClient() *WebSocketClient {
	if defaultClient == nil {
		log.Fatal("WebSocket client belum diinisialisasi. Panggil InitWebSocketClient terlebih dahulu.")
	}
	return defaultClient
}

func (c *WebSocketClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return nil
	}

	wsURL := fmt.Sprintf("%s?token=%s", c.url, c.token)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("gagal dial WebSocket: %w", err)
	}

	c.conn = conn
	c.isConnected = true

	go c.readMessages()

	return nil
}

func (c *WebSocketClient) autoReconnect() {
	ticker := time.NewTicker(c.reconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if !c.isConnected {
				log.Println("🔄 Mencoba reconnect ke WebSocket...")
				if err := c.Connect(); err != nil {
					log.Printf("❌ Reconnect gagal: %v", err)
				} else {
					log.Println("✅ Reconnect berhasil")

					c.resubscribeAll()
				}
			}
		}
	}
}

func (c *WebSocketClient) resubscribeAll() {
	c.subsMu.RLock()
	channels := make([]string, 0, len(c.subscriptions))
	for channel := range c.subscriptions {
		channels = append(channels, channel)
	}
	c.subsMu.RUnlock()

	for _, channel := range channels {
		if err := c.Subscribe(channel, "$", nil); err != nil {
			log.Printf("❌ Gagal re-subscribe ke %s: %v", channel, err)
		}
	}
}

func (c *WebSocketClient) readMessages() {
	defer func() {
		c.mu.Lock()
		c.isConnected = false
		if c.conn != nil {
			c.conn.Close()
		}
		c.mu.Unlock()
		log.Println("⚠️ WebSocket connection closed, akan reconnect...")
		go c.autoReconnect()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("❌ Error membaca pesan: %v", err)
			return
		}

		var ackResp AckResponse
		if err := json.Unmarshal(message, &ackResp); err == nil {

			if ackResp.Status == "ack" || ackResp.Status == "error_ack" || ackResp.Status == "subscribed" {
				c.handleAck(ackResp)
				continue
			}
		}

		var incoming IncomingMessage
		if err := json.Unmarshal(message, &incoming); err != nil {
			log.Printf("⚠️ Gagal parse pesan: %v", err)
			continue
		}

		if incoming.Event == "message" {
			c.subsMu.RLock()
			handlers, ok := c.subscriptions[incoming.Channel]
			c.subsMu.RUnlock()

			if ok {
				for _, handler := range handlers {
					go handler(incoming)
				}
			}
		}
	}
}

func (c *WebSocketClient) handleAck(ack AckResponse) {
	c.acksMu.RLock()
	ackChan, ok := c.pendingAcks[ack.MessageID]
	c.acksMu.RUnlock()

	if ok {
		select {
		case ackChan <- ack:
		case <-time.After(1 * time.Second):
			log.Printf("⚠️ Timeout mengirim ack untuk messageId: %s", ack.MessageID)
		}
	}
}

func (c *WebSocketClient) Subscribe(channel string, lastMessageID string, handler MessageHandler) error {
	if !c.isConnected {
		return fmt.Errorf("WebSocket tidak terhubung")
	}

	if handler != nil {
		c.subsMu.Lock()
		c.subscriptions[channel] = append(c.subscriptions[channel], handler)
		c.subsMu.Unlock()
	}

	msg := OutgoingMessage{
		Action:        "subscribe",
		Channel:       channel,
		LastMessageID: lastMessageID,
	}

	c.mu.Lock()
	err := c.conn.WriteJSON(msg)
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("gagal subscribe: %w", err)
	}

	log.Printf("📡 Subscribed ke channel: %s", channel)
	return nil
}

func (c *WebSocketClient) Publish(channel string, data interface{}) (string, error) {
	if !c.isConnected {
		return "", fmt.Errorf("WebSocket tidak terhubung")
	}

	messageID := uuid.New().String()
	ackChan := make(chan AckResponse, 1)

	c.acksMu.Lock()
	c.pendingAcks[messageID] = ackChan
	c.acksMu.Unlock()

	defer func() {
		c.acksMu.Lock()
		delete(c.pendingAcks, messageID)
		c.acksMu.Unlock()
		close(ackChan)
	}()

	msg := OutgoingMessage{
		Action:    "publish",
		Channel:   channel,
		Data:      data,
		MessageID: messageID,
	}

	c.mu.Lock()
	err := c.conn.WriteJSON(msg)
	c.mu.Unlock()

	if err != nil {
		return "", fmt.Errorf("gagal publish: %w", err)
	}

	select {
	case ack := <-ackChan:
		if ack.Status == "error_ack" {
			return "", fmt.Errorf("publish error: %s", ack.Error)
		}
		log.Printf("✅ Pesan berhasil di-publish ke %s (ID: %s)", channel, messageID)
		return messageID, nil
	case <-time.After(10 * time.Second):
		return "", fmt.Errorf("timeout menunggu acknowledgment")
	}
}

func (c *WebSocketClient) PublishAsync(channel string, data interface{}) error {
	if !c.isConnected {
		return fmt.Errorf("WebSocket tidak terhubung")
	}

	messageID := uuid.New().String()
	msg := OutgoingMessage{
		Action:    "publish",
		Channel:   channel,
		Data:      data,
		MessageID: messageID,
	}

	c.mu.Lock()
	err := c.conn.WriteJSON(msg)
	c.mu.Unlock()

	if err != nil {
		return fmt.Errorf("gagal publish async: %w", err)
	}

	log.Printf("📤 Pesan async di-publish ke %s", channel)
	return nil
}

func (c *WebSocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

func (c *WebSocketClient) Close() error {
	close(c.stopChan)
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
