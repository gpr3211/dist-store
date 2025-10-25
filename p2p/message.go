package p2p

const (
	// Message types (0x01-0x0F)
	IncomingMessage = 0x01 + iota
	IncomingStream
	IncomingVideoStream
	IncomingAudioStream
)

const (
	// Control messages (0xF0-0xFE)
	ControlPing = 0xF0 + iota
	ControlAck
	ControlClose
)

// RPC holds any arbirtrary data being sent over each transport b/w two nodes.
type RPC struct {
	Stream  bool
	From    string
	Payload []byte
}
