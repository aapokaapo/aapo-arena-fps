class_name Weapon
extends Node3D

@export_group("Weapon Stats")
@export var weapon_name: String = "Rifle"
@export var damage: float = 10.0
@export var fire_rate: float = 0.2 # Seconds between shots (lower is faster)
@export var max_ammo: int = 30
@export var range: float = 50.0

var current_ammo: int
var can_fire: bool = true
var fire_timer: Timer

func _ready() -> void:
	# Initialize ammo
	current_ammo = max_ammo
	
	# Programmatically create a timer to handle fire rate delays
	fire_timer = Timer.new()
	fire_timer.one_shot = true
	fire_timer.wait_time = fire_rate
	fire_timer.timeout.connect(_on_fire_timer_timeout)
	add_child(fire_timer)

# Virtual function: Intended to be overridden by child weapons
func fire() -> bool:
	if current_ammo > 0 and can_fire:
		current_ammo -= 1
		can_fire = false
		fire_timer.start()
		
		# (Visuals, sound, and hit detection will go here in specific weapons)
		print(weapon_name + " fired! Ammo: " + str(current_ammo) + "/" + str(max_ammo))
		return true 
	else:
		if current_ammo <= 0:
			print("Click! Out of ammo.")
		return false

func reload() -> void:
	current_ammo = max_ammo
	print(weapon_name + " reloaded.")

func _on_fire_timer_timeout() -> void:
	can_fire = true
