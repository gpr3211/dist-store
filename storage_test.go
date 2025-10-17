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
	expectedPath := "ad1e7/d6c18/15d8e/3f9bf/a634c/12d6a/3f31f/9894e"
	if pathname != expectedPath {
		t.Error("bad test but ok")
	}
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
