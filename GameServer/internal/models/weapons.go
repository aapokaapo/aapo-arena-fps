package models

// Weapon IDs
const (
	WeaponNone       uint8 = 0
	WeaponPistol     uint8 = 1
	WeaponRifle      uint8 = 2
	WeaponBurstRifle uint8 = 3
	WeaponSniper     uint8 = 4
)

// Fire Modes
const (
	FireModeSemi  uint8 = 1
	FireModeAuto  uint8 = 2
	FireModeBurst uint8 = 3
)

type WeaponStats struct {
	LifeState    uint8 // Whether the weapon is currently active (alive) or not (dead)
	Damage       uint8
	ClipSize     uint16
	ReserveAmmo  uint16 // Total ammo available for the weapon (not including the current clip)
	ReloadTime   uint64 // Time in ticks to reload the weapon
	FireMode     uint8
	TicksPerShot uint64 // The fire rate (e.g., 60 Ticks = 1 shot per sec)
	BurstCount   uint8  // How many bullets per burst
}

// The global registry the server uses to validate shots
var WeaponRegistry = map[uint8]WeaponStats{
	WeaponPistol:     {Damage: 25, ClipSize: 12, FireMode: FireModeSemi, TicksPerShot: 20},
	WeaponRifle:      {Damage: 15, ClipSize: 15, FireMode: FireModeAuto, TicksPerShot: 65},
	WeaponBurstRifle: {Damage: 18, ClipSize: 24, FireMode: FireModeBurst, TicksPerShot: 5, BurstCount: 3},
	WeaponSniper:     {Damage: 90, ClipSize: 5, FireMode: FireModeSemi, TicksPerShot: 80},
}
