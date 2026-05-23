package game

import (
	"github.com/aapokaapo/aapo-arena-fps/GameServer/internal/models"
)

// A helper function to process sliding
func applySlidePhysics(player *models.PlayerState, deltaTime float32) {
	// If the player is sliding, ignore their normal WASD inputs
	// and move them forward based on their current momentum

	player.PosX += 1.5 * deltaTime // Example math

	// You can also access global variables or server structs defined in main.go
}

func applyGravity(player *models.PlayerState, deltaTime float32) {
	// If the player is not grounded, apply gravity
	if player.Vertical != models.VerticalGrounded {
		player.PosY -= 9.81 * deltaTime
	}
}

func checkLedgeInFront(x, y, z float32) (float32, bool) {
	// Raycast logic here
	return 0, false
}

func processPlayerInput(_ *World, player *models.PlayerState, input models.PlayerInput) {
	// 1. Decrement the lock if it's active
	if player.LockedTicks > 0 {
		player.LockedTicks--
	}

	// 2. Handle Slide Initiation
	// Let's say bit 4 is "Slide Request"
	if input.Buttons&16 != 0 && player.Locomotion == models.LocomotionSprinting {
		player.Locomotion = models.LocomotionSliding
		player.Posture = models.PostureCrouching

		// Lock the state for exactly the duration of the animation.
		// E.g., a 0.5 second slide at 60Hz = 30 ticks
		player.LockedTicks = 30
	}
	// bit 5 is "Jump"
	if input.Buttons&32 != 0 && player.Vertical != models.VerticalGrounded {
		return
		// TODO: Raycast logic that checks for a ledge and if it can be climbed or mantled
	}

	// 3. Block illegal transitions if locked
	if player.LockedTicks > 0 {
		// Ignore inputs that try to break the slide
		// The player continues moving based on slide friction/momentum rules
	} else {
		// 4. Slide is over, resume normal locomotion logic
		if player.Locomotion == models.LocomotionSliding {
			player.Locomotion = models.LocomotionRunning // Default back to running
			player.Posture = models.PostureStanding
		}

		// Normal input processing for sprint/crouch goes here...
	}
}
