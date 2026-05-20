extends Node

# Signals to expose the network events to the rest of the game architecture
signal connected_to_server()
signal disconnected_from_server()
signal data_received(payload: PackedByteArray)

var js_callback: JavaScriptObject
var cert_hash: Array = [73, 8, 85, 136, 235, 121, 141, 49, 240, 187, 55, 35, 172, 96, 89, 251, 134, 248, 191, 67, 220, 10, 118, 27, 152, 122, 213, 92, 217, 4, 207, 225]

func _ready():
	# If we are not running in a browser, do not execute
	if not OS.has_feature("web"):
		return

	# 1. Inject the pure WebTransport JavaScript pipeline
	var js_code = """
	var wtSession = null;
	var wtWriter = null;

	async function connectWebTransport(url, certHashArray) {
		try {
			const certHash = new Uint8Array(certHashArray);
			
			wtSession = new WebTransport(url, {
				serverCertificateHashes: [
					{
						algorithm: "sha-256",
						value: certHash
					}
				]
			});
			
			await wtSession.ready;
			
			// Notify Godot that the UDP tunnel is wide open
			if (window.onNetworkConnected) {
				window.onNetworkConnected();
			}
			
			wtWriter = wtSession.datagrams.writable.getWriter();
			readDatagramLoop();
		} catch (error) {
			if (window.onNetworkDisconnected) {
				window.onNetworkDisconnected();
			}
		}
	}

	async function readDatagramLoop() {
		const reader = wtSession.datagrams.readable.getReader();
		while (true) {
			try {
				const { value, done } = await reader.read();
				if (done) break;
				
				if (window.onDatagramReceived) {
					window.onDatagramReceived(Array.from(value));
				}
			} catch (error) {
				break;
			}
		}
		if (window.onNetworkDisconnected) {
			window.onNetworkDisconnected();
		}
	}

	function sendDatagram(payloadArray) {
		if (wtWriter) {
			wtWriter.write(new Uint8Array(payloadArray));
		}
	}
	"""
	JavaScriptBridge.eval(js_code, true)

	# 2. Bind JS lifecycle events back to internal GDScript methods
	js_callback = JavaScriptBridge.create_callback(_on_datagram_received)
	JavaScriptBridge.get_interface("window").onDatagramReceived = js_callback
	
	var connected_callback = JavaScriptBridge.create_callback(_on_connected)
	JavaScriptBridge.get_interface("window").onNetworkConnected = connected_callback
	
	var disconnected_callback = JavaScriptBridge.create_callback(_on_disconnected)
	JavaScriptBridge.get_interface("window").onNetworkDisconnected = disconnected_callback

	# 3. Open connection pipeline
	JavaScriptBridge.get_interface("window").connectWebTransport("https://127.0.0.1:4433/game-server", cert_hash)


# --- INTERNAL BRIDGE ROUTERS ---

func _on_connected(_args):
	connected_to_server.emit()

func _on_disconnected(_args):
	disconnected_from_server.emit()

func _on_datagram_received(args):
	var js_array = args[0]
	var raw_bytes = PackedByteArray()
	
	# Pack the data efficiently into native memory
	for i in range(len(js_array)):
		raw_bytes.append(js_array[i])
		
	# Emit the raw bytes out to the game logic
	data_received.emit(raw_bytes)


# --- PUBLIC API METHODS (Callable by other scripts) ---

## Sends a raw byte array down the WebTransport tunnel
func send_bytes(bytes: PackedByteArray) -> void:
	if OS.has_feature("web"):
		JavaScriptBridge.get_interface("window").sendDatagram(Array(bytes))

## Utility method: Converts a string to bytes and transmits it
func send_string(message: String) -> void:
	send_bytes(message.to_utf8_buffer())
