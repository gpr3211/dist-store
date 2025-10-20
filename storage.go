package main

import (
	"bytes"
	"errors"

	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type PathKey struct {
	PathName string
	Filename string
}

func (p PathKey) Fullpath() string {
	return fmt.Sprintf("%s/%s", p.PathName, p.Filename)
}

func (p PathKey) FirstPath() string {
	paths := strings.Split(p.PathName, "/")[0]
	if len(paths) == 0 {
		return ""
	}
	return paths
}

// CASPathTransform
// TODO:
// - change to user_hash/key_hash
func CASPathTransform(key string) PathKey {

	hash := sha1.Sum([]byte(key))
	hashString := hex.EncodeToString(hash[:]) // [:] converts [20]byte array to a slice.

	blocksize := 5
	sliceLen := len(hashString) / blocksize
	paths := make([]string, sliceLen)
	for i := range paths {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashString[from:to]
	}
	return PathKey{
		PathName: strings.Join(paths, "/"),
		Filename: hashString, // root ?
	}

}

type PathTransformFunc func(string) PathKey

var DefaultPathTransformFunc = func(key string) PathKey {

	return PathKey{
		PathName: key,
		Filename: key,
	}
}

type Store struct {
	StoreOpts
}

func (s *Store) Delete(key string) error {
	pkey := s.PathTransformFunc(key)
	return os.RemoveAll(s.Root + "/" + pkey.FirstPath())

}
func (s *Store) Clear() error { return os.RemoveAll(s.Root) }

// Has checks if the store contains a key.
// - returns true if found.
func (s *Store) Has(key string) bool {
	pk := s.PathTransformFunc(key)

	rootPath := s.Root + "/" + pk.Fullpath()
	_, err := os.Stat(rootPath) // check if file exists.

	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Read(key string) (io.Reader, error) {
	f, err := s.readStream(key)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)

	return buf, err

}

const defaultRootFolder = "ggdata"

type StoreOpts struct {
	Root              string // Root is the root data dir.
	PathTransformFunc PathTransformFunc
}

func NewStore(opts StoreOpts) *Store {

	// set default.
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolder
	}
	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) Write(key string, r io.Reader) error {

	return s.writeStream(key, r)
}

func (s *Store) readStream(key string) (io.Reader, error) {

	pKey := s.PathTransformFunc(key)
	fullPath := pKey.Fullpath()
	f, err := os.Open(s.Root + "/" + fullPath)
	if err != nil {
		return nil, err
	}
	return f, err

}

func (s *Store) writeStream(key string, r io.Reader) error {

	pathkey := s.PathTransformFunc(key) //change dir structure here
	if err := os.MkdirAll(s.Root+"/"+pathkey.PathName, os.ModePerm); err != nil {
		return err

	}

	pathandFilename := pathkey.Fullpath()
	// f, err := os.OpenFile(pathandFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	f, err := os.OpenFile(s.Root+"/"+pathandFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return err
	}
	log.Printf("Written (%d) bytes to disk", n)

	return nil
}
