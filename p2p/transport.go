package p2p

import (
	"crypto/rsa"
	"net"
)

// Peer represents remote node.
type Peer interface {
	net.Conn
	Sec
	Close() error
}

// conn between peers
type Sec interface {
	PublicKey() rsa.PublicKey
	PrivateKey() rsa.PrivateKey
}

type Transport interface {
	ListenAndAccept() error
	Addr() string
	Consume() <-chan RPC
	Close() error
	Dial(string) error
}
