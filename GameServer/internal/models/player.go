package models

// Posture
const (
	PostureStanding  uint8 = 0
	PostureCrouching uint8 = 1
)

// Locomotion
const (
	LocomotionIdle      uint8 = 0
	LocomotionRunning   uint8 = 1
	LocomotionSprinting uint8 = 2
	LocomotionSliding   uint8 = 3
)

// Vertical
const (
	VerticalGrounded uint8 = 0
	VerticalJumping  uint8 = 1
	VerticalFalling  uint8 = 2
)

// Action Overrides (Specific animations that lock out normal behavior)
const (
	ActionNone      uint8 = 0
	ActionReloading uint8 = 1
	ActionMantling  uint8 = 2
	ActionClimbing  uint8 = 3
)

// The main player struct
type PlayerState struct {
	ID          string  `json:"id"`
	PosX        float32 `json:"x"`
	PosY        float32 `json:"y"`
	PosZ        float32 `json:"z"`
	Yaw         float32 `json:"yaw"`
	Pitch       float32 `json:"pitch"`
	
	// The 4 State Layers
	Posture     uint8   `json:"pos"`
	Locomotion  uint8   `json:"loc"`
	Vertical    uint8   `json:"vert"`
	Action      uint8   `json:"act"`
	
	Health      uint8   `json:"hp"`
	LockedTicks uint8   `json:"-"` // Not sent to client
	LastProcessedCommandID uint64 `json:"lcid"` // The client NEEDS this!
	NextAttackTick         uint64 `json:"nat"`  // Helps the client predict shooting animations
	
	// Weapon State
	Weapons        [2]uint8  `json:"wep"`
	Ammo           [2]uint16 `json:"ammo"`
	AmmoInReserve  [2]uint16 `json:"ammore"`
	ActiveWeaponIndex uint8 `json:"awi"`
	IsAiming       bool   `json:"ads"`
	
	// Server-Internal Variables (Not sent to client)
	WasShootingLast bool   `json:"-"` // Tracks trigger-pulls for Semi-Auto
	BurstShotsLeft  uint8  `json:"-"` // Tracks active bursts
	LastProcessedCommandID uint64 `json:"-"` // Tracks the last sub-tick processed
}
