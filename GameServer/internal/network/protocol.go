package network

import (
	"encoding/json"
	"fmt"

	// Replace this with your actual module path
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/models"
)

// WelcomeMessage is sent reliably on first connection
type WelcomeMessage struct {
	Type   string             `json:"type"`
	MyID   string             `json:"my_id"`
	Player models.PlayerState `json:"player"`
}

// SnapshotMessage is broadcasted 60 times a second to all clients
type SnapshotMessage struct {
	Tick      uint64               `json:"t"`
	Timestamp int64                `json:"ts"`
	Players   []models.PlayerState `json:"p"`
}

// SerializeWelcomeMessage packages the initial spawn data
func SerializeWelcomeMessage(sessionID string, player models.PlayerState) ([]byte, error) {
	msg := WelcomeMessage{
		Type:   "WELCOME",
		MyID:   sessionID,
		Player: player,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal welcome message: %w", err)
	}

	return data, nil
}

// SerializeSnapshot packages the entire world state for datagram broadcasting
func SerializeSnapshot(tick uint64, timestamp int64, players map[string]*models.PlayerState) ([]byte, error) {
	// Convert the map to a flat array for the client
	playerArray := make([]models.PlayerState, 0, len(players))
	for _, p := range players {
		playerArray = append(playerArray, *p)
	}

	snap := SnapshotMessage{
		Tick:      tick,
		Timestamp: timestamp,
		Players:   playerArray,
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return data, nil
}

// DeserializeClientInputPacket takes a raw datagram from the client and extracts the sub-tick inputs
func DeserializeClientInputPacket(data []byte) (models.ClientInputPacket, error) {
	var packet models.ClientInputPacket
	
	err := json.Unmarshal(data, &packet)
	if err != nil {
		return packet, fmt.Errorf("failed to unmarshal client input: %w", err)
	}

	return packet, nil
}
