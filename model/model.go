package model

import "encoding/gob"

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(Message{})
}

type Message struct {
	Payload any
}
type MessageStoreFile struct {
	ID   string
	Key  string
	Size int64
}
