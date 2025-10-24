package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/gpr3211/dist-store/p2p"
	"io"
	"log"
	"log/slog"
	"sync"
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
	store    *Store
	peers    map[string]p2p.Peer
	mu       sync.Mutex
	QuitChan chan struct{}
	ctx      context.Context
}

func (f *FileServer) SaveData(id, key string, r io.Reader) error {

	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	err := f.store.Write(id, key, tee)
	if err != nil {
		return err
	}

	_, err = io.Copy(buf, r)
	if err != nil {
		return err
	}
	p := &Payload{
		ID:   id,
		Key:  key,
		Data: buf.Bytes(),
	}
	return f.broadcast(p)

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
	log.Printf("Accepted conn from %s", p.RemoteAddr())
	return nil
}
func NewFileServer(opts ServerOpts) *FileServer {
	op := StoreOpts{Root: opts.StorageRoot, PathTransformFunc: opts.PathTransformFunc}
	return &FileServer{
		ServerOpts: opts,
		store:      NewStore(op),
		peers:      make(map[string]p2p.Peer), // string reperesents peer address.
		mu:         sync.Mutex{},
		ctx:        context.Background(),
		QuitChan:   make(chan struct{}),
	}
}

func (f *FileServer) Get(id, key string) (io.Reader, error) {
	if f.store.Has(id, key) {
		fmt.Printf("item found locally, fetching . . .n")
		r, err := f.store.Read(id, key)
		return r, err
	}
	return nil, errors.New("not found")
}

// Stop closes conn with all peers and closes transport.
func (f *FileServer) Stop() {
	f.mu.Lock()
	for _, v := range f.peers {
		v.Close()
	}
	f.Transport.Close()

	f.mu.Unlock()

}

type Payload struct {
	ID   string
	Key  string
	Data []byte
}

// broadcast send key file to all connected peers.
func (fs *FileServer) broadcast(msg *Payload) error {
	fmt.Println("broadcasting ... to ", len(fs.peers))
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	peers := []io.Writer{}
	for addr, peer := range fs.peers {
		peers = append(peers, peer)
		fmt.Println("added peer", " ", addr)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (f *FileServer) readLoop(ctx context.Context) {
	defer func() {
		log.Printf("Closing server on %s", f.Transport.Addr())
	}()
	for {
		select {
		case msg := <-f.Transport.Consume(): // read from transpot msg chan

			var p Payload
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&p); err != nil {

				//				panic(err)
			}

			fmt.Printf(" MSG: %+v\n", string(p.Data))

		case <-ctx.Done():
			fmt.Println("Shutting down ...")

			f.Stop()
			f.QuitChan <- struct{}{}
			return

		}
	}
}

func (f *FileServer) Start(ctx context.Context) {
	if err := f.Transport.ListenAndAccept(); err != nil {
		panic(err)
	}
	LoadConfig()

	f.bootsrapNodes()

	go f.readLoop(ctx)

}
