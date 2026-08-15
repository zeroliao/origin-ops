package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"origin-ops/internal/metrics"
)

type samplerStub struct {
	status metrics.SamplerStatus
}

func (s samplerStub) Status() metrics.SamplerStatus {
	return s.status
}

type metricStoreStub struct {
	snapshots []metrics.Snapshot
}

func (s metricStoreStub) Scan(start, end time.Time, visit func(metrics.Snapshot) error) error {
	for _, snapshot := range s.snapshots {
		if !snapshot.Time.Before(start) && snapshot.Time.Before(end) {
			if err := visit(snapshot); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestAggregateReturnsAveragePeakAndMissingState(t *testing.T) {
	start := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	store := metricStoreStub{snapshots: []metrics.Snapshot{
		{Time: start, CPUPercent: 10, MemoryUsedBytes: 20, MemoryTotalBytes: 100},
		{Time: start.Add(time.Minute), CPUPercent: 30, MemoryUsedBytes: 40, MemoryTotalBytes: 100},
		{Time: start.Add(3 * time.Minute), CPUPercent: 50, MemoryUsedBytes: 60, MemoryTotalBytes: 100},
	}}

	buckets, duration, err := aggregate(store, start, start.Add(4*time.Minute), time.Minute, 2)
	if err != nil {
		t.Fatal(err)
	}
	if duration != 2*time.Minute || len(buckets) != 2 {
		t.Fatalf("duration = %v, buckets = %d", duration, len(buckets))
	}
	if buckets[0].Average.CPUPercent != 20 || buckets[0].Maximum.CPUPercent != 30 || buckets[0].Missing {
		t.Fatalf("first bucket = %+v", buckets[0])
	}
	if buckets[1].Average.CPUPercent != 50 || buckets[1].Maximum.CPUPercent != 50 || !buckets[1].Missing {
		t.Fatalf("second bucket = %+v", buckets[1])
	}
}

func TestMetricsEndpointValidatesMinutePrecision(t *testing.T) {
	handler := testHandler(t, metricStoreStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics?start=2026-08-15T00:00:01Z&end=2026-08-15T01:00:00Z", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestOverviewReturnsSamplerStatus(t *testing.T) {
	assets := fstest.MapFS{"index.html": {Data: []byte("ok")}}
	handler := NewHandler(Dependencies{
		Assets: assets,
		Sampler: samplerStub{status: metrics.SamplerStatus{
			HasLatest: true,
			Latest:    metrics.Snapshot{CPUPercent: 22},
		}},
		Store: metricStoreStub{}, SamplingInterval: time.Minute,
		MaxQueryDays: 31, MaxChartPoints: 240,
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var payload struct {
		Sampler metrics.SamplerStatus `json:"sampler"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Sampler.HasLatest || payload.Sampler.Latest.CPUPercent != 22 {
		t.Fatalf("payload = %+v", payload)
	}
}

func testHandler(t *testing.T, store MetricStore) http.Handler {
	t.Helper()
	var assets fs.FS = fstest.MapFS{"index.html": {Data: []byte("ok")}}
	return NewHandler(Dependencies{
		Assets: assets, Sampler: samplerStub{}, Store: store,
		SamplingInterval: time.Minute, MaxQueryDays: 31, MaxChartPoints: 240,
	})
}
