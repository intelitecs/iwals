package server

import (
	"fmt"
	"sync"
)

type Log struct {
	mu      sync.Mutex
	records []Record
}

type Record struct {
	Value  []byte `json:"value"`
	Offset int64  `json:"offset"`
}

func NewLog() *Log {
	return &Log{}
}

func (l *Log) Append(record Record) (int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	record.Offset = int64(len(l.records))
	l.records = append(l.records, record)
	return record.Offset, nil
}

func (l *Log) Read(offset int64) (Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if offset >= int64(len(l.records)) {
		return Record{}, ErrOffsetNotFound(offset)
	}
	return l.records[offset], nil
}

func ErrOffsetNotFound(offset int64) error {
	return fmt.Errorf("Record not found for offset %d", offset)
}
