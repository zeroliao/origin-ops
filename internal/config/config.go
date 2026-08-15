package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
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
	ListenAddress          string        `json:"listenAddress"`
	MetricsDir             string        `json:"metricsDir"`
	SamplingIntervalSecond int           `json:"samplingIntervalSeconds"`
	MaxQueryDays           int           `json:"maxQueryDays"`
	MaxChartPoints         int           `json:"maxChartPoints"`
	Applications           []Application `json:"applications"`
}

type Application struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	PublicURL     string   `json:"publicUrl"`
	Services      []string `json:"services"`
	HealthURL     string   `json:"healthUrl"`
	ReleaseRecord string   `json:"releaseRecord"`
}

var (
	applicationIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	serviceUnitPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.@-]*\.service$`)
)

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
	applicationIDs := make(map[string]struct{}, len(c.Applications))
	for _, application := range c.Applications {
		if err := application.Validate(); err != nil {
			return fmt.Errorf("applications[%q]: %w", application.ID, err)
		}
		if _, exists := applicationIDs[application.ID]; exists {
			return fmt.Errorf("applications[%q]: duplicate id", application.ID)
		}
		applicationIDs[application.ID] = struct{}{}
	}
	return nil
}

func (a Application) Validate() error {
	if !applicationIDPattern.MatchString(a.ID) {
		return fmt.Errorf("id must use lowercase letters, numbers, and hyphens")
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateWebURL(a.PublicURL, false); err != nil {
		return fmt.Errorf("publicUrl: %w", err)
	}
	if a.HealthURL != "" {
		if err := validateWebURL(a.HealthURL, true); err != nil {
			return fmt.Errorf("healthUrl: %w", err)
		}
	}
	if a.ReleaseRecord == "" {
		return fmt.Errorf("releaseRecord is required")
	}
	for _, service := range a.Services {
		if !serviceUnitPattern.MatchString(service) {
			return fmt.Errorf("service %q must be a .service unit name", service)
		}
	}
	return nil
}

func validateWebURL(value string, requireLoopback bool) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("must use http or https")
	}
	if parsed.User != nil {
		return fmt.Errorf("must not include credentials")
	}
	if requireLoopback && !isLoopbackHost(parsed.Hostname()) {
		return fmt.Errorf("must use a loopback host")
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
