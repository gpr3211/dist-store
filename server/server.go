package server

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"

	"github.com/gpr3211/dist-store/p2p"
)

type ServerOpts struct {
	configJson
	ID                int    `mapstructure:"id"`
	key               []byte `mapstructure:"key"`
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	logger            *slog.Logger
	Nodes             []string
}
type configJson struct {
	ListenAddr  string `mapstructure:"addr"`
	StorageRoot string `mapstructure:"root"`
}

type FileServer struct {
	ServerOpts
	store *Store
	peers map[string]p2p.Peer
	mu    sync.Mutex
	qChan chan struct{}
}

func (f *FileServer) SaveData(key string, r io.Reader) {

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
	fs.peers[p.RemoteAddr().String()] = p
	log.Printf("%s Accepted conn from %s", p.LocalAddr(), p.RemoteAddr())

	return nil
}
func NewFileServer(opts ServerOpts) *FileServer {
	op := StoreOpts{Root: opts.StorageRoot, PathTransformFunc: opts.PathTransformFunc}
	return &FileServer{
		ServerOpts: opts,
		store:      NewStore(op),
		qChan:      make(chan struct{}),
		peers:      make(map[string]p2p.Peer),
		mu:         sync.Mutex{},
	}
}

func (f *FileServer) Stop() {
	f.mu.Lock()
	for _, v := range f.peers {
		v.Close()
	}

	f.mu.Unlock()
	close(f.qChan)

}

type Payload struct {
	Key  string
	Data []byte
}

// broadcast send key file to all connected peers.
func (fs *FileServer) broadcast() error {
	return nil
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {

	return nil
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
	LoadConfig()

	f.bootsrapNodes()
	f.readLoop()

	return nil
}
