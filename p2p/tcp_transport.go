package p2p

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type TCPTransportOpts struct {
	HandshakeFunc HandshakeFunc
	ListenAddr    string
	Decoder       Decoder
	OnPeer        func(Peer) error
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
	for _, v := range t.peers {
		// TODO:
		// add disc notify msg.
		v.Close()
	}
	return t.listener.Close()
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		Config:  opts,
		rpcChan: make(chan RPC, 5),
		peers:   make(map[net.Addr]Peer),
	}
}

// startAcceptLoop listens for new conns and send them to be handled.
// TODO:
// - add context.
func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return
		}
		go t.handleConn(conn, false)

	}
}

// Dial implements the Transport interface.
func (t *TCPTransport) Dial(adr string) error {

	conn, err := net.Dial("tcp", adr)

	if err != nil {
		return err
	}
	go t.handleConn(conn, true)

	return nil
}
func (t *TCPTransport) Addr() string {
	return t.Config.ListenAddr
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	peer := NewTCPPeer(conn, outbound)

	if !outbound {
		fmt.Printf("New incoming conn from %s\n", peer.RemoteAddr())
	} else {
		fmt.Printf("New incoming conn from %s\n", peer.LocalAddr())
	}
	var err error
	defer func() {
		fmt.Printf("dropping peer connection: %s\n", peer.RemoteAddr())
		conn.Close()
	}()

	if _, err := t.Config.HandshakeFunc(peer); err != nil {

		log.Printf("closing conn due to error %s: ", err.Error())
		return
	}
	if t.Config.OnPeer != nil {
		if err = t.Config.OnPeer(peer); err != nil {
			log.Printf("closing conn due to error %s: ", err.Error())
			return
		}
	}
	msg := RPC{}
	//read loop

	for {
		err := t.Config.Decoder.Decode(conn, &msg)
		if err == net.ErrClosed {
			return
		}
		if err != nil {
			switch err {
			case io.EOF:
				return
			}
			continue
		}

		msg.From = conn.RemoteAddr().String()
		if msg.Stream {
			peer.wg.Add(1)
			fmt.Printf("[%s] incoming stream, waiting...\n", conn.RemoteAddr())
			peer.wg.Wait()
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
			continue
		}
		t.rpcChan <- msg
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	ln, err := net.Listen("tcp", t.Config.ListenAddr)
	if err != nil {
		return err

	}
	t.listener = ln
	go t.startAcceptLoop()
	log.Println("Listening on port : ", t.listener.Addr())

	return nil
}
