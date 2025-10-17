package p2p

import ()

// Peer represents remote node.
type Peer interface {
	Close() error
}

type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
}
