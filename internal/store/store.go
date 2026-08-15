package store

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"origin-ops/internal/metrics"
)

const recordSize = 64

type Store struct {
	directory string
	mu        sync.RWMutex
}

func New(directory string) (*Store, error) {
	if directory == "" {
		return nil, fmt.Errorf("metric directory is required")
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, fmt.Errorf("create metric directory: %w", err)
	}
	return &Store{directory: directory}, nil
}

func (s *Store) Append(snapshot metrics.Snapshot) error {
	if snapshot.Time.IsZero() {
		return fmt.Errorf("snapshot time is required")
	}
	data := encode(snapshot)
	path := s.pathFor(snapshot.Time)

	s.mu.Lock()
	defer s.mu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return fmt.Errorf("open metric segment: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(data[:]); err != nil {
		return fmt.Errorf("append metric record: %w", err)
	}
	return nil
}

func (s *Store) Scan(start, end time.Time, visit func(metrics.Snapshot) error) error {
	if !start.Before(end) {
		return fmt.Errorf("start must be before end")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for day := utcDay(start); day.Before(end); day = day.AddDate(0, 0, 1) {
		if err := s.scanFile(s.pathFor(day), start, end, visit); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) scanFile(path string, start, end time.Time, visit func(metrics.Snapshot) error) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open metric segment: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, recordSize)
	for {
		_, err := io.ReadFull(file, buffer)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read metric segment: %w", err)
		}
		snapshot := decode(buffer)
		if !snapshot.Time.Before(start) && snapshot.Time.Before(end) {
			if err := visit(snapshot); err != nil {
				return err
			}
		}
	}
}

func (s *Store) pathFor(timestamp time.Time) string {
	return filepath.Join(s.directory, timestamp.UTC().Format("2006-01-02")+".bin")
}

func utcDay(timestamp time.Time) time.Time {
	utc := timestamp.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func encode(snapshot metrics.Snapshot) [recordSize]byte {
	var data [recordSize]byte
	binary.LittleEndian.PutUint64(data[0:8], uint64(snapshot.Time.UTC().Unix()))
	binary.LittleEndian.PutUint32(data[8:12], math.Float32bits(float32(snapshot.CPUPercent)))
	binary.LittleEndian.PutUint64(data[12:20], snapshot.MemoryUsedBytes)
	binary.LittleEndian.PutUint64(data[20:28], snapshot.MemoryTotalBytes)
	binary.LittleEndian.PutUint64(data[28:36], snapshot.DiskUsedBytes)
	binary.LittleEndian.PutUint64(data[36:44], snapshot.DiskTotalBytes)
	binary.LittleEndian.PutUint32(data[44:48], math.Float32bits(float32(snapshot.RXBytesPerSecond)))
	binary.LittleEndian.PutUint32(data[48:52], math.Float32bits(float32(snapshot.TXBytesPerSecond)))
	binary.LittleEndian.PutUint32(data[52:56], math.Float32bits(float32(snapshot.Load1)))
	binary.LittleEndian.PutUint32(data[56:60], math.Float32bits(float32(snapshot.Load5)))
	binary.LittleEndian.PutUint32(data[60:64], math.Float32bits(float32(snapshot.Load15)))
	return data
}

func decode(data []byte) metrics.Snapshot {
	return metrics.Snapshot{
		Time:             time.Unix(int64(binary.LittleEndian.Uint64(data[0:8])), 0).UTC(),
		CPUPercent:       float64(math.Float32frombits(binary.LittleEndian.Uint32(data[8:12]))),
		MemoryUsedBytes:  binary.LittleEndian.Uint64(data[12:20]),
		MemoryTotalBytes: binary.LittleEndian.Uint64(data[20:28]),
		DiskUsedBytes:    binary.LittleEndian.Uint64(data[28:36]),
		DiskTotalBytes:   binary.LittleEndian.Uint64(data[36:44]),
		RXBytesPerSecond: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[44:48]))),
		TXBytesPerSecond: float64(math.Float32frombits(binary.LittleEndian.Uint32(data[48:52]))),
		Load1:            float64(math.Float32frombits(binary.LittleEndian.Uint32(data[52:56]))),
		Load5:            float64(math.Float32frombits(binary.LittleEndian.Uint32(data[56:60]))),
		Load15:           float64(math.Float32frombits(binary.LittleEndian.Uint32(data[60:64]))),
	}
}
