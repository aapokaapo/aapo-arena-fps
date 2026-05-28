package game

import (
	"fmt"

	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/models"
)

// ProcessShooting validates the player's trigger pulls against server fire rates
func ProcessShooting(w *World, player *models.PlayerState, input models.PlayerInput, currentTick uint64) {
	stats := models.WeaponRegistry[player.Weapons[player.ActiveWeaponIndex]]
	wantsToShoot := input.Buttons&2 != 0 // Assuming Bit 1 is the "Shoot" button

	// Handle automatic burst firing (player doesn't need to hold the button)
	if player.BurstShotsLeft > 0 && currentTick >= player.NextAttackTick {
		fireBullet(w, player, stats)
		player.BurstShotsLeft--
		player.NextAttackTick = currentTick + stats.TicksPerShot
		return
	}

	// Normal shooting logic
	if wantsToShoot && currentTick >= player.NextAttackTick && player.Ammo[player.ActiveWeaponIndex] > 0 {
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
			player.Ammo[player.ActiveWeaponIndex]--
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
// If the target dies, it also queues a killfeed event.
func applyDamage(w *World, attacker, target *models.PlayerState, damage uint8, currentTick uint64, weaponID uint8, wasHeadshot bool) {
	stats := models.WeaponRegistry[weaponID]

	if target.DamageTracker == nil {
		target.DamageTracker = make(map[string]uint16)
	}

	// 1. Track consecutive hits on the same player.
	// Reset count if the attacker hit someone else previously.
	if attacker.LastHitVictimID != target.ID {
		attacker.ConsecutiveHits = 0
		attacker.LastHitVictimID = target.ID
	}
	attacker.ConsecutiveHits++

	target.DamageTracker[attacker.ID] += uint16(damage)

	if target.Health <= damage {
		target.Health = 0
		attacker.KillStreak++

		// 2. Determine if it's a Perfect Kill:
		// - Hit count matches the minimum required for the weapon.
		// - No one else dealt damage (DamageTracker size is 1).
		isPerfect := (attacker.ConsecutiveHits == uint16(stats.PerfectKillCount)) && (len(target.DamageTracker) == 1)

		killEvent, err := models.NewKillFeedEvent(attacker.ID, target.ID, weaponID, wasHeadshot, isPerfect, attacker.KillStreak, "", target.DamageTracker)
		if err == nil {
			w.QueueReliableEvent(killEvent)
		}

		if attacker.KillStreak >= 3 {
			streakMessage := fmt.Sprintf("%s is on a killstreak!", attacker.ID)
			streakEvent, err := models.NewKillFeedEvent(attacker.ID, "", weaponID, false, false, attacker.KillStreak, streakMessage, nil)
			if err == nil {
				w.QueueReliableEvent(streakEvent)
			}
		}

		target.KillStreak = 0
		target.ConsecutiveHits = 0
		target.ShotsSinceLastHit = 0
		target.DamageTracker = nil // Reset for next life
	} else {
		target.Health -= damage
		target.ShotsSinceLastHit = 0
	}

	// The Descope Mechanic
	if target.IsAiming {
		target.IsAiming = false
		target.ADSLockTicks = currentTick + 15 // Lock out of ADS for 15 ticks
	}
}

// ApplyHealing increases player health and clears the damage tracker.
// Use this for health packs or passive regeneration.
func ApplyHealing(target *models.PlayerState, amount uint8) {
	if amount == 0 {
		return
	}

	newHealth := uint16(target.Health) + uint16(amount)
	if newHealth > 100 {
		newHealth = 100
	}
	target.Health = uint8(newHealth)

	// Reset damage history since the player has regenerated
	target.DamageTracker = nil
}
