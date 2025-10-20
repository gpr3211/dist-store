package p2p

import "errors"

// ErrInvalidHandshake is returned if the handshake
// beteween local and remote node could not be established.
var ErrInvalidHandshake = errors.New("bad handshake")

type HandshakeFunc func(Peer) error

func NOPHandshakefunc(Peer) error {

	return nil
}
