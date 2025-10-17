package p2p

import (
	"fmt"
	"net"
	"sync"
)

type TCPTransportOpts struct {
	HandshakeFunc HandshakeFunc
	ListenAddr    string
	Decoder       Decoder
	OnPeer        func(Peer) error
}

// TCPPeer represent remote node over a tcp connection.
type TCPPeer struct {
	outbound bool // true if conn is outbound.
	conn     net.Conn
}

func (t TCPPeer) Close() error {
	return t.conn.Close()
}

func NewTCPPeer(con net.Conn, out bool) *TCPPeer {
	return &TCPPeer{
		conn:     con,
		outbound: out,
	}
}

type TCPTransport struct {
	Config   TCPTransportOpts
	listener net.Listener
	peers    map[net.Addr]Peer
	rpcChan  chan RPC
	mu       *sync.RWMutex
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcChan
}
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		Config:  opts,
		rpcChan: make(chan RPC, 5),
		peers:   make(map[net.Addr]Peer),
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
	peer := NewTCPPeer(conn, true)

	fmt.Printf("New incoming conn %+v\n", peer.conn)
	var err error
	defer func() {
		fmt.Printf("dropping peer connection: %s", err)
		conn.Close()

	}()

	// HANDLE !! TODO:
	if err := t.Config.HandshakeFunc(conn); err != nil {
		return

	}
	if t.Config.OnPeer != nil {
		if err = t.Config.OnPeer(peer); err != nil {
			return

		}
	}
	msg := RPC{}

	// READLOOP

	for {
		err := t.Config.Decoder.Decode(conn, &msg)
		if err == net.ErrClosed {
			return
		}
		if err != nil {
			fmt.Printf("TCP ERR: %s\n", err)
			continue
		}

		msg.From = conn.RemoteAddr()
		fmt.Printf("%s: message: %+v\n", msg.From, string(msg.Payload))
		t.rpcChan <- msg
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
