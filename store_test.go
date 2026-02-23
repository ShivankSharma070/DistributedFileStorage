package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createNewStore() *Store {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t *testing.T, s *Store) {
	err := s.Clear()
	assert.Nil(t, err)
}

func TestPathTransformFunc(t *testing.T) {
	key := "myspecialpicture"
	path := CASPathTransformFunc(key)
	expectedPathname := "d9e06/924cb/e4f7c/5f592/69e62/67f97/1d027/74564"
	expectedOriginalKey := "d9e06924cbe4f7c5f59269e6267f971d02774564"
	assert.Equal(t, path.Pathname, expectedPathname)
	assert.Equal(t, path.Filename, expectedOriginalKey)
}

func TestDelete(t *testing.T) {
	s := createNewStore()
	key := "myotherkey"
	data := []byte("some jpg")
	err := s.Write(key, bytes.NewReader(data))
	assert.Nil(t, err)

	err = s.Delete(key)
	assert.Nil(t, err)
}

func TestNewStore(t *testing.T) {
	s := createNewStore()
	defer teardown(t, s)
	key := "mykey"
	data := []byte("some jpg")
	if err := s.Write(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	assert.True(t, s.Has(key))

	r, err := s.Read(key)
	assert.Nil(t, err)

	buf, err := io.ReadAll(r)
	assert.Nil(t, err)
	assert.Equal(t, string(buf), string(data))
}

func TestHas(t *testing.T) {
	opts := StoreOpts{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "mykey"
	doesExists := s.Has(key)
	fmt.Println(doesExists)
	assert.False(t, doesExists)
}
