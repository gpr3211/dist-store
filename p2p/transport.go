package p2p

// Peer represents remote node.
type Peer interface{}

type Transport interface {
	ListenAndAccept() error
}
