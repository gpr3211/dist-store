package p2p

import (
	"fmt"
	"net"
	"sync"
)

type TCPTransport struct {
	listenAddress string
	listener      net.Listener
	peers         map[net.Addr]Peer
	mu            *sync.RWMutex
}

// TCPPeer represent remote node over a tcp connection.
type TCPPeer struct {
	outbound bool // true if conn is outbound.
	conn     net.Conn
}

func NewTCPPeer(con net.Conn, out bool) *TCPPeer {
	return &TCPPeer{
		conn:     con,
		outbound: out,
	}
}

func NewTCPTransport(addr string) *TCPTransport {
	return &TCPTransport{
		listenAddress: addr,
	}
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Println("TCP accept err: ", err)
		}
		go t.handleConn(conn)

	}
}
func (t *TCPTransport) handleConn(conn net.Conn) {
	fmt.Printf("New incoming conn %+v\n", conn)
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.listenAddress)
	if err != nil {
		return err

	}
	t.listener = ln
	t.startAcceptLoop()

	return nil
}
