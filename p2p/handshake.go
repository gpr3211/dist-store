package p2p

import "errors"
import "net"

// ErrHandshakeInvalid is returned if the handshake
// beteween local and remote node could not be established.
var ErrHandshakeInvalid = errors.New("bad handshake")

var ErrHandshakeTimeout = errors.New("bad handshake")
var ErrHandshakeHashMismatch = errors.New("handshake confirmation mismatch") // possible attack LOG

type HandshakeFunc func(Peer) (net.Conn, error)

func NOPHandshakefunc(Peer) (net.Conn, error) {

	return nil, nil
}
