package game

import (
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/models"
)

const HistorySize = 60 // 1 second of history at 60Hz

// HistoricState represents a frozen moment in time
// Inside internal/game/history.go

type HistoricState struct {
	Tick    uint64                           `json:"t"`
	Players map[string]models.PlayerState    `json:"p"`
	Events  []models.ServerEvent             `json:"e,omitempty"`
}

// HistoryBuffer manages lag compensation and recording
type HistoryBuffer struct {
	buffer [HistorySize]HistoricState
	head   int // Tracks the current write index
}

func NewHistoryBuffer() *HistoryBuffer {
	return &HistoryBuffer{}
}

// SaveSnapshot takes the current world and saves a deep copy into the ring buffer
// Update the signature to accept events
func (hb *HistoryBuffer) SaveSnapshot(tick uint64, activePlayers map[string]*models.PlayerState, events []models.ServerEvent) {
	hb.head = (hb.head + 1) % HistorySize

	stateCopy := make(map[string]models.PlayerState, len(activePlayers))
	for id, p := range activePlayers {
		stateCopy[id] = *p 
	}

	hb.buffer[hb.head] = HistoricState{
		Tick:    tick,
		Players: stateCopy,
		Events:  events, // Attach the events to this specific slice of time
	}
}

// GetStateAtTick searches backwards in time to find the exact hitboxes the client saw
func (hb *HistoryBuffer) GetStateAtTick(targetTick uint64) (HistoricState, bool) {
	// Look through the buffer (backwards is usually faster)
	for i := 0; i < HistorySize; i++ {
		if hb.buffer[i].Tick == targetTick {
			return hb.buffer[i], true
		}
	}
	// If the tick is too old (e.g., Ping > 1000ms), we reject the rewind
	return HistoricState{}, false
}