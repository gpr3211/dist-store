package server

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransform,
	}
	s := NewStore(opts)
	return s

}

func teardown(t *testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
		return
	}

}

// TODO: fix
func TestPathTransform(t *testing.T) {
	key := "best-pics"
	id := "user-test"
	pathname := CASPathTransform(id, key)
	fmt.Println(pathname)
	expectedPath := "f549249a2c3e3c3d8dd9da28e707523d359cfb46"
	t.Log(pathname.Fullpath())
	if pathname.PathName != expectedPath {
		t.Errorf("pathname not expected %s", pathname.PathName)
	}
	if pathname.Filename != key {
		t.Errorf("Filename: got [%s] expected [%s]", pathname.Filename, expectedPath)
	}
}

func TestStoreDelete(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransform,
	}
	s := NewStore(opts)
	key := "ss"
	id := "user-test"
	data := []byte{1, 2, 3}
	if err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	if err := s.Delete(id, key); err != nil {
		t.Error(err)
	}

}

func TestStorage(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: DefaultPathTransformFunc}

	s := NewStore(opts)

	defer teardown(t, s)
	data := []byte("some bytess")
	key := "Moe's Specials"
	id := "user-test"
	err := s.writeStream(id, key, bytes.NewReader(data))
	if err != nil {
		t.Error(err)
	}

	r, err := s.Read(id, key)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(r)

	if string(b) != string(data) {
		t.Errorf("want [%s] have [%s]", data, b)
	}
	t.Attr("Storage", "test Has func")
	if ok := s.Has(id, key); !ok {
		t.Errorf("expected to have key")
	}

	if err != nil {
		t.Errorf("failed to delete")
	}
}
