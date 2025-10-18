package main

import (
	"bytes"

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

var DefaultPathTransformFunc = func(key string) PathKey { return CASPathTransform(key) }

type Store struct {
	StoreOpts
}

func (s *Store) Delete(key string) error {
	pkey := s.PathTransformFunc(key)
	if err := os.RemoveAll(pkey.Fullpath()); err != nil {
		fmt.Printf("Failed to remove key %s from %s", key, pkey.Fullpath())
		return err
	}
	return os.RemoveAll(pkey.PathName)

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

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

func NewStore(opts StoreOpts) *Store {

	return &Store{
		StoreOpts: opts,
	}
}

func (s *Store) readStream(key string) (io.Reader, error) {

	pKey := s.PathTransformFunc(key)
	fullPath := pKey.Fullpath()
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	return f, err

}

func (s *Store) writeStream(key string, r io.Reader) error {

	pathkey := s.PathTransformFunc(key) //change dir structure here
	if err := os.MkdirAll(pathkey.PathName, os.ModePerm); err != nil {
		return err

	}

	pathandFilename := pathkey.Fullpath()
	// f, err := os.OpenFile(pathandFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	f, err := os.OpenFile(pathandFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
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
