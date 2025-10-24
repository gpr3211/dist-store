package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gpr3211/dist-store/p2p"
	"github.com/gpr3211/dist-store/server"
)

type Config struct {
	addr    string
	FServer *server.FileServer
	// secret  []byte
}

func makeServ(addr string, root string, nodes ...string) *server.FileServer {
	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    addr,
		HandshakeFunc: p2p.HybridHandshake, // initial handshake acts as a wrapper around net.Conn for data encryption in transition. Uses RSA for  key exchange and AES for data encryption.
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
// - add slogger

func main() {
	s := makeServ(":3000", "s1")  // port,root dir
	s2 := makeServ(":4000", "s2") // + bootstrap variadic string.

	cfg := &Config{
		FServer: s,
	}
	cfg2 := &Config{FServer: s2}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go cfg.FServer.Start(ctx)

	time.Sleep(time.Second * 3)
	go cfg2.FServer.Start(ctx)

	time.Sleep(time.Second * 3) // give time to start and establish conn.

	cfg.FServer.SaveData("user-test", "data", bytes.NewReader([]byte("test string"))) // save and broadcast data

	time.Sleep(time.Second * 2)
	cfg2.FServer.SaveData("user-test2", "data2", bytes.NewReader([]byte("test string2"))) // save and broadcast data

	fmt.Printf("\n╔════════════════════════════════════════╗\n")
	fmt.Printf("║  Distributed File Storage System      ║\n")
	fmt.Printf("║  Listening on: %-23s ║\n")
	fmt.Printf("╚════════════════════════════════════════╝\n\n")

	printHelp()
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("> ") // print initially

	for {
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			fmt.Print("> ")
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]

		switch cmd {
		case "store":
			// your code...
		case "storefile":
			// your code...
		case "get":
		case "delete":
			if len(parts) < 2 {
				fmt.Println("Usage: delete <id> <key>")
			}
		case "has":
			if len(parts) < 2 {
				fmt.Println("Usage: has <key>")
			}
		case "peers":
		case "connect":
			if len(parts) < 2 {
				fmt.Println("Usage: connect <address>")
				fmt.Print("> ")
				continue
			}
			addr := parts[1]
			fmt.Printf("Attempting to connect to %s...\n", addr)
			if err := cfg.FServer.Transport.Dial(addr); err != nil {
				fmt.Printf("Error connecting: %v\n", err)
			} else {
				time.Sleep(500 * time.Millisecond)
				fmt.Printf("✓ Connected to %s\n", addr)
			}

		case "help":
			printHelp()

		case "exit", "quit":
			return // or break out cleanly

		default:
			fmt.Printf("Unknown command: %s (type 'help' for commands)\n", cmd)
		}

		fmt.Print("> ") // ✅ print prompt after each command
	}

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

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  store <key> <content>       - Store text content with a key")
	fmt.Println("  storefile <key> <filepath>  - Store a file with a key")
	fmt.Println("  get <key>                   - Retrieve and display content")
	fmt.Println("  delete <key>                - Delete local copy of file")
	fmt.Println("  has <key>                   - Check if file exists locally")
	fmt.Println("  connect <address>           - Manually connect to a peer")
	fmt.Println("  peers                       - List connected peers")
	fmt.Println("  help                        - Show this help")
	fmt.Println("  exit/quit                   - Exit the program")

}
