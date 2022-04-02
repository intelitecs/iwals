package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var (
	Enc = binary.BigEndian
)

const (
	LenWidth = 8
)

type Store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64
}

func NewStore(f *os.File) (*Store, error) {
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	size := uint64(fi.Size())
	return &Store{
		File: f,
		size: size,
		buf:  bufio.NewWriter(f),
	}, nil

}

func (s *Store) Append(b []byte) (n uint64, pos uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pos = s.size
	if err := binary.Write(s.buf, Enc, uint64(len(b))); err != nil {
		return 0, 0, err
	}
	w, err := s.buf.Write(b)
	if err != nil {
		return 0, 0, err
	}
	w += LenWidth
	s.size += uint64(w)
	return uint64(w), pos, nil
}

func (s *Store) Read(pos uint64) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.buf.Flush(); err != nil {
		return nil, err
	}
	size := make([]byte, LenWidth)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}
	b := make([]byte, Enc.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(pos+LenWidth)); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *Store) ReadAt(b []byte, off int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return 0, err
	}
	return s.File.ReadAt(b, off)
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.buf.Flush(); err != nil {
		return err
	}
	return s.File.Close()
}
