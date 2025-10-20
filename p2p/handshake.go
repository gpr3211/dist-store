package p2p

import "errors"

// ErrHandshakeInvalid is returned if the handshake
// beteween local and remote node could not be established.
var ErrHandshakeInvalid = errors.New("bad handshake")

var ErrHandshakeTimeout = errors.New("bad handshake")

type HandshakeFunc func(Peer) error

func NOPHandshakefunc(Peer) error {

	return nil
}
