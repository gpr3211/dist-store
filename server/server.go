package server

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/gpr3211/dist-store/model"
	"github.com/gpr3211/dist-store/p2p"
)

var ErrNotFound = errors.New("Not found")
var ErrClosedConn = errors.New("closed conn")

//TODO:
//
//
//
//

// ServerOpts
//
//	-- ID(int)
//	-- key([]byte) secret
type ServerOpts struct {
	configJson
	ID                int               `mapstructure:"id"`  // unused
	key               []byte            `mapstructure:"key"` // unused
	PathTransformFunc PathTransformFunc // naming convetion for storing files. curr {root}{user}{}key
	Transport         p2p.Transport     // transport type(ex p2p)
	Logger            *slog.Logger      //
	Nodes             []string
}
type configJson struct {
	ListenAddr  string `mapstructure:"addr"`
	StorageRoot string `mapstructure:"root"`
	AutoSync    bool   `mapstructure:"auto-sync"`
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
	tee := io.TeeReader(r, buf) //

	n, err := f.store.Write(id, key, tee)
	if err != nil {
		return err
	}
	//TODO:
	// add to flag or config.
	if f.AutoSync {

		// send metadata msg to peers.
		msg := model.Message{
			Payload: model.MessageStoreFile{
				ID:   id,
				Key:  key,
				Size: n,
			},
		}
		if err := f.broadcast(msg); err != nil {
			return nil
		}
		time.Sleep(time.Millisecond * 100)

		peers := []io.Writer{}
		for _, peer := range f.peers {
			peers = append(peers, peer)
		}
		mw := io.MultiWriter(peers...)
		mw.Write([]byte{p2p.IncomingStream}) // warn client that stream of data is inc so they can lock the peer.
		_, err := io.Copy(mw, buf)
		if err != nil {
			return err
		}

	}
	return nil

}

func (fs *FileServer) bootsrapNodes() error {
	for _, addr := range fs.Nodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				fs.Logger.Warn("Dial Error", err.Error(), err)
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
	fs.Logger.Info("Accepted conn from ", p.RemoteAddr().String(), nil)
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
func (f *FileServer) Stop() {
	f.Logger.Info("Server Stop called")

	// Close peers with timeout
	f.mu.Lock()
	peerList := make([]p2p.Peer, 0, len(f.peers))
	for _, v := range f.peers {
		peerList = append(peerList, v)
	}
	f.mu.Unlock()

	// Close peers without holding the lock
	for _, peer := range peerList {
		go func(p p2p.Peer) {
			// Use a timeout to prevent hanging
			done := make(chan struct{})
			go func() {
				p.Close()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):

				f.Logger.Warn("Timeout closing peer ...", p.RemoteAddr().String())

			}
		}(peer)
	}

	// Give peers a moment to close
	time.Sleep(500 * time.Millisecond)

	// Close transport
	if err := f.Transport.Close(); err != nil {
		f.Logger.Warn(err.Error())
		panic(err)
	}
}

/*
// Stop closes conn with all peers and closes transport.
func (f *FileServer) Stop() {
	f.mu.Lock()
	for _, v := range f.peers {
		v.Close()
	}
	f.Transport.Close()

	f.mu.Unlock()

}
*/

type Payload struct {
	ID   string
	Key  string
	Data []byte
}

// broadcast send key file to all connected peers.
func (fs *FileServer) broadcast(msg model.Message) error {
	fmt.Println("broadcasting ... to ", len(fs.peers))
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	for addr, peer := range fs.peers {
		fmt.Println("added peer", " ", addr)
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (f *FileServer) readLoop(ctx context.Context) {
	defer func() {
		close(f.QuitChan)
	}()
	for {
		select {
		case msg := <-f.Transport.Consume(): // read from transpot msg chan
			fmt.Println("got msg")

			var p model.Message
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&p); err != nil {
				log.Println("decoding error")
			}
			err := f.handleMessage(msg.From, &p)
			if err != nil {
				log.Println("handleMessage rror", err)
			}

		case <-ctx.Done():

			f.Stop()
			f.QuitChan <- struct{}{}
			return

		}
	}
}

// handleMessagehan
func (f *FileServer) handleMessage(from string, msg *model.Message) error {
	switch v := msg.Payload.(type) {
	case model.MessageStoreFile:
		return f.handleMessageStoreFile(from, v)
	}

	return nil
}

func (f *FileServer) handleMessageStoreFile(from string, data model.MessageStoreFile) error {
	peer, ok := f.peers[from]
	if !ok {
		return ErrNotFound
	}
	n, err := f.store.Write(data.ID, data.Key, io.LimitReader(peer, data.Size))
	if err != nil {
		return err
	}

	log.Printf("Written (%d) bytes to disk", n)
	peer.CloseStream()

	return nil

}

func (f *FileServer) Start(ctx context.Context) {
	if err := f.Transport.ListenAndAccept(); err != nil {
		panic(err)
	}
	//	LoadConfig()

	f.bootsrapNodes()

	go f.readLoop(ctx)

}
