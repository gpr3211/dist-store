package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	opts := TCPTransportOpts{
		ListenAddr:    ":3000",
		HandshakeFunc: NOPHandshakefunc,
		Decoder:       &DefaultDecoder{},
	}
	tr := NewTCPTransport(opts)
	assert.Equal(t, tr.Config.ListenAddr, ":3000")

	assert.Nil(t, tr.ListenAndAccept())
}
