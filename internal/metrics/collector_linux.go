//go:build linux

package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type systemCounters struct {
	time        time.Time
	cpuTotal    uint64
	cpuIdle     uint64
	networkRX   uint64
	networkTX   uint64
	memoryTotal uint64
	memoryUsed  uint64
	load1       float64
	load5       float64
	load15      float64
}

type linuxCollector struct {
	previous *systemCounters
}

func NewCollector() (Collector, error) {
	return &linuxCollector{}, nil
}

func (c *linuxCollector) Collect() (Snapshot, error) {
	current, err := readSystemCounters()
	if err != nil {
		return Snapshot{}, err
	}
	if c.previous == nil {
		c.previous = &current
		time.Sleep(250 * time.Millisecond)
		current, err = readSystemCounters()
		if err != nil {
			return Snapshot{}, err
		}
	}

	diskUsed, diskTotal, err := readDiskUsage("/")
	if err != nil {
		return Snapshot{}, err
	}
	previous := c.previous
	c.previous = &current

	elapsed := current.time.Sub(previous.time).Seconds()
	if elapsed <= 0 || current.cpuTotal <= previous.cpuTotal {
		return Snapshot{}, fmt.Errorf("invalid metric counter interval")
	}
	cpuDelta := current.cpuTotal - previous.cpuTotal
	idleDelta := current.cpuIdle - previous.cpuIdle
	cpuPercent := 100 * float64(cpuDelta-idleDelta) / float64(cpuDelta)

	return Snapshot{
		Time:             current.time.UTC(),
		CPUPercent:       cpuPercent,
		MemoryUsedBytes:  current.memoryUsed,
		MemoryTotalBytes: current.memoryTotal,
		DiskUsedBytes:    diskUsed,
		DiskTotalBytes:   diskTotal,
		RXBytesPerSecond: counterRate(previous.networkRX, current.networkRX, elapsed),
		TXBytesPerSecond: counterRate(previous.networkTX, current.networkTX, elapsed),
		Load1:            current.load1,
		Load5:            current.load5,
		Load15:           current.load15,
	}, nil
}

func counterRate(previous, current uint64, elapsed float64) float64 {
	if current < previous {
		return 0
	}
	return float64(current-previous) / elapsed
}

func readSystemCounters() (systemCounters, error) {
	total, idle, err := readCPU()
	if err != nil {
		return systemCounters{}, err
	}
	memoryTotal, memoryUsed, err := readMemory()
	if err != nil {
		return systemCounters{}, err
	}
	rx, tx, err := readNetwork()
	if err != nil {
		return systemCounters{}, err
	}
	load1, load5, load15, err := readLoad()
	if err != nil {
		return systemCounters{}, err
	}
	return systemCounters{
		time: time.Now(), cpuTotal: total, cpuIdle: idle,
		networkRX: rx, networkTX: tx,
		memoryTotal: memoryTotal, memoryUsed: memoryUsed,
		load1: load1, load5: load5, load15: load15,
	}, nil
}

func readCPU() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, fmt.Errorf("read /proc/stat: %w", err)
	}
	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("parse /proc/stat: invalid cpu row")
	}
	var values []uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse /proc/stat: %w", err)
		}
		values = append(values, value)
	}
	var total uint64
	for _, value := range values {
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return total, idle, nil
}

func readMemory() (uint64, uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, fmt.Errorf("read /proc/meminfo: %w", err)
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		if key != "MemTotal" && key != "MemAvailable" {
			continue
		}
		value, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			return 0, 0, fmt.Errorf("parse /proc/meminfo: %w", parseErr)
		}
		values[key] = value * 1024
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("scan /proc/meminfo: %w", err)
	}
	total, totalOK := values["MemTotal"]
	available, availableOK := values["MemAvailable"]
	if !totalOK || !availableOK || available > total {
		return 0, 0, fmt.Errorf("parse /proc/meminfo: required values missing")
	}
	return total, total - available, nil
}

func readNetwork() (uint64, uint64, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, fmt.Errorf("read /proc/net/dev: %w", err)
	}
	defer file.Close()

	var rx, tx uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		separator := strings.IndexByte(line, ':')
		if separator < 0 {
			continue
		}
		name := strings.TrimSpace(line[:separator])
		if name == "lo" {
			continue
		}
		fields := strings.Fields(line[separator+1:])
		if len(fields) < 9 {
			return 0, 0, fmt.Errorf("parse /proc/net/dev: invalid row for %s", name)
		}
		rxValue, parseErr := strconv.ParseUint(fields[0], 10, 64)
		if parseErr != nil {
			return 0, 0, fmt.Errorf("parse /proc/net/dev rx: %w", parseErr)
		}
		txValue, parseErr := strconv.ParseUint(fields[8], 10, 64)
		if parseErr != nil {
			return 0, 0, fmt.Errorf("parse /proc/net/dev tx: %w", parseErr)
		}
		rx += rxValue
		tx += txValue
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("scan /proc/net/dev: %w", err)
	}
	return rx, tx, nil
}

func readLoad() (float64, float64, float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("read /proc/loadavg: %w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0, fmt.Errorf("parse /proc/loadavg: invalid row")
	}
	values := make([]float64, 3)
	for index := range values {
		values[index], err = strconv.ParseFloat(fields[index], 64)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("parse /proc/loadavg: %w", err)
		}
	}
	return values[0], values[1], values[2], nil
}

func readDiskUsage(path string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, fmt.Errorf("stat filesystem: %w", err)
	}
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	return total - available, total, nil
}
