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
	if player.Vertical == models.VerticalGrounded {
		player.VelY = 0
		return
	}

	// Gravity is an acceleration (units/s^2). It affects velocity first.
	// Note: 20.0-30.0 often feels "snappier" in FPS games than the real-world 9.81.
	const gravityAccel float32 = 25.0
	player.VelY -= gravityAccel * deltaTime
	player.PosY += player.VelY * deltaTime

	// Basic ground check (assuming a flat plane at Y=0)
	if player.PosY <= 0 {
		player.PosY = 0
		player.VelY = 0
		player.Vertical = models.VerticalGrounded
	} else if player.VelY < 0 {
		player.Vertical = models.VerticalFalling
	}
}

func checkLedgeInFront(x, y, z float32) (float32, bool) {
	// Raycast logic here
	return 0, false
}

// applyPhysicsAndMovement applies movement, slide physics, and gravity to the player
// moveVector is a normalized direction vector (should be pre-normalized by input layer)
func applyPhysicsAndMovement(player *models.PlayerState, moveVector [2]float32, deltaTime float32) {
	// Apply gravity first (affects vertical velocity)
	applyGravity(player, deltaTime)

	// Apply slide physics if the player is sliding
	if player.Locomotion == models.LocomotionSliding {
		applySlidePhysics(player, deltaTime)
	} else {
		// Apply normal movement based on normalized input vector
		applyNormalMovement(player, moveVector, deltaTime)
	}
}

// applyNormalMovement handles regular movement when not sliding
// moveVector: [x, z] normalized direction vector from input layer (-1.0 to 1.0)
func applyNormalMovement(player *models.PlayerState, moveVector [2]float32, deltaTime float32) {
	// Movement speed varies by locomotion state
	var speed float32
	switch player.Locomotion {
	case models.LocomotionSprinting:
		speed = 7.0 // Units per second
	case models.LocomotionRunning:
		speed = 5.0
	case models.LocomotionCrouching:
		speed = 2.5
	default:
		speed = 0
	}

	// Apply movement from the normalized input vector
	player.PosX += moveVector[0] * speed * deltaTime
	player.PosZ += moveVector[1] * speed * deltaTime
}

func processPlayerPhysics(w *World, player *models.PlayerState, moveVector [2]float32, wantsToJump, wantsToSlide bool, deltaTime float32) {
	// 1. Decrement the lock if it's active
	if player.LockedTicks > 0 {
		player.LockedTicks--
	}

	// 2. Handle Slide Initiation
	if wantsToSlide && player.Locomotion == models.LocomotionSprinting {
		player.Locomotion = models.LocomotionSliding
		player.Posture = models.PostureCrouching

		// Lock the state for exactly the duration of the animation.
		// E.g., a 0.5 second slide at 60Hz = 30 ticks
		player.LockedTicks = 30
	}

	if wantsToJump && player.Vertical == models.VerticalGrounded {
		player.Vertical = models.VerticalJumping
		player.VelY = 10.0 // Initial upward impulse
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

	// 5. Apply Movement and Gravity
	applyPhysicsAndMovement(player, moveVector, deltaTime)
}
