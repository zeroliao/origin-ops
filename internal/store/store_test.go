package store

import (
	"os"
	"testing"
	"time"

	"origin-ops/internal/metrics"
)

func TestStoreAppendAndScanAcrossDays(t *testing.T) {
	directory := t.TempDir()
	metricStore, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}

	first := testSnapshot(time.Date(2026, 8, 15, 23, 59, 0, 0, time.UTC), 12.5)
	second := testSnapshot(time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC), 37.5)
	for _, snapshot := range []metrics.Snapshot{first, second} {
		if err := metricStore.Append(snapshot); err != nil {
			t.Fatal(err)
		}
	}

	var found []metrics.Snapshot
	err = metricStore.Scan(first.Time, second.Time.Add(time.Minute), func(snapshot metrics.Snapshot) error {
		found = append(found, snapshot)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("found %d snapshots", len(found))
	}
	if found[0].CPUPercent != 12.5 || found[1].CPUPercent != 37.5 {
		t.Fatalf("unexpected CPU values: %v, %v", found[0].CPUPercent, found[1].CPUPercent)
	}

	for _, day := range []time.Time{first.Time, second.Time} {
		info, statErr := os.Stat(metricStore.pathFor(day))
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Size() != recordSize {
			t.Fatalf("segment size = %d, want %d", info.Size(), recordSize)
		}
	}
}

func TestStoreIgnoresIncompleteTail(t *testing.T) {
	metricStore, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	timestamp := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	if err := metricStore.Append(testSnapshot(timestamp, 25)); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(metricStore.pathFor(timestamp), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte{1, 2, 3, 4}); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	count := 0
	err = metricStore.Scan(timestamp, timestamp.Add(time.Minute), func(metrics.Snapshot) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func testSnapshot(timestamp time.Time, cpu float64) metrics.Snapshot {
	return metrics.Snapshot{
		Time: timestamp, CPUPercent: cpu,
		MemoryUsedBytes: 50, MemoryTotalBytes: 100,
		DiskUsedBytes: 25, DiskTotalBytes: 100,
		RXBytesPerSecond: 10, TXBytesPerSecond: 5,
		Load1: 0.5, Load5: 0.4, Load15: 0.3,
	}
}
