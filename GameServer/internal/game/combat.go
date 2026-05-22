package game

func applyDamage(target *PlayerState, damage uint8, currentTick uint64) {
	if target.Health <= damage {
		target.Health = 0
		target.Life = LifeDead
	} else {
		target.Health -= damage
	}

	// The Descope Mechanic
	if target.IsAiming {
		target.IsAiming = false
		
		// Lock them out of ADS for a short duration (e.g., 15 ticks / 250ms)
		// You will need to add ADSLockoutTick to your PlayerState struct
		target.ADSLockoutTick = currentTick + 15
	}
}
