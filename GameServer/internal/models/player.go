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
	LocomotionCrouching uint8 = 4
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

const (
	TeamSpectator uint8 = 0
	// In a Free-for-All mode, all players are on the same team and can shoot each other
	TeamFreeForAll uint8 = 1
	// In a Team Deathmatch mode, players are divided into teams
	TeamOne   uint8 = 2
	TeamTwo   uint8 = 3
	TeamThree uint8 = 4
	TeamFour  uint8 = 5
)

const (
	LifeAlive      uint8 = 0
	LifeDead       uint8 = 1
	LifeSpectating uint8 = 2
)

// The main player struct
type PlayerState struct {
	ID    string  `json:"id"`
	PosX  float32 `json:"x"`
	PosY  float32 `json:"y"`
	PosZ  float32 `json:"z"`
	VelY  float32 `json:"-"` // Vertical velocity (internal physics)
	Yaw   float32 `json:"yaw"`
	Pitch float32 `json:"pitch"`

	LifeState uint8 `json:"life"` // Alive or Dead or Spectating
	TeamState uint8 `json:"team"` // Team 1, Team 2, or Free-for-All

	// The 4 State Layers
	Posture    uint8 `json:"pos"`
	Locomotion uint8 `json:"loc"`
	Vertical   uint8 `json:"vert"`
	Action     uint8 `json:"act"`

	Health                 uint8  `json:"hp"`
	Shield                 uint8  `json:"shd"`
	LockedTicks            uint8  `json:"-"`    // Not sent to client
	LastProcessedCommandID uint64 `json:"lcid"` // The client NEEDS this!
	NextAttackTick         uint64 `json:"nat"`  // Helps the client predict shooting animations

	// Weapon State
	Weapons           [2]uint8  `json:"wep"`
	Ammo              [2]uint16 `json:"ammo"`
	AmmoInReserve     [2]uint16 `json:"ammore"`
	ActiveWeaponIndex uint8     `json:"awi"`
	IsAiming          bool      `json:"ads"`
	ADSLockTicks      uint64    `json:"-"` // Ticks left before they can toggle aiming again

	// Server-only killfeed and accuracy tracking
	KillStreak        uint8  `json:"-"`
	ConsecutiveHits   uint16 `json:"-"`
	ShotsSinceLastHit uint16 `json:"-"`
	LastHitVictimID   string `json:"-"`

	// Server-Internal Variables (Not sent to client)
	WasShootingLast bool  `json:"-"` // Tracks trigger-pulls for Semi-Auto
	BurstShotsLeft  uint8 `json:"-"` // Tracks active bursts

	// Tracks damage received since last spawn or health regeneration
	DamageTracker map[string]uint16 `json:"-"`
}
