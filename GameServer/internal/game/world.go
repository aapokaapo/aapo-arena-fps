package game

import (
	"log"
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

	History  *HistoryBuffer
	Recorder *MatchRecorder

	pendingReplayEvents   []models.ServerEvent // Events saved into replay/history
	pendingReliableEvents []models.ServerEvent // Events sent to live clients
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
		ID:            sessionID,
		PosX:          pos[0],
		PosY:          pos[1],
		PosZ:          pos[2],
		Health:        100,
		Shield:        100,
		LifeState:     models.LifeAlive,
		TeamState:     models.TeamFreeForAll,
		Locomotion:    models.LocomotionIdle,
		Posture:       models.PostureStanding,
		Vertical:      models.VerticalGrounded,
		Action:        models.ActionNone,
		Weapons:       [2]uint8{models.WeaponRifle, models.WeaponNone}, // Default weapons
		Ammo:          [2]uint16{15, 0},
		AmmoInReserve: [2]uint16{15, 0}, // Default ammo
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
	if !exists || player.LifeState == models.LifeDead {
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
		moveVector := [2]float32{input.MoveX, input.MoveY}
		wantsToSlide := (input.Buttons & 16) != 0
		wantsToJump := (input.Buttons & 32) != 0

		processPlayerPhysics(w, player, moveVector, wantsToJump, wantsToSlide, input.DeltaTime)

		// 3. Process Combat & Weapons (Delegated to combat.go)
		ProcessShooting(w, player, input, w.currentTick)

		// 4. Update the highest command ID processed
		player.LastProcessedCommandID = input.CommandID
	}
}

// Tick advances the server time by one frame
// This is called exactly 60 times a second by the broadcastLoop in transport.go
func (w *World) Tick() []models.ServerEvent {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentTick++
	w.currentTimestamp = time.Now().UnixMilli()

	// 1. Grab any events that happened in the last 16ms
	reliableEvents := w.pendingReliableEvents
	replayEvents := w.pendingReplayEvents
	w.pendingReliableEvents = nil
	w.pendingReplayEvents = nil

	// 2. Save to the ring buffer (now with events!)
	w.History.SaveSnapshot(w.currentTick, w.players, replayEvents)

	// 3. Queue it to be written to the SSD
	if w.Recorder != nil {
		w.Recorder.RecordTick(w.History.buffer[w.History.head])
	}

	return reliableEvents
}

func (w *World) AddEventToReplay(event models.ServerEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pendingReplayEvents = append(w.pendingReplayEvents, event)
}

func (w *World) QueueReliableEvent(event models.ServerEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pendingReliableEvents = append(w.pendingReliableEvents, event)
	w.pendingReplayEvents = append(w.pendingReplayEvents, event)
}
