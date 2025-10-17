package main

import (
	"fmt"
	"github.com/gpr3211/dist-store/p2p"
	"log"
	"log/slog"
	"os"
)

type Config struct {
	Transport p2p.Transport
	logger    *slog.Logger
}

func OnPeer(p2p.Peer) error {
	fmt.Println("Doing soem logic outside of transport tcp blah ")
	return nil
}

func main() {

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: p2p.NOPHandshakefunc,
		Decoder:       &p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)
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
		Transport: tr,
		logger:    logger,
	}

	// Create a logger with the file handler

	// Use the logger
	logger.Info("Application started")

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("%+v\n", msg)

		}
	}()

	go func() {
		if err := cfg.Transport.ListenAndAccept(); err != nil {
			cfg.logger.Error(err.Error())
			log.Fatal(err)
		}
	}()

	fmt.Println("Listening on ", tcpOpts.ListenAddr)

	select {}
}
