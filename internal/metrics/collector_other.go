//go:build !linux

package metrics

import "fmt"

type unsupportedCollector struct{}

func NewCollector() (Collector, error) {
	return unsupportedCollector{}, nil
}

func (unsupportedCollector) Collect() (Snapshot, error) {
	return Snapshot{}, fmt.Errorf("metric collection is supported only on Linux")
}
