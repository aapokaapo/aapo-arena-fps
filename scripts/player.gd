extends CharacterBody3D

# --- Speeds & Acceleration ---
const WALK_SPEED = 5.0
const SPRINT_SPEED = 8.0
const CROUCH_SPEED = 2.5
const JUMP_VELOCITY = 4.5

const GROUND_ACCEL = 10.0
const AIR_ACCEL = 3.0

# --- Slide Variables ---
const SLIDE_INITIAL_SPEED = 10.0
const SLIDE_FRICTION = 3.0
var current_slide_speed = 0.0
var slide_direction = Vector3.ZERO

# --- State Machine Setup ---
enum State { IDLE, WALK, SPRINT, CROUCH, SLIDE, AIR }
var current_state = State.IDLE

var gravity = ProjectSettings.get_setting("physics/3d/default_gravity")

# --- Nodes ---
@onready var head = $Head
@onready var collision_shape = $CollisionShape3D
@onready var mesh_shape = $MeshInstance3D

const MOUSE_SENSITIVITY = 0.002

func _ready():
	Input.set_mouse_mode(Input.MOUSE_MODE_CAPTURED)

func _unhandled_input(event):
	if event is InputEventMouseMotion:
		rotate_y(-event.relative.x * MOUSE_SENSITIVITY)
		head.rotate_x(-event.relative.y * MOUSE_SENSITIVITY)
		head.rotation.x = clamp(head.rotation.x, deg_to_rad(-80), deg_to_rad(80))

func _physics_process(delta):
	if not is_on_floor():
		velocity.y -= gravity * delta

	var input_dir = Input.get_vector("move_left", "move_right", "move_forward", "move_backward")
	var direction = (transform.basis * Vector3(input_dir.x, 0, input_dir.y)).normalized()

	# --- CHANGED: We now pass input_dir into our state machine too ---
	handle_state_transitions(direction, input_dir)
	handle_movement(direction, delta)
	handle_crouch_animation(delta)

	move_and_slide()

	if Input.is_action_just_pressed("ui_cancel"):
		Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)

# --- State Machine Logic ---
func handle_state_transitions(direction: Vector3, input_dir: Vector2):
	var was_sliding = (current_state == State.SLIDE)
	
	# In Godot's 2D input vectors, moving Forward pushes the Y axis negative.
	# We use -0.1 instead of 0.0 to account for slight analog stick drift on controllers.
	var is_moving_forward = input_dir.y < -0.1 

	if not is_on_floor():
		current_state = State.AIR
	elif Input.is_action_pressed("crouch"):
		if current_state == State.SPRINT and direction != Vector3.ZERO:
			current_state = State.SLIDE
			slide_direction = direction
			current_slide_speed = SLIDE_INITIAL_SPEED
		elif current_state == State.SLIDE and current_slide_speed > CROUCH_SPEED:
			current_state = State.SLIDE
		else:
			current_state = State.CROUCH
			
	# --- CHANGED: Sticky Sprint & Directional Constraint ---
	# We sprint if we hit Shift OR if we are already sprinting, BUT ONLY if moving forward/diagonally
	elif (Input.is_action_pressed("sprint") or current_state == State.SPRINT) and is_moving_forward:
		current_state = State.SPRINT
		
	elif direction != Vector3.ZERO:
		current_state = State.WALK
	else:
		current_state = State.IDLE

	if Input.is_action_just_pressed("jump") and is_on_floor() and current_state != State.SLIDE:
		velocity.y = JUMP_VELOCITY
		current_state = State.AIR

# --- Movement Execution ---
func handle_movement(direction: Vector3, delta: float):
	var target_speed = 0.0
	var accel = GROUND_ACCEL

	match current_state:
		State.IDLE:
			target_speed = 0.0
		State.WALK:
			target_speed = WALK_SPEED
		State.SPRINT:
			target_speed = SPRINT_SPEED
		State.CROUCH:
			target_speed = CROUCH_SPEED
		State.AIR:
			target_speed = WALK_SPEED # You can change this to an AIR_SPEED variable if you prefer
			accel = AIR_ACCEL
		State.SLIDE:
			# Sliding loses speed over time and overrides normal input direction
			current_slide_speed = lerp(current_slide_speed, 0.0, SLIDE_FRICTION * delta)
			velocity.x = slide_direction.x * current_slide_speed
			velocity.z = slide_direction.z * current_slide_speed
			return # Skip the standard lerp below so we don't interfere with the slide

	# Standard Movement Lerp for non-slide states
	var target_velocity_x = direction.x * target_speed
	var target_velocity_z = direction.z * target_speed
	
	velocity.x = lerp(velocity.x, target_velocity_x, accel * delta)
	velocity.z = lerp(velocity.z, target_velocity_z, accel * delta)

# --- Visuals & Hitbox ---
func handle_crouch_animation(delta: float):
	var target_head_y = 0.5
	var target_capsule_height = 2.0
	
	if current_state == State.CROUCH or current_state == State.SLIDE:
		target_head_y = -0.1  # Lower the camera
		target_capsule_height = 1.2 # Shrink the hitbox
		
	# Smoothly interpolate both the camera height and the physics hitbox
	head.position.y = lerp(head.position.y, target_head_y, delta * 10.0)
	collision_shape.shape.height = lerp(collision_shape.shape.height, target_capsule_height, delta * 10.0)
	mesh_shape.mesh.height = lerp(collision_shape.shape.height, target_capsule_height, delta * 10.0)
