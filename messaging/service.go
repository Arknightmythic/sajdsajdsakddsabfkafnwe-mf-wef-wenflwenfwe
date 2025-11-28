package messaging

import (
	"dokuprime-be/config"
	"dokuprime-be/external"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MessageService struct {
	db             *sqlx.DB
	wsClient       *config.WebSocketClient
	externalClient *external.Client
}

func NewMessageService(db *sqlx.DB, wsURL, wsToken string, externalClient *external.Client) *MessageService {
	return &MessageService{
		db:             db,
		wsClient:       config.NewWebSocketClient(wsURL, wsToken),
		externalClient: externalClient,
	}
}

func (s *MessageService) CreateChatHistory(sessionID uuid.UUID, messageData map[string]interface{}, startTimestamp string) (int, error) {
	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal message data: %w", err)
	}

	query := `
		INSERT INTO chat_history (session_id, message, start_timestamp)
		VALUES ($1, $2::jsonb, $3)
		RETURNING id
	`

	var id int
	err = s.db.QueryRow(query, sessionID, messageJSON, startTimestamp).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert chat history: %w", err)
	}

	return id, nil
}

func (s *MessageService) CreateUserMessage(sessionID uuid.UUID, content string, startTimestamp string) (map[string]interface{}, int, error) {
	messageData := map[string]interface{}{
		"data": map[string]interface{}{
			"id":                nil,
			"name":              nil,
			"type":              "human",
			"content":           content,
			"additional_kwargs": map[string]interface{}{},
			"response_metadata": map[string]interface{}{},
		},
		"type": "human",
	}

	id, err := s.CreateChatHistory(sessionID, messageData, startTimestamp)
	return messageData, id, err
}

func (s *MessageService) CreateAgentMessage(sessionID uuid.UUID, content string, startTimestamp string) (map[string]interface{}, int, error) {
	messageData := map[string]interface{}{
		"data": map[string]interface{}{
			"id":                 nil,
			"name":               nil,
			"type":               "ai",
			"content":            content,
			"tool_calls":         []interface{}{},
			"usage_metadata":     map[string]interface{}{},
			"additional_kwargs":  map[string]interface{}{},
			"response_metadata":  map[string]interface{}{},
			"invalid_tool_calls": []interface{}{},
		},
		"type": "ai",
	}

	id, err := s.CreateChatHistory(sessionID, messageData, startTimestamp)
	return messageData, id, err
}

func (s *MessageService) PublishToChannel(channel string, data map[string]interface{}) error {
	if !s.wsClient.IsConnected() {
		log.Println("WebSocket not connected, attempting to reconnect...")
		if err := s.wsClient.Connect(); err != nil {
			return fmt.Errorf("failed to connect to websocket: %w", err)
		}
	}

	if err := s.wsClient.Publish(channel, data); err != nil {
		log.Printf("Failed to publish to channel %s: %v", channel, err)
		return fmt.Errorf("failed to publish to channel: %w", err)
	}

	log.Printf("✅ Published message to channel: %s", channel)
	return nil
}

type HelpdeskMessageResponse struct {
	User             string      `json:"user"`
	ConversationID   string      `json:"conversation_id"`
	Query            string      `json:"query"`
	Answer           string      `json:"answer"`
	Platform         string      `json:"platform"`
	PlatformUniqueID *string     `json:"platform_unique_id"`
	IsHelpdesk       bool        `json:"is_helpdesk"`
	Metadata         interface{} `json:"metadata,omitempty"`
}

func (s *MessageService) HandleHelpdeskMessage(sessionID uuid.UUID, message string, userType string, platform string, platformUniqueID *string, startTimestamp string) error {
	var chatHistoryID int
	var err error

	switch userType {
	case "user":
		_, chatHistoryID, err = s.CreateUserMessage(sessionID, message, startTimestamp)
	case "agent":
		_, chatHistoryID, err = s.CreateAgentMessage(sessionID, message, startTimestamp)
	default:
		return fmt.Errorf("invalid user_type: %s", userType)
	}

	if err != nil {
		return fmt.Errorf("failed to create chat history: %w", err)
	}

	log.Printf("💾 Saved message to database with ID: %d", chatHistoryID)

	messageUID := fmt.Sprintf("%s-%s-%d", sessionID.String(), userType, chatHistoryID)

	publishData := map[string]interface{}{
		"message_uid":        messageUID,
		"chat_history_id":    chatHistoryID,
		"session_id":         sessionID.String(),
		"message":            message,
		"user_type":          userType,
		"timestamp":          time.Now().Unix(),
		"platform":           platform,
		"platform_unique_id": platformUniqueID,
	}

	if platform == "web" {
		mainChannel := sessionID.String()
		agentChannel := sessionID.String() + "-agent"

		if err := s.PublishToChannel(mainChannel, publishData); err != nil {
			log.Printf("Failed to publish to main channel: %v", err)
		}

		if err := s.PublishToChannel(agentChannel, publishData); err != nil {
			log.Printf("Failed to publish to agent channel: %v", err)
		}

		log.Printf("✅ Published to both channels: %s and %s", mainChannel, agentChannel)
	} else {

		if userType == "agent" {

			userID := sessionID.String()
			if platformUniqueID != nil && *platformUniqueID != "" {
				userID = *platformUniqueID
			}

			response := HelpdeskMessageResponse{
				User:             userID,
				ConversationID:   sessionID.String(),
				Query:            "",
				Answer:           message,
				Platform:         platform,
				PlatformUniqueID: platformUniqueID,
				IsHelpdesk:       true,
			}

			if err := s.externalClient.SendMessageToAPI(response); err != nil {
				log.Printf("❌ Failed to send message to external API: %v", err)
				return fmt.Errorf("failed to send message to external API: %w", err)
			}

			log.Printf("✅ Sent agent message to external API for platform: %s", platform)
		} else {

			agentChannel := sessionID.String() + "-agent"
			if err := s.PublishToChannel(agentChannel, publishData); err != nil {
				log.Printf("Failed to publish user message to agent channel: %v", err)
			}
			log.Printf("✅ Published user message to agent channel: %s", agentChannel)
		}
	}

	return nil
}

func (s *MessageService) SubscribeToHelpdeskChannels(sessionID string) error {
	userChannel := sessionID
	if err := s.wsClient.Subscribe(userChannel, "$"); err != nil {
		return fmt.Errorf("failed to subscribe to user channel: %w", err)
	}

	agentChannel := sessionID + "-agent"
	if err := s.wsClient.Subscribe(agentChannel, "$"); err != nil {
		return fmt.Errorf("failed to subscribe to agent channel: %w", err)
	}

	s.wsClient.OnMessage(userChannel, func(data json.RawMessage) {
		log.Printf("📨 User channel %s: %s", userChannel, string(data))
	})

	s.wsClient.OnMessage(agentChannel, func(data json.RawMessage) {
		log.Printf("📨 Agent channel %s: %s", agentChannel, string(data))
	})

	log.Printf("✅ Subscribed to helpdesk channels for session: %s", sessionID)
	return nil
}
