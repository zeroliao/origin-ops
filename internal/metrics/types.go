package metrics

import "time"

type Snapshot struct {
	Time             time.Time `json:"time"`
	CPUPercent       float64   `json:"cpuPercent"`
	MemoryUsedBytes  uint64    `json:"memoryUsedBytes"`
	MemoryTotalBytes uint64    `json:"memoryTotalBytes"`
	DiskUsedBytes    uint64    `json:"diskUsedBytes"`
	DiskTotalBytes   uint64    `json:"diskTotalBytes"`
	RXBytesPerSecond float64   `json:"rxBytesPerSecond"`
	TXBytesPerSecond float64   `json:"txBytesPerSecond"`
	Load1            float64   `json:"load1"`
	Load5            float64   `json:"load5"`
	Load15           float64   `json:"load15"`
}

type Collector interface {
	Collect() (Snapshot, error)
}

type Appender interface {
	Append(Snapshot) error
}
