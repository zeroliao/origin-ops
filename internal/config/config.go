package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

const (
	defaultListenAddress          = "127.0.0.1:9080"
	defaultMetricsDir             = "data/metrics"
	defaultSamplingIntervalSecond = 60
	defaultMaxQueryDays           = 31
	defaultMaxChartPoints         = 240
)

type Config struct {
	ListenAddress          string `json:"listenAddress"`
	MetricsDir             string `json:"metricsDir"`
	SamplingIntervalSecond int    `json:"samplingIntervalSeconds"`
	MaxQueryDays           int    `json:"maxQueryDays"`
	MaxChartPoints         int    `json:"maxChartPoints"`
}

func Default() Config {
	return Config{
		ListenAddress:          defaultListenAddress,
		MetricsDir:             defaultMetricsDir,
		SamplingIntervalSecond: defaultSamplingIntervalSecond,
		MaxQueryDays:           defaultMaxQueryDays,
		MaxChartPoints:         defaultMaxChartPoints,
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode config: multiple JSON values")
		}
		return fmt.Errorf("decode config: %w", err)
	}
	return nil
}

func (c Config) Validate() error {
	if c.ListenAddress == "" {
		return fmt.Errorf("listenAddress is required")
	}
	host, _, err := net.SplitHostPort(c.ListenAddress)
	if err != nil {
		return fmt.Errorf("listenAddress must include a valid host and port: %w", err)
	}
	if !isLoopbackHost(host) {
		return fmt.Errorf("listenAddress must use a loopback host")
	}
	if c.MetricsDir == "" {
		return fmt.Errorf("metricsDir is required")
	}
	if c.SamplingIntervalSecond < 10 || c.SamplingIntervalSecond > 3600 {
		return fmt.Errorf("samplingIntervalSeconds must be between 10 and 3600")
	}
	if c.MaxQueryDays < 1 || c.MaxQueryDays > 366 {
		return fmt.Errorf("maxQueryDays must be between 1 and 366")
	}
	if c.MaxChartPoints < 1 || c.MaxChartPoints > 2000 {
		return fmt.Errorf("maxChartPoints must be between 1 and 2000")
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func (c Config) SamplingInterval() time.Duration {
	return time.Duration(c.SamplingIntervalSecond) * time.Second
}
