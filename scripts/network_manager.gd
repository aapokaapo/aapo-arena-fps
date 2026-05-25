extends Node

signal connected_to_server()
signal disconnected_from_server()
signal data_received(payload: PackedByteArray)

const CRT_PATH = "res://GameServer/server.crt"

var js_callback: JavaScriptObject
var cert_hash: Array = []

func _ready():
	if not OS.has_feature("web"):
		return

	if not FileAccess.file_exists(CRT_PATH):
		print("❌ Certificate file missing at path: ", CRT_PATH)
		return

	var file = FileAccess.open(CRT_PATH, FileAccess.READ)
	var crt_text = file.get_as_text()
	file.close()

	var base64_str = crt_text.replace("-----BEGIN CERTIFICATE-----", "")
	base64_str = base64_str.replace("-----END CERTIFICATE-----", "")
	base64_str = base64_str.replace("\n", "").replace("\r", "").strip_edges()

	var der_buffer = Marshalls.base64_to_raw(base64_str)
	if der_buffer.is_empty():
		print("❌ Failed to parse Base64 data from certificate!")
		return

	print("✅ DER buffer size: ", der_buffer.size(), " bytes")

	var ctx = HashingContext.new()
	ctx.start(HashingContext.HASH_SHA256)
	ctx.update(der_buffer)
	var raw_hash_bytes = ctx.finish()
	
	cert_hash = Array(raw_hash_bytes)
	print("✅ Certificate SHA-256 hash computed locally: ", cert_hash)
	
	var hex_str = ""
	for byte in raw_hash_bytes:
		hex_str += "%02x" % byte
	print("✅ Hash (hex): ", hex_str)

	var js_code = """
	var wtSession = null;
	var wtWriter = null;

	async function connectWebTransport(url, certHashArray) {
		try {
			if (!certHashArray || certHashArray.length === 0) {
				console.error("❌ Certificate hash not initialized!");
				if (window.onNetworkDisconnected) {
					window.onNetworkDisconnected();
				}
				return;
			}
			
			console.log("📋 Certificate hash received. Length:", certHashArray.length);
			
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
			console.log("✅ WebTransport Connected!");
			
			if (window.onNetworkConnected) {
				window.onNetworkConnected();
			}
			
			wtWriter = wtSession.datagrams.writable.getWriter();
			readDatagramLoop();
		} catch (error) {
			console.error("❌ WebTransport connection failed:", error);
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
				
				// Convert Uint8Array to Godot-compatible Array
				const dataArray = [];
				for (let i = 0; i < value.length; i++) {
					dataArray.push(value[i]);
				}
				console.log("📨 Sending datagram to Godot:", dataArray.length, "bytes");
				
				if (window.onDatagramReceived) {
					window.onDatagramReceived(dataArray);
				}
			} catch (error) {
				console.error("❌ Datagram read error:", error);
				break;
			}
		}
		console.log("❌ Datagram reader closed");
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

	js_callback = JavaScriptBridge.create_callback(_on_datagram_received)
	JavaScriptBridge.get_interface("window").onDatagramReceived = js_callback
	
	var connected_callback = JavaScriptBridge.create_callback(_on_connected)
	JavaScriptBridge.get_interface("window").onNetworkConnected = connected_callback
	
	var disconnected_callback = JavaScriptBridge.create_callback(_on_disconnected)
	JavaScriptBridge.get_interface("window").onNetworkDisconnected = disconnected_callback

	var hash_json = JSON.stringify(cert_hash)
	var connect_code = """
	var certHashArray = JSON.parse('%s');
	window.connectWebTransport('https://localhost:4433/fps', certHashArray);
	""" % hash_json
	
	JavaScriptBridge.eval(connect_code, true)


func _on_connected(_args):
	print("✅ Network layer connected!")
	connected_to_server.emit()

func _on_disconnected(_args):
	print("❌ Network layer disconnected!")
	disconnected_from_server.emit()

func _on_datagram_received(args):
	var js_array = args[0]
	
	if js_array == null:
		print("❌ Received null datagram")
		return
	
	var raw_bytes = PackedByteArray()
	
	# Iterate through the array safely
	var i = 0
	while i < 1024:  # Safety limit
		var byte_val = js_array[i]
		if byte_val == null:
			break
		raw_bytes.append(int(byte_val) & 0xFF)
		i += 1
	
	if raw_bytes.size() > 0:
		var message = raw_bytes.get_string_from_utf8()
		print("📨 Received %d bytes: %s" % [raw_bytes.size(), message])
		data_received.emit(raw_bytes)


func send_bytes(bytes: PackedByteArray) -> void:
	if OS.has_feature("web"):
		JavaScriptBridge.get_interface("window").sendDatagram(Array(bytes))

func send_string(message: String) -> void:
	send_bytes(message.to_utf8_buffer())
