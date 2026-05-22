package models

// PlayerInput represents a single frame of input from the Godot client
type PlayerInput struct {
	CommandID uint64  `json:"cid"`
	DeltaTime float32 `json:"dt"`
	MoveX     float32 `json:"mx"`
	MoveY     float32 `json:"my"`
	Pitch     float32 `json:"p"`
	Yaw       float32 `json:"y"`
	Buttons   uint8   `json:"b"`
}

// ClientInputPacket is the payload sent over WebTransport datagrams
type ClientInputPacket struct {
	LatestTick uint64        `json:"t"`
	Inputs     []PlayerInput `json:"i"`
}
