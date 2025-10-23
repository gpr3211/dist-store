package main

import (
	"bytes"
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

func makeServ(addr string, root string, nodes ...string) *server.FileServer {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    addr,
		HandshakeFunc: p2p.NOPHandshakefunc,
		Decoder:       &p2p.DefaultDecoder{},
	}

	tr := p2p.NewTCPTransport(tcpOpts)
	serverOpts := server.ServerOpts{
		PathTransformFunc: server.DefaultPathTransformFunc,
		Transport:         tr,
		Nodes:             nodes,
	}
	serverOpts.StorageRoot = root
	s := server.NewFileServer(serverOpts)
	tr.Config.OnPeer = s.OnPeer
	return s

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
	time.Sleep(time.Second)
	cfg.FServer.SaveData("user-test", "data", bytes.NewReader([]byte("test string")))

	// Create a logger with the file handler

	// Use the logger

	select {}
}
