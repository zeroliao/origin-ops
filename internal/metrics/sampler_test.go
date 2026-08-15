package metrics

import (
	"errors"
	"testing"
	"time"
)

type collectorStub struct {
	snapshot Snapshot
	err      error
}

func (c collectorStub) Collect() (Snapshot, error) {
	return c.snapshot, c.err
}

type appenderStub struct {
	appended []Snapshot
	err      error
}

func (a *appenderStub) Append(snapshot Snapshot) error {
	a.appended = append(a.appended, snapshot)
	return a.err
}

func TestSamplerKeepsLastSuccessfulSnapshotAfterFailure(t *testing.T) {
	initial := Snapshot{Time: time.Now().UTC(), CPUPercent: 12}
	appender := &appenderStub{}
	sampler := NewSampler(collectorStub{snapshot: initial}, appender, time.Minute)
	sampler.sample()

	sampler.collector = collectorStub{err: errors.New("collector unavailable")}
	sampler.sample()
	status := sampler.Status()
	if !status.HasLatest || status.Latest.CPUPercent != 12 {
		t.Fatalf("latest snapshot was not retained: %+v", status)
	}
	if status.LastError != "collector unavailable" || status.LastFailure.IsZero() {
		t.Fatalf("failure was not recorded: %+v", status)
	}
}

func TestSamplerDoesNotPublishSnapshotWhenStoreFails(t *testing.T) {
	appender := &appenderStub{err: errors.New("disk full")}
	sampler := NewSampler(collectorStub{snapshot: Snapshot{Time: time.Now().UTC()}}, appender, time.Minute)
	sampler.sample()
	status := sampler.Status()
	if status.HasLatest {
		t.Fatal("sampler published a snapshot that was not persisted")
	}
	if status.LastError != "disk full" {
		t.Fatalf("LastError = %q", status.LastError)
	}
}
