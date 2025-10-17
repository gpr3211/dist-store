package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"log"
	"os"
	"strings"
)

func CASPathTransform(key string) string {

	hash := sha1.Sum([]byte(key))
	hashString := hex.EncodeToString(hash[:]) // [:] converts [20]byte array to a slice.

	blocksize := len(hashString) / 2
	sliceLen := len(hashString) / blocksize
	paths := make([]string, sliceLen)
	for i := range paths {
		from, to := i*blocksize, (i*blocksize)+blocksize
		paths[i] = hashString[from:to]
	}
	return strings.Join(paths, "/")

}

type PathTransformFunc func(string) string

var DefaultPathTransformFunc = func(key string) string { return key }

type Store struct {
	StoreOpts
}

type StoreOpts struct {
	PathTransformFunc PathTransformFunc
}

func NewStore(opts StoreOpts) *Store {

	return &Store{
		StoreOpts: opts,
	}
}

type PathKey struct {
	PathName string
	Original string
}

func (s *Store) readStream(key string) {

}

func (s *Store) writeStream(key string, r io.Reader) error {

	pathname := s.PathTransformFunc(key) //change dir structure here
	if err := os.MkdirAll(pathname, os.ModePerm); err != nil {
		return err

	}
	buf := new(bytes.Buffer)

	io.Copy(buf, r)

	filenameBytes := md5.Sum(buf.Bytes())
	filename := hex.EncodeToString(filenameBytes[:])

	pathandFilename := pathname + "/" + filename
	f, err := os.OpenFile(pathandFilename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
