package game

import (
	"sync"
	"time"

	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/models"
)

// Define some static spawn points for the arena
var spawnPoints = [][3]float32{
	{0.0, 10.0, 0.0},
	{50.0, 10.0, -20.0},
	{-50.0, 10.0, 20.0},
	{25.0, 10.0, 25.0},
}

// World represents the authoritative game simulation
type World struct {
	mu               sync.RWMutex
	players          map[string]*models.PlayerState
	currentTick      uint64
	currentTimestamp int64
	
	History          *HistoryBuffer
	Recorder         *MatchRecorder
	
	pendingEvents    []models.ServerEvent // NEW: Holding pen for events
}

// NewWorld initializes the game state, optionally starting the match recorder
func NewWorld(enableRecording bool, matchID string) *World {
	w := &World{
		players:          make(map[string]*models.PlayerState),
		History:          NewHistoryBuffer(),
		currentTick:      0,
		currentTimestamp: time.Now().UnixMilli(),
	}
	
	// Only initialize the recorder if the flag is true
	if enableRecording {
		recorder, err := NewMatchRecorder(matchID)
		if err == nil {
			w.Recorder = recorder
		} else {
			log.Printf("Failed to start recorder: %v", err)
		}
	} else {
		log.Println("⚠️ Match recording is DISABLED for this session.")
	}
	
	return w
}

// AddPlayer creates a new player entity and assigns them a spawn point
func (w *World) AddPlayer(sessionID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Pick a spawn point based on current player count (round-robin)
	spawnIndex := len(w.players) % len(spawnPoints)
	pos := spawnPoints[spawnIndex]

	newPlayer := &models.PlayerState{
		ID:             sessionID,
		PosX:           pos[0],
		PosY:           pos[1],
		PosZ:           pos[2],
		Health:         100,
		Locomotion:     models.LocomotionIdle,
		Posture:        models.PostureStanding,
		Vertical:       models.VerticalGrounded,
		Action:         models.ActionNone,
		EquippedWeapon: models.WeaponRifle, // Default weapon
		AmmoInClip:     30,
	}

	w.players[sessionID] = newPlayer
}

// RemovePlayer cleans up the entity when a client disconnects
func (w *World) RemovePlayer(sessionID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.players, sessionID)
}

// GetPlayer safely retrieves a player pointer (used by the welcome message)
func (w *World) GetPlayer(sessionID string) *models.PlayerState {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.players[sessionID]
}

// GetSnapshotData extracts the current state for protocol serialization
func (w *World) GetSnapshotData() (uint64, int64, map[string]*models.PlayerState) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Note: We return the map directly here because protocol.go iterates over it.
	// In a highly concurrent scenario with massive player counts, you might 
	// deep-copy this state to avoid race conditions during JSON marshaling.
	return w.currentTick, w.currentTimestamp, w.players
}

// ProcessClientInputs processes the array of sub-tick inputs received from a client datagram
func (w *World) ProcessClientInputs(sessionID string, packet models.ClientInputPacket) {
	w.mu.Lock()
	defer w.mu.Unlock()

	player, exists := w.players[sessionID]
	if !exists || player.Life == models.LifeDead {
		return // Ignore inputs from dead or missing players
	}

	// Loop through the micro-inputs to catch up to the client's predicted state
	for _, input := range packet.Inputs {
		// Drop redundant inputs we've already processed (due to UDP overlaps)
		if input.CommandID <= player.LastProcessedCommandID {
			continue
		}

		// Security: Clamp delta time to prevent speed hacks (max ~1/20th of a second)
		if input.DeltaTime > 0.05 {
			input.DeltaTime = 0.05
		}

		// 1. Update Look Angles
		player.Pitch = input.Pitch
		player.Yaw = input.Yaw

		// 2. Process Physics & Movement (Delegated to physics.go)
		// We pass the World instance so physics can check wall collisions if needed
		ApplyPlayerPhysics(w, player, input)

		// 3. Process Combat & Weapons (Delegated to combat.go)
		ProcessShooting(w, player, input, w.currentTick)

		// 4. Update the highest command ID processed
		player.LastProcessedCommandID = input.CommandID
	}
}

// Tick advances the server time by one frame
// This is called exactly 60 times a second by the broadcastLoop in transport.go
func (w *World) Tick() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentTick++
	w.currentTimestamp = time.Now().UnixMilli()

	// 1. Grab any events that happened in the last 16ms
	tickEvents := w.pendingEvents
	w.pendingEvents = nil // Clear the holding pen for the next tick

	// 2. Save to the ring buffer (now with events!)
	w.History.SaveSnapshot(w.currentTick, w.players, tickEvents)
	
	// 3. Queue it to be written to the SSD
	if w.Recorder != nil {
		w.Recorder.RecordTick(w.History.buffer[w.History.head])
	}
}

func (w *World) AddEventToReplay(event models.ServerEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pendingEvents = append(w.pendingEvents, event)
}