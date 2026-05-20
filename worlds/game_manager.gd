extends Node

@onready var network_manager = $NetworkManager

func _ready():
	# Listen to the NetworkManager singleton signals cleanly
	network_manager.connected_to_server.connect(_on_server_connected)
	network_manager.data_received.connect(_on_network_data_received)

func _on_server_connected():
	print("Game logic layer initialized: Connected to server!")

func _on_network_data_received(payload: PackedByteArray):
	# Convert raw bytes back to string for interpretation
	var raw_string = payload.get_string_from_utf8()
	
	if raw_string.begins_with("SNAPSHOT"):
		# Route snapshot to your interpolation engine
		_process_world_snapshot(raw_string)

func _process_world_snapshot(data: String):
	print(data)
	pass
