package main

import (
	"bytes"
	"fmt"
	"testing"
)

// TODO: fix
func TestPathTransform(t *testing.T) {
	key := "best-pics"
	pathname := CASPathTransform(key)
	fmt.Println(pathname)
}

func TestStorage(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: DefaultPathTransformFunc}

	s := NewStore(opts)
	data := bytes.NewReader([]byte("some bytess"))
	err := s.writeStream("myspecialPic", data)
	if err != nil {
		t.Error(err)
	}

}
