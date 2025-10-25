package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct {
}

func (d *GOBDecoder) Decode(r io.Reader, v *RPC) error {

	return gob.NewDecoder(r).Decode(v)
}

type DefaultDecoder struct{}

func (d *DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	// read first byte to get type of message
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return err
	}

	typePrefix := peekBuf[0]
	switch typePrefix {

	//if incoming  stream of data we peer.wg.Add(1) and wait for the data stream to finish before releasing lock inside tcp_transport read loop for that connection.
	case IncomingStream:
		msg.Stream = true
		return nil

	case IncomingMessage: // basic msg.
		buf := make([]byte, 1028)
		n, err := r.Read(buf)
		if err != nil {
			return err
		}

		msg.Payload = buf[:n]

		return nil

	}
	return nil
	// In case of a stream we are not decoding what is being sent over the network.
	// We are just setting Stream true so we can handle that in our logic.

}
