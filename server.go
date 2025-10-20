package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"sync"

	"github.com/gpr3211/dist-store/p2p"
)

type ServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	key               []byte
	Transport         p2p.Transport
	logger            *slog.Logger
	Nodes             []string
}

type FileServer struct {
	ServerOpts
	store *Store
	peers map[net.Addr]p2p.Peer
	mu    sync.Mutex
	qChan chan struct{}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}

func (fs *FileServer) bootsrapNodes() error {
	for _, addr := range fs.Nodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}

// OnPeer logic after connection is comleted.
func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.peers[p.RemoteAddr()] = p
	fs.logger.Info("connect", fmt.Sprintf("%s Accepted conn from %s", p.LocalAddr(), p.RemoteAddr()))

	return nil
}
func NewFileServer(opts ServerOpts) *FileServer {
	op := StoreOpts{Root: opts.StorageRoot, PathTransformFunc: opts.PathTransformFunc}
	return &FileServer{
		ServerOpts: opts,
		store:      NewStore(op),
		qChan:      make(chan struct{}),
	}
}

func (f *FileServer) Stop() {

	close(f.qChan)

}
func (f *FileServer) readLoop() {
	defer func() {

		log.Printf("Closing server on %s", f.ListenAddr)
		f.Transport.Close()
		f.Stop()
	}()
	for {
		select {
		case msg := <-f.Transport.Consume(): // read from transpot msg chan
			fmt.Printf("Got %+v\n", msg.Payload)

		case <-f.qChan:
			return

		}
	}
}

func (f *FileServer) Start() error {
	if err := f.Transport.ListenAndAccept(); err != nil {
		return err
	}

	f.bootsrapNodes()
	f.readLoop()

	return nil
}
