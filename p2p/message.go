package p2p

import "net"

// RPC holds any arbirtrary data being sent over each transport b/w two nodes.
type RPC struct {
	Payload []byte
	From    net.Addr
}
