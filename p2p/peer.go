package p2p

import (
	"crypto/rand"
	"crypto/rsa"
	"net"
	"sync"
)

// TCPPeer represent remote node over a tcp connection.
type TCPPeer struct {
	net.Conn
	outbound bool // true if conn is outbound.
	pKey     rsa.PrivateKey
	wg       *sync.WaitGroup
}

// make sure we satisfy interfaces
var _ Peer = (*TCPPeer)(nil)

func (t TCPPeer) Close() error {
	return t.Close()
}
func (t *TCPPeer) Send([]byte) error { return nil }
func (t *TCPPeer) CloseStream()      {}
func NewTCPPeer(con net.Conn, out bool) *TCPPeer {

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	peer := &TCPPeer{
		outbound: out,
		wg:       &sync.WaitGroup{},
		pKey:     *privKey,
		Conn:     con,
	}

	return peer
}

func (p *TCPPeer) PublicKey() rsa.PublicKey {
	return p.pKey.PublicKey
}
func (p *TCPPeer) PrivateKey() rsa.PrivateKey {
	return p.pKey
}
