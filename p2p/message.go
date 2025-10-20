package p2p

import "net"

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// RPC holds any arbirtrary data being sent over each transport b/w two nodes.
type RPC struct {
	Stream  bool
	From    net.Addr
	Payload []byte
}
