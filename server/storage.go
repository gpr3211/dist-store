package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
)

type PathKey struct {
	ID       string
	Filename string
	Hash     string
}

func (p PathKey) Fullpath() string {
	return fmt.Sprintf("%s/%s", p.ID, p.Filename)
}

func (p PathKey) FirstPath() string {
	return p.ID
}
func (p PathKey) SetHash() {

}

type Store struct {
	StoreOpts
	dir *os.Root
}
type PathTransformFunc func(string, string) PathKey

// DefaultPathTransformFunc creates path: /id/data
var DefaultPathTransformFunc = func(id, key string) PathKey {

	return PathKey{
		ID:       id,
		Filename: key}
}

func (s *Store) Delete(id, key string) error {
	pkey := s.PathTransformFunc(id, key)
	return s.dir.Remove(pkey.Fullpath())

}

func (s *Store) Clear() error {

	return os.RemoveAll(s.Root)
}

func (s *Store) GetKeys(id string) ([]string, error) {

	list := []string{}
	ls := s.dir.FS() //  convert to FS in order to use ReadDir, os.Root doesnt implement it.
	l, err := fs.ReadDir(ls, id)
	if err != nil {
		return nil, err
	}
	for _, v := range l {
		list = append(list, v.Name())
	}
	return list, nil
}

// Has checks if the store contains a key.
// - returns true if found.
func (s *Store) Has(id, key string) bool {
	pk := s.PathTransformFunc(id, key)
	rootPath := s.Root + "/" + pk.Fullpath()
	_, err := os.Stat(rootPath)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Read(id, key string) (io.Reader, error) {
	f, err := s.readStream(id, key)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	return buf, err
}

// Write saves a file to disk.
//
// -- will currently overwrite keys.
func (s *Store) Write(id, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
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
	s := &Store{
		StoreOpts: opts,
	}
	f, err := HandleRootDir(s.Root)
	if err != nil {
		panic(err)
	}

	s.dir = f
	return s
}

func HandleRootDir(s string) (*os.Root, error) {
	info, err := os.Stat(s)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(s, os.ModePerm); err != nil {
			return nil, err
		}
	} else if info.IsDir() {
		return os.OpenRoot(s)
	}
	return os.OpenRoot(s)
}
func (s *Store) OpenFileForWrite(id, key string) (*os.File, error) {
	pathkey := s.PathTransformFunc(id, key)
	// Create full directory structure: root/user_id/item_name
	fullDir := s.Root + "/" + pathkey.ID
	_, err := os.Stat(fullDir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(fullDir, os.ModePerm); err != nil {
			return nil, err
		}
	}
	pathandFilename := pathkey.Fullpath()
	f, err := s.dir.OpenFile(pathandFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Store) writeStream(id, key string, r io.Reader) (int64, error) {
	f, err := s.OpenFileForWrite(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

func (s *Store) readStream(id, key string) (io.Reader, error) {
	pKey := s.PathTransformFunc(id, key)
	fullPath := pKey.Fullpath()
	return s.dir.Open(fullPath)
}
