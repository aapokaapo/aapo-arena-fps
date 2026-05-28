package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Event Types
const (
	EventTypeChat     = "CHAT"
	EventTypeScore    = "SCORE"
	EventTypeKillFeed = "KILL"
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

// KillFeedPayload is the specific data for killfeed and notification messages.
// It supports formatted text plus structured metadata for client-side rendering.
type KillFeedPayload struct {
	Message      string            `json:"msg"`
	AttackerID   string            `json:"attacker,omitempty"`
	VictimID     string            `json:"victim,omitempty"`
	WeaponID     uint8             `json:"weapon,omitempty"`
	WeaponName   string            `json:"weapon_name,omitempty"`
	Headshot     bool              `json:"headshot,omitempty"`
	Perfect      bool              `json:"perfect,omitempty"`
	Streak       uint8             `json:"streak,omitempty"`
	PingTarget   string            `json:"ping_target,omitempty"`
	PingLocation string            `json:"ping_location,omitempty"`
	DamageStats  map[string]uint16 `json:"damage_stats,omitempty"`
}

// NewKillFeedEvent builds a killfeed ServerEvent with a serialized payload.
func NewKillFeedEvent(attackerID, victimID string, weaponID uint8, headshot, perfect bool, streak uint8, message string, damageStats map[string]uint16) (ServerEvent, error) {
	if message == "" {
		weaponName := WeaponName(weaponID)
		if attackerID != "" && victimID != "" && weaponID != WeaponNone {
			message = fmt.Sprintf("%s %s %s", attackerID, weaponName, victimID)
			modifiers := []string{}
			if headshot {
				modifiers = append(modifiers, "headshot")
			}
			if perfect {
				modifiers = append(modifiers, "perfect")
			}
			if len(modifiers) > 0 {
				message = fmt.Sprintf("%s (%s)", message, strings.Join(modifiers, ", "))
			}
		} else if attackerID != "" {
			message = fmt.Sprintf("%s is on a killstreak!", attackerID)
		}
	}

	payload := KillFeedPayload{
		Message:     message,
		AttackerID:  attackerID,
		VictimID:    victimID,
		WeaponID:    weaponID,
		WeaponName:  WeaponName(weaponID),
		Headshot:    headshot,
		Perfect:     perfect,
		Streak:      streak,
		DamageStats: damageStats,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return ServerEvent{}, err
	}

	return ServerEvent{
		Type:    EventTypeKillFeed,
		Payload: payloadBytes,
	}, nil
}
