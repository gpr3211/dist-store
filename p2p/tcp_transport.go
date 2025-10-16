package p2p

import (
	"fmt"
	"log"
	"net"
	"sync"
)

type TCPTransportOpts struct {
	HandshakeFunc HandshakeFunc
	ListenAddr    string
	Decoder       Decoder
}

type TCPTransport struct {
	Config   TCPTransportOpts
	listener net.Listener
	peers    map[net.Addr]Peer
	mu       *sync.RWMutex
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

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {

	return &TCPTransport{
		Config: opts,
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

type Temp struct{}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)

	fmt.Printf("New incoming conn %+v\n", peer.conn)
	// HANDLE !! TODO:
	if err := t.Config.HandshakeFunc(conn); err != nil {
		conn.Close()
		log.Println(ErrInvalidHandshake)
		return

	}
	//msg := &Message{}

	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("TCP ERR: %s\n", err)
		}
		fmt.Printf("message: %+v\n", string(buf[:n]))
		//	if err := t.Config.Decoder.Decode(conn, msg); err != nil {
		//		fmt.Printf("TCP ERR: %s\n", err)
		//		continue
		//	}

	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.Config.ListenAddr)
	if err != nil {
		return err

	}
	t.listener = ln
	t.startAcceptLoop()

	return nil
}
