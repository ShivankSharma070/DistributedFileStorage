package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// DefaultRootDirName is the default directory name used for storing files
// when no root directory is specified
const DefaultRootDirName = "files"

// CASPathTransformFunc uses a key to generate a path which can be used to
// store a file on disk. A SHA1 hash is generated for the key, which is then
// split into fixed-size blocks to create a CAS (Content Addressable Storage)
// style path
func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 5
	sliceLen := len(hashStr) / blockSize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

// PathTransformFunc is a function type that transforms a key into a PathKey
// containing the pathname and filename used for storing a file on disk
type PathTransformFunc func(string) PathKey

// PathKey holds the pathname and filename components used to construct
// the full file path for storage
type PathKey struct {
	// Pathname is the directory path where the file will be stored
	Pathname string
	// Filename is the name of the file within the pathname
	Filename string
}

// DefaultPathTransformFunc is the default implementation of PathTransformFunc.
// It returns the key as both the pathname and filename without any transformation
func DefaultPathTransformFunc(key string) PathKey {
	return PathKey{
		Pathname: key,
		Filename: key,
	}
}

// FullPath returns the full path by joining pathname and filename
func (pk *PathKey) FullPath() string {
	return fmt.Sprintf("%s/%s", pk.Pathname, pk.Filename)
}

// FirstPathName returns the first component of the pathname (the root directory)
// by splitting the pathname on "/" and returning the first element
func (pk *PathKey) FirstPathName() string {
	paths := strings.Split(pk.Pathname, "/")
	if len(paths) == 0 {
		return ""
	}

	return paths[0]
}

// StoreOpts holds configuration options for creating a new Store
type StoreOpts struct {
	// Root is the root directory name that will contain all
	// files and subdirectories of the storage system
	root string
	PathTransformFunc
}

// Store is responsible for managing file storage on disk. It provides
// methods to read, write, delete, and check for the existence of files
type Store struct {
	StoreOpts
}

// NewStore creates a *Store object using the opts provided.
// If PathTransformFunc is not provided in the opts, a DefaultPathTransformFunc
// will be used
func NewStore(opts StoreOpts) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(opts.root) == 0 {
		opts.root = DefaultRootDirName
	}
	return &Store{
		StoreOpts: opts,
	}
}

// Clear will remove all the storage file on a server by removing the root
// directory used for storing files
func (s *Store) Clear() error {
	defer func() {
		log.Println("Removed all files")
	}()
	return os.RemoveAll(s.root)
}

// Has checks whether a file exists in the store for the given id and key.
// Returns true if the file exists, false otherwise
func (s *Store) Has(id string, key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullNameWithRoot := fmt.Sprintf("%s/%s/%s", s.root, id, pathKey.FullPath())
	_, err := os.Stat(fullNameWithRoot)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}

	return true
}

// Delete removes the file associated with the given id and key from the store.
// It removes the entire directory containing the file
func (s *Store) Delete(id string, key string) error {
	pathKey := s.PathTransformFunc(key)
	firstPathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.root, id, pathKey.FirstPathName())
	defer func() {
		log.Printf("Deleted File: %s\n", pathKey.FullPath())
	}()

	return os.RemoveAll(firstPathNameWithRoot)
}

// Read retrieves file content from the store for the given id and key.
// It returns the file size, an io.Reader to read the contents, and any error
func (s *Store) Read(id string, key string) (int64, io.Reader, error) {
	return s.readStream(id, key)
	// n, f, err := s.readStream(key)
	// if err != nil {
	// 	return 0,nil, err
	// }
	//
	// defer f.Close()
	//
	// buf := new(bytes.Buffer)
	// _, err = io.Copy(buf, f)
	// return n, buf, err
}

// readStream opens a file for reading and returns its size along with
// an io.ReadCloser to read the file contents
func (s *Store) readStream(id string, key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathTransformFunc(key)
	FullNameWithRoot := fmt.Sprintf("%s/%s/%s", s.root, id, pathKey.FullPath())

	file, err := os.Open(FullNameWithRoot)
	if err != nil {
		return 0, nil, err
	}

	fileStat, err := file.Stat()
	if err != nil {
		return 0, nil, err
	}
	return fileStat.Size(), file, nil
}

// Write stores data from the provided io.Reader to disk. The path of the
// stored file is determined by the id and key. The key should be unique
// for each file. Returns the number of bytes written and any error
func (s *Store) Write(id string, key string, r io.Reader) (int64, error) {
	return s.writeStream(id, key, r)
}

// writeDecrypt opens a file for writing, decrypts the data from the provided
// io.Reader using the given encryption key, and writes the decrypted content
// to the file
func (s *Store) writeDecrypt(encKey []byte, id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := copyDecrypt(encKey, r, f)
	return int64(n), err
}

// writeStream opens a file for writing and copies data from the provided
// io.Reader into the file, returning the number of bytes written
func (s *Store) writeStream(id string, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(id, key)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return io.Copy(f, r)
}

// openFileForWriting creates the necessary directory structure for the given
// id and key, then opens or creates the file for writing and returns it
func (s *Store) openFileForWriting(id string, key string) (*os.File, error) {
	pathKey := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s/%s", s.root, id, pathKey.Pathname)

	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	FullNameWithRoot := fmt.Sprintf("%s/%s", pathNameWithRoot, pathKey.Filename)

	return os.Create(FullNameWithRoot)
}
