package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
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
	serverOpts.AutoSync = true

	// SLogger.
	optsLog := slog.HandlerOptions{
		Level: slog.LevelDebug,
		//		AddSource: true, // noisy but useful
	}

	serverOpts.Logger = slog.New(slog.NewJSONHandler(os.Stdout, &optsLog))
	s := server.NewFileServer(serverOpts)
	tr.Config.OnPeer = s.OnPeer
	return s

}

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

	time.Sleep(time.Second * 3)
	go cfg2.FServer.Start(ctx)

	time.Sleep(time.Second * 3) // give time to start and establish conn.

	cfg.FServer.SaveData("user-test", "data", bytes.NewReader([]byte("test string"))) // save and broadcast data

	time.Sleep(time.Second * 1)
	bigData := make([]byte, 10*1024*1024)
	if _, err := io.ReadFull(rand.Reader, bigData); err != nil {
		panic(err)
	}

	cfg2.FServer.SaveData("user-test2", "data2", bytes.NewReader(bigData)) // save and broadcast data

	/*
		fmt.Printf("\n╔═══════════════════════════════════════╗\n")
		fmt.Printf("║  Distributed File Storage System      ║\n")
		fmt.Printf("║  Listening on: %-23s ║\n", cfg.FServer.Transport.Addr())
		fmt.Printf("╚═══════════════════════════════════════╝\n\n")

		printHelp()

		// Run the interactive loop in a goroutine
		go func() {
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
					stop() // Cancel the context to trigger shutdown
					return

				default:
					fmt.Printf("Unknown command: %s (type 'help' for commands)\n", cmd)
				}

				fmt.Print("> ")
			}
		}()
	*/

	// Wait for context cancellation
	<-ctx.Done()
	fmt.Println("\nShutdown signal received, cleaning up...")

	// Wait for both servers to finish with a timeout
	shutdownComplete := make(chan struct{})
	go func() {
		<-cfg.FServer.QuitChan
		<-cfg2.FServer.QuitChan
		close(shutdownComplete)
	}()

	select {
	case <-shutdownComplete:
		slog.Info("Shutdown complete.")
	case <-time.After(5 * time.Second):
		slog.Warn("Shutdown timeout - forcing exit")
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
