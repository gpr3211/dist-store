package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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
		HandshakeFunc: p2p.HybridHandshake, // rsa for  key exchange and aes for data encryption.
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

//TODO:
//
//
// - add logger

func main() {
	s := makeServ(":3000", "s1")           // port,root dir
	s2 := makeServ(":4000", "s2", ":3000") // + bootstrap variadic string.

	cfg := &Config{
		FServer: s,
	}
	cfg2 := &Config{FServer: s2}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go cfg.FServer.Start(ctx)

	time.Sleep(time.Second * 2)
	go cfg2.FServer.Start(ctx)
	time.Sleep(time.Second * 2)

	cfg.FServer.SaveData("user-test", "data", bytes.NewReader([]byte("test string"))) // save and broadcast data

	<-ctx.Done() // block

	stop()

	forceQ := make(chan os.Signal, 1)
	signal.Notify(forceQ, os.Interrupt, syscall.SIGTERM)
	select {
	case <-cfg.FServer.QuitChan:
		slog.Info("Shutdown complete .")
	case <-forceQ:
		slog.Info("force quit")
		os.Exit(1)

	}

}
