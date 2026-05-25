package models

import "encoding/json"

// Event Types
const (
	EventTypeChat       = "CHAT"
	EventTypeScore      = "SCORE"
	EventTypeKillFeed   = "KILL"
)

// ServerEvent is the wrapper for all reliable messages
type ServerEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"` // Raw JSON to be parsed based on Type
}

// ChatPayload is the specific data for a chat message
type ChatPayload struct {
	SenderID string `json:"sender"`
	Message  string `json:"msg"`
}