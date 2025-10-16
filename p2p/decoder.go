package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *Message) error
}

type GOBDecoder struct {
}

func (d *GOBDecoder) Decode(r io.Reader, v *Message) error {

	return gob.NewDecoder(r).Decode(v)
}

type DefaultDecoder struct{}

func (d *DefaultDecoder) Decode(r io.Reader, msg *Message) error {

	buf := make([]byte, 2048)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]

	return nil
}
