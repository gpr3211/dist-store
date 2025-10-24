package p2p

import (
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

func (p *TCPPeer) SetConn(c net.Conn) {
	p.Conn = c
}

// make sure we satisfy interfaces
var _ Peer = (*TCPPeer)(nil)

func (t TCPPeer) Close() error {
	return t.Conn.Close()
}
func (t *TCPPeer) Send(b []byte) error {
	_, err := t.Conn.Write(b)
	return err
}
func (t *TCPPeer) CloseStream() {
	t.wg.Done()
}

func NewTCPPeer(con net.Conn, out bool) *TCPPeer {

	peer := &TCPPeer{
		outbound: out,
		wg:       &sync.WaitGroup{},
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
