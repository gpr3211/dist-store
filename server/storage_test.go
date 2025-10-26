package server

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/gpr3211/dist-store/assert"
)

func newStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: DefaultPathTransformFunc,
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

func TestStorage(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: DefaultPathTransformFunc}

	s := NewStore(opts)
	defer teardown(t, s)

	data := []byte("some bytess")
	key := "Moe's Specials"
	id := "user-test-storage"
	_, err := s.writeStream(id, key, bytes.NewReader(data))
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

// TestDirectoryTraversal tests if we can escape user directories.
func TestDirectoryTraversal(t *testing.T) {
	s := newStore()
	defer teardown(t, s)

	user1 := "user1"
	user2 := "user2"

	// user1 writes.
	user1Data := []byte("user1's secret data")
	_, err := s.Write(user1, "secrets", bytes.NewReader(user1Data))
	if err != nil {
		t.Fatalf("Failed to write user1 data: %v", err)
	}

	// basic bath treversal.
	attackKey := "../user1/secrets"

	if s.Has(user2, attackKey) {
		reader, err := s.Read(user2, attackKey)
		if err == nil {
			attackData, _ := io.ReadAll(reader)
			if bytes.Equal(attackData, user1Data) {
				t.Error("DIRECTORY TRAVERSAL: user2 accessed user1's data!")
			}
		}
	}
}

func TestGetUserKeys(t *testing.T) {
	opts := StoreOpts{PathTransformFunc: DefaultPathTransformFunc}

	store := NewStore(opts)
	defer teardown(t, store)

	data := []byte("some bytess")

	key := "Moe's Specials"
	key2 := "moe's 2"
	key3 := "moe's 3"

	expected := []string{key, key2, key3}
	expected2 := []string{key3}

	id := "user-test-storage"
	id2 := "user-test-storage2"
	store.Write(id, key, bytes.NewReader(data))

	store.Write(id, key2, bytes.NewReader(data))

	store.Write(id, key3, bytes.NewReader(data))

	store.Write(id2, key3, bytes.NewReader(data))

	items, err := store.GetKeys(id)
	if err != nil {
		t.Error("failed to get user keys", err)
	}
	for i, v := range items {
		assert.Equal(t, v, expected[i])
	}
	t2, err := store.GetKeys(id2)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, t2, expected2)

}

func TestPathTransform(t *testing.T) {
	key := "best-pics.jpg"
	id := "user-test"
	pathname := DefaultPathTransformFunc(id, key)
	fmt.Println(pathname)
	expectedPath := pathname.Fullpath()

	if pathname.Fullpath() != expectedPath {
		t.Errorf("pathname not expected %s", pathname.Fullpath())
	}
	if pathname.Filename != key {
		t.Errorf("Filename: got [%s] expected [%s]", pathname.Filename, expectedPath)
	}
}

func TestStoreDelete(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: DefaultPathTransformFunc,
	}
	s := NewStore(opts)
	defer teardown(t, s)

	key := "ss"
	id := "user-test-delete"
	data := []byte{1, 2, 3}
	if _, err := s.writeStream(id, key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}
	if err := s.Delete(id, key); err != nil {
		t.Error(err)
	}
}
