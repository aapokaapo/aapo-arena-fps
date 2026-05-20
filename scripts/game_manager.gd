extends Node

@onready var network_manager = $NetworkManager

func _process_world_snapshot(data: String):
	print(data)
	pass


func _on_network_manager_data_received(payload: PackedByteArray) -> void:
	# Convert raw bytes back to string for interpretation
	var raw_string = payload.get_string_from_utf8()
	
	if raw_string.begins_with("SNAPSHOT"):
		# Route snapshot to your interpolation engine
		_process_world_snapshot(raw_string)


func _on_network_manager_connected_to_server() -> void:
	print("Game logic layer initialized: Connected to server!")


func _on_network_manager_disconnected_from_server() -> void:
	print("Game logic layer initialized: Disconnected from server!")
