package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/gpr3211/dist-store/p2p"
)

type Config struct {
	addr    string
	FServer *FileServer
	secret  []byte
}

func OnPeer(p2p.Peer) error {
	fmt.Println("Doing soem logic outside of transport tcp blah ")
	return nil
}

func main() {
	/*
		handshake := func(p p2p.Peer) error {
					z := p.(*p2p.TCPPeer)

					secureConn, err := p2p.SecureHandshake(z)
					if err != nil {
						return err
					}
					z.Conn = secureConn
					// continue using secureConn transparently
					return nil
				},

		}
	*/

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakefunc,
		Decoder:       &p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}
	tcpOpts2 := p2p.TCPTransportOpts{
		ListenAddr:    ":4000",
		HandshakeFunc: p2p.NOPHandshakefunc,
		Decoder:       &p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)
	serverOpts := ServerOpts{
		PathTransformFunc: CASPathTransform,
		Transport:         tr,
		StorageRoot:       "data",
		Nodes:             []string{":4000"},
	}
	tr2 := p2p.NewTCPTransport(tcpOpts2)
	serverOpts2 := ServerOpts{
		PathTransformFunc: CASPathTransform,
		Transport:         tr2,
		StorageRoot:       "data",
		Nodes:             []string{":3000"},
	}

	s := NewFileServer(serverOpts)
	s2 := NewFileServer(serverOpts2)

	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	// Create a JSON handler that writes to the file
	handler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true, // Adds source file and line number
	})

	logger := slog.New(handler)
	cfg := &Config{
		FServer: s,
	}
	cfg2 := &Config{FServer: s2}
	cfg.FServer.logger = logger

	cfg2.FServer.logger = logger

	go func() {
		err = cfg.FServer.Start()
		if err != nil {
			log.Fatalln("Failed to start server")
		}
	}()
	time.Sleep(time.Millisecond * 100)
	go func() {
		err = cfg2.FServer.Start()
		if err != nil {
			log.Fatalln("Failed to start server")
		}
	}()

	// Create a logger with the file handler

	// Use the logger

	select {}
}
