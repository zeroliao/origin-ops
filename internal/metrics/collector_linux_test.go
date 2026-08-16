//go:build linux

package metrics

import "testing"

func TestLinuxCollectorPublishesFirstSnapshot(t *testing.T) {
	collector := &linuxCollector{}
	snapshot, err := collector.Collect()
	if err != nil {
		t.Fatalf("first collection failed: %v", err)
	}
	if snapshot.Time.IsZero() {
		t.Fatal("first collection returned a zero timestamp")
	}
}
