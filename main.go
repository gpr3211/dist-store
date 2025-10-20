package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gpr3211/dist-store/p2p"
	"github.com/gpr3211/dist-store/server"
)

type Config struct {
	addr    string
	FServer *server.FileServer
	secret  []byte
}

func OnPeer(p2p.Peer) error {
	fmt.Println("Doing soem logic outside of transport tcp blah ")
	return nil
}

func makeServ(addr string, root string, nodes ...string) *server.FileServer {
	handshake := func(p p2p.Peer) error {
		z := p.(*p2p.TCPPeer)

		secureConn, err := p2p.HybridHandshake(z)
		if err != nil {
			return err
		}
		z.Conn = secureConn
		// continue using secureConn transparently
		return nil
	}

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    addr,
		HandshakeFunc: handshake,
		Decoder:       &p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)
	serverOpts := server.ServerOpts{
		PathTransformFunc: server.CASPathTransform,
		Transport:         tr,
		Nodes:             nodes,
	}

	return server.NewFileServer(serverOpts)

}

func main() {
	s := makeServ(":3000", "s1")

	s2 := makeServ(":4000", "s2", ":3000")

	cfg := &Config{
		FServer: s,
	}
	cfg2 := &Config{FServer: s2}

	go func() {
		err := cfg.FServer.Start()
		if err != nil {
			log.Fatalln("Failed to start server")
		}
	}()
	time.Sleep(time.Millisecond * 100)
	go func() {
		err := cfg2.FServer.Start()
		if err != nil {
			log.Fatalln("Failed to start server")
		}
	}()

	// Create a logger with the file handler

	// Use the logger

	select {}
}
