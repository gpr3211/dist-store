package p2p

import "net"

// Message holds any arbirtrary data being sent over each transport b/w two nodes.
type Message struct {
	Payload []byte
	From    net.Addr
}
