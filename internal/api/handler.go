package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"origin-ops/internal/inventory"
	"origin-ops/internal/metrics"
)

type Sampler interface {
	Status() metrics.SamplerStatus
}

type MetricStore interface {
	Scan(start, end time.Time, visit func(metrics.Snapshot) error) error
}

type ApplicationInventory interface {
	List(context.Context) []inventory.Application
	Releases(context.Context, string) ([]inventory.Release, error)
}

type Dependencies struct {
	Assets           fs.FS
	Sampler          Sampler
	Store            MetricStore
	SamplingInterval time.Duration
	MaxQueryDays     int
	MaxChartPoints   int
	Inventory        ApplicationInventory
}

func NewHandler(dependencies Dependencies) http.Handler {
	mux := http.NewServeMux()
	querySlots := make(chan struct{}, 4)
	mux.HandleFunc("GET /api/v1/health", func(response http.ResponseWriter, request *http.Request) {
		writeJSON(response, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().UTC(),
		})
	})
	mux.HandleFunc("GET /api/v1/overview", func(response http.ResponseWriter, request *http.Request) {
		hostname, _ := os.Hostname()
		writeJSON(response, http.StatusOK, map[string]any{
			"host": map[string]any{
				"hostname": hostname,
				"os":       runtime.GOOS,
				"arch":     runtime.GOARCH,
				"cpuCount": runtime.NumCPU(),
			},
			"sampler": dependencies.Sampler.Status(),
		})
	})
	mux.HandleFunc("GET /api/v1/metrics", func(response http.ResponseWriter, request *http.Request) {
		select {
		case querySlots <- struct{}{}:
			defer func() { <-querySlots }()
		default:
			writeError(response, http.StatusServiceUnavailable, "too many concurrent metric queries")
			return
		}
		handleMetrics(response, request, dependencies)
	})
	mux.HandleFunc("GET /api/v1/applications", func(response http.ResponseWriter, request *http.Request) {
		if dependencies.Inventory == nil {
			writeError(response, http.StatusServiceUnavailable, "application inventory is unavailable")
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"applications": dependencies.Inventory.List(request.Context()),
		})
	})
	mux.HandleFunc("GET /api/v1/applications/{id}/releases", func(response http.ResponseWriter, request *http.Request) {
		if dependencies.Inventory == nil {
			writeError(response, http.StatusServiceUnavailable, "application inventory is unavailable")
			return
		}
		id := request.PathValue("id")
		releases, err := dependencies.Inventory.Releases(request.Context(), id)
		if errors.Is(err, inventory.ErrApplicationNotFound) {
			writeError(response, http.StatusNotFound, "application not found")
			return
		}
		if err != nil {
			writeError(response, http.StatusInternalServerError, "read release records")
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{
			"applicationID": id,
			"releases":      releases,
		})
	})
	mux.Handle("GET /", http.FileServer(http.FS(dependencies.Assets)))
	return securityHeaders(limitRequestTarget(mux))
}

func handleMetrics(response http.ResponseWriter, request *http.Request, dependencies Dependencies) {
	start, end, err := parseRange(request)
	if err != nil {
		writeError(response, http.StatusBadRequest, err.Error())
		return
	}
	if end.Sub(start) > time.Duration(dependencies.MaxQueryDays)*24*time.Hour {
		writeError(response, http.StatusBadRequest, fmt.Sprintf("range exceeds %d days", dependencies.MaxQueryDays))
		return
	}

	buckets, bucketDuration, err := aggregate(
		dependencies.Store,
		start,
		end,
		dependencies.SamplingInterval,
		dependencies.MaxChartPoints,
	)
	if err != nil {
		writeError(response, http.StatusInternalServerError, "query metric history")
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"start":            start,
		"end":              end,
		"bucketSeconds":    int(bucketDuration.Seconds()),
		"samplingInterval": int(dependencies.SamplingInterval.Seconds()),
		"buckets":          buckets,
	})
}

func parseRange(request *http.Request) (time.Time, time.Time, error) {
	startValue := request.URL.Query().Get("start")
	endValue := request.URL.Query().Get("end")
	if startValue == "" && endValue == "" {
		end := time.Now().UTC().Truncate(time.Minute).Add(time.Minute)
		return end.Add(-24 * time.Hour), end, nil
	}
	if startValue == "" || endValue == "" {
		return time.Time{}, time.Time{}, fmt.Errorf("start and end must be provided together")
	}
	start, err := time.Parse(time.RFC3339, startValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("start must be RFC3339")
	}
	end, err := time.Parse(time.RFC3339, endValue)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("end must be RFC3339")
	}
	if start.Second() != 0 || start.Nanosecond() != 0 || end.Second() != 0 || end.Nanosecond() != 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("start and end must use minute precision")
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("start must be before end")
	}
	return start.UTC(), end.UTC(), nil
}

type metricValues struct {
	CPUPercent       float64 `json:"cpuPercent"`
	MemoryPercent    float64 `json:"memoryPercent"`
	DiskPercent      float64 `json:"diskPercent"`
	RXBytesPerSecond float64 `json:"rxBytesPerSecond"`
	TXBytesPerSecond float64 `json:"txBytesPerSecond"`
	Load1            float64 `json:"load1"`
	Load5            float64 `json:"load5"`
	Load15           float64 `json:"load15"`
}

type metricBucket struct {
	Start   time.Time    `json:"start"`
	End     time.Time    `json:"end"`
	Average metricValues `json:"avg"`
	Maximum metricValues `json:"max"`
	Samples int          `json:"samples"`
	Missing bool         `json:"missing"`
}

type bucketAccumulator struct {
	metricBucket
	sum metricValues
}

func aggregate(store MetricStore, start, end time.Time, samplingInterval time.Duration, maxPoints int) ([]metricBucket, time.Duration, error) {
	if samplingInterval <= 0 {
		samplingInterval = time.Minute
	}
	rangeMinutes := int(math.Ceil(end.Sub(start).Minutes()))
	bucketMinutes := int(math.Ceil(float64(rangeMinutes) / float64(maxPoints)))
	if bucketMinutes < 1 {
		bucketMinutes = 1
	}
	bucketDuration := time.Duration(bucketMinutes) * time.Minute
	bucketCount := int(math.Ceil(float64(end.Sub(start)) / float64(bucketDuration)))
	accumulators := make([]bucketAccumulator, bucketCount)
	for index := range accumulators {
		bucketStart := start.Add(time.Duration(index) * bucketDuration)
		bucketEnd := bucketStart.Add(bucketDuration)
		if bucketEnd.After(end) {
			bucketEnd = end
		}
		accumulators[index].Start = bucketStart
		accumulators[index].End = bucketEnd
	}

	err := store.Scan(start, end, func(snapshot metrics.Snapshot) error {
		index := int(snapshot.Time.Sub(start) / bucketDuration)
		if index < 0 || index >= len(accumulators) {
			return nil
		}
		values := valuesFromSnapshot(snapshot)
		accumulator := &accumulators[index]
		accumulator.Samples++
		addValues(&accumulator.sum, values)
		if accumulator.Samples == 1 {
			accumulator.Maximum = values
		} else {
			maxValues(&accumulator.Maximum, values)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	buckets := make([]metricBucket, len(accumulators))
	for index := range accumulators {
		accumulator := &accumulators[index]
		if accumulator.Samples > 0 {
			accumulator.Average = divideValues(accumulator.sum, float64(accumulator.Samples))
		}
		expected := int(math.Ceil(accumulator.End.Sub(accumulator.Start).Seconds() / samplingInterval.Seconds()))
		accumulator.Missing = accumulator.Samples < expected
		buckets[index] = accumulator.metricBucket
	}
	return buckets, bucketDuration, nil
}

func valuesFromSnapshot(snapshot metrics.Snapshot) metricValues {
	return metricValues{
		CPUPercent:       snapshot.CPUPercent,
		MemoryPercent:    percentage(snapshot.MemoryUsedBytes, snapshot.MemoryTotalBytes),
		DiskPercent:      percentage(snapshot.DiskUsedBytes, snapshot.DiskTotalBytes),
		RXBytesPerSecond: snapshot.RXBytesPerSecond,
		TXBytesPerSecond: snapshot.TXBytesPerSecond,
		Load1:            snapshot.Load1,
		Load5:            snapshot.Load5,
		Load15:           snapshot.Load15,
	}
}

func percentage(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(used) / float64(total)
}

func addValues(target *metricValues, value metricValues) {
	target.CPUPercent += value.CPUPercent
	target.MemoryPercent += value.MemoryPercent
	target.DiskPercent += value.DiskPercent
	target.RXBytesPerSecond += value.RXBytesPerSecond
	target.TXBytesPerSecond += value.TXBytesPerSecond
	target.Load1 += value.Load1
	target.Load5 += value.Load5
	target.Load15 += value.Load15
}

func divideValues(value metricValues, divisor float64) metricValues {
	value.CPUPercent /= divisor
	value.MemoryPercent /= divisor
	value.DiskPercent /= divisor
	value.RXBytesPerSecond /= divisor
	value.TXBytesPerSecond /= divisor
	value.Load1 /= divisor
	value.Load5 /= divisor
	value.Load15 /= divisor
	return value
}

func maxValues(target *metricValues, value metricValues) {
	target.CPUPercent = math.Max(target.CPUPercent, value.CPUPercent)
	target.MemoryPercent = math.Max(target.MemoryPercent, value.MemoryPercent)
	target.DiskPercent = math.Max(target.DiskPercent, value.DiskPercent)
	target.RXBytesPerSecond = math.Max(target.RXBytesPerSecond, value.RXBytesPerSecond)
	target.TXBytesPerSecond = math.Max(target.TXBytesPerSecond, value.TXBytesPerSecond)
	target.Load1 = math.Max(target.Load1, value.Load1)
	target.Load5 = math.Max(target.Load5, value.Load5)
	target.Load15 = math.Max(target.Load15, value.Load15)
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeError(response http.ResponseWriter, status int, message string) {
	writeJSON(response, status, map[string]string{"error": message})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		response.Header().Set("Referrer-Policy", "no-referrer")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(response, request)
	})
}

func limitRequestTarget(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if len(request.RequestURI) > 4096 || strings.Count(request.URL.RawQuery, "&") > 16 {
			writeError(response, http.StatusRequestURITooLong, "request target is too long")
			return
		}
		next.ServeHTTP(response, request)
	})
}
