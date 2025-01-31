package storage

import (
	"bytes"
	"fmt"
	"io"

	"testing"
)

func newStorage() *Storage {
	opts := StorageOPTS{
		PathFunc: CASPathFunc,
	}

	return NewStorage(opts)
}

func clear(t *testing.T, s *Storage) {
	if err := s.ClearAll(); err != nil {
		t.Error(err)
	}
}

func TestHas(t *testing.T) {
	s := newStorage()
	defer clear(t, s)

	key := "test"

	data := []byte("some bytes")
	if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	exists := s.Has(key)
	if !exists {
		t.Errorf("want %v, have %v", true, exists)
	}

	exists = s.Has("nosuchfile")
	if exists {
		t.Errorf("want %v, have %v", false, exists)
	}
}

func TestDelete(t *testing.T) {
	s := newStorage()
	defer clear(t, s)

	key := "test"

	data := []byte("some bytes")
	if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	if err := s.Delete(key); err != nil {
		t.Error(err)
	}
}

func TestRead(t *testing.T) {
	s := newStorage()
	defer clear(t, s)

	key := "test"

	data := []byte("some bytes")
	if _, err := s.writeStream(key, bytes.NewReader(data)); err != nil {
		t.Error(err)
	}

	r, err := s.Read(key)
	if err != nil {
		t.Error(err)
	}

	rData, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}

	if string(rData) != string(data) {
		t.Errorf("want %s, have %s", data, rData)
	}
}

func TestWrite(t *testing.T) {
	s := newStorage()
	defer clear(t, s)

	for i := 1; i <= 100; i++ {
		key := fmt.Sprintf("test_%d", i)

		data := bytes.NewBuffer([]byte("some bytes"))
		if _, err := s.writeStream(key, data); err != nil {
			t.Error(err)
		}
	}

}
