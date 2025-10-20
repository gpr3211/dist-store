package server

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/gpr3211/dist-store/p2p"
	"github.com/spf13/viper"
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

var CONFIG = ".config.json"

func LoadConfig() error {
	if len(os.Args) == 1 {
		fmt.Println("Using Default Config")
	} else {
		CONFIG = os.Args[1]
	}
	viper.SetConfigType("json")
	viper.SetConfigFile(CONFIG)
	fmt.Printf("Using config: %s\n", viper.ConfigFileUsed())
	viper.ReadInConfig()
	if viper.IsSet("addr") {
		fmt.Println("Address: ", viper.Get("addr"))
	} else {
		fmt.Println("Address not set")
	}
	var t ServerOpts
	err := viper.Unmarshal(&t)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
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
	log.Printf("%s Accepted conn from %s", p.LocalAddr(), p.RemoteAddr())

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
	LoadConfig()

	f.bootsrapNodes()
	f.readLoop()

	return nil
}
