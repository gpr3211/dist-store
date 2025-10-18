package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

// TODO: fix
func TestPathTransform(t *testing.T) {
	key := "best-pics"
	pathname := CASPathTransform(key)
	fmt.Println(pathname)
	expectedPath := "ad1e7/d6c18/15d8e/3f9bf/a634c/12d6a/3f31f/9894e"
	expectedO := "ad1e7d6c1815d8e3f9bfa634c12d6a3f31f9894e"
	if pathname.PathName != expectedPath {
		t.Error("pathname not expected")
	}
	if pathname.Filename != expectedO {
		t.Errorf("Filename: got [%s] expected [%s]", pathname.Filename, expectedO)
	}
}

func TestStoreDelete(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransform,
	}
	s := NewStore(opts)
	key := "ss"
	data := []byte{1, 2, 3}
	if err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	if err := s.Delete(key); err != nil {
		t.Error(err)
	}

}

func TestStorage(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: DefaultPathTransformFunc}

	s := NewStore(opts)
	data := []byte("some bytess")
	key := "Moe's Specials"
	err := s.writeStream(key, bytes.NewReader(data))
	if err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)

	if string(b) != string(data) {
		t.Errorf("want [%s] have [%s]", data, b)
	}
	t.Attr("Storage", "test Has func")
	if ok := s.Has(key); !ok {
		t.Errorf("expected to have key")
	}

	err = s.Delete(key)

	if err != nil {
		t.Errorf("failed to delete")
	}
}
