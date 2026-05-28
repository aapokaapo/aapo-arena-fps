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
	LifeState        uint8 // Whether the weapon is currently active (alive) or not (dead)
	Damage           uint8
	ClipSize         uint16
	ReserveAmmo      uint16 // Total ammo available for the weapon (not including the current clip)
	ReloadTime       uint64 // Time in ticks to reload the weapon
	FireMode         uint8
	TicksPerShot     uint64 // The fire rate (e.g., 60 Ticks = 1 shot per sec)
	BurstCount       uint8  // How many bullets per burst
	PerfectKillCount uint8  // Minimum shots to kill from full health
}

// The global registry the server uses to validate shots
var WeaponRegistry = map[uint8]WeaponStats{
	WeaponPistol:     {Damage: 25, ClipSize: 12, FireMode: FireModeSemi, TicksPerShot: 30, PerfectKillCount: 4},
	WeaponRifle:      {Damage: 30, ClipSize: 15, FireMode: FireModeAuto, TicksPerShot: 40, PerfectKillCount: 4},
	WeaponBurstRifle: {Damage: 18, ClipSize: 24, FireMode: FireModeBurst, TicksPerShot: 5, BurstCount: 3, PerfectKillCount: 6},
	WeaponSniper:     {Damage: 100, ClipSize: 5, FireMode: FireModeSemi, TicksPerShot: 80, PerfectKillCount: 1},
}

// WeaponName returns a display-friendly name for a weapon identifier.
func WeaponName(weaponID uint8) string {
	switch weaponID {
	case WeaponPistol:
		return "Pistol"
	case WeaponRifle:
		return "Rifle"
	case WeaponBurstRifle:
		return "Burst Rifle"
	case WeaponSniper:
		return "Sniper"
	default:
		return "Weapon"
	}
}
