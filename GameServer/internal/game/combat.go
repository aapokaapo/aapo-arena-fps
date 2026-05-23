package game

import (
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/models"
)

// ProcessShooting validates the player's trigger pulls against server fire rates
func ProcessShooting(w *World, player *models.PlayerState, input models.PlayerInput, currentTick uint64) {
	stats := models.WeaponRegistry[player.EquippedWeapon]
	wantsToShoot := input.Buttons&2 != 0 // Assuming Bit 1 is the "Shoot" button

	// Handle automatic burst firing (player doesn't need to hold the button)
	if player.BurstShotsLeft > 0 && currentTick >= player.NextAttackTick {
		fireBullet(w, player, stats)
		player.BurstShotsLeft--
		player.NextAttackTick = currentTick + stats.TicksPerShot
		return
	}

	// Normal shooting logic
	if wantsToShoot && currentTick >= player.NextAttackTick && player.AmmoInClip > 0 {
		canFire := false

		switch stats.FireMode {
		case models.FireModeAuto:
			canFire = true // Just hold the button down
		case models.FireModeSemi:
			// Require the player to release and re-pull the trigger
			if !player.WasShootingLast {
				canFire = true
			}
		case models.FireModeBurst:
			if !player.WasShootingLast {
				canFire = true
				player.BurstShotsLeft = stats.BurstCount - 1 // -1 because we fire the first shot now
			}
		}

		if canFire {
			fireBullet(w, player, stats)
			player.AmmoInClip--
			player.NextAttackTick = currentTick + stats.TicksPerShot
		}
	}

	// Update the trigger state for the next sub-tick
	player.WasShootingLast = wantsToShoot
}

// fireBullet is where you would do raycasting or spawn projectiles
func fireBullet(w *World, player *models.PlayerState, stats models.WeaponStats) {
	// TODO: Perform a raycast forward using player.Pitch and player.Yaw
	// If it hits another player, call applyDamage() on them!
}

// applyDamage processes health reduction and descoping
func applyDamage(target *models.PlayerState, damage uint8, currentTick uint64) {
	if target.Health <= damage {
		target.Health = 0
	} else {
		target.Health -= damage
	}

	// The Descope Mechanic
	if target.IsAiming {
		target.IsAiming = false
		target.ADSLockTicks = currentTick + 15 // Lock out of ADS for 15 ticks
	}
}
