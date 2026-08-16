package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadUsesDefaultsWithoutPath(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ListenAddress != "127.0.0.1:9080" {
		t.Fatalf("ListenAddress = %q", cfg.ListenAddress)
	}
	if cfg.SamplingIntervalSecond != 60 || cfg.MaxQueryDays != 31 || cfg.MaxChartPoints != 240 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{"listenAddress":"127.0.0.1:9080","metricsDir":"metrics","samplingIntervalSeconds":60,"maxQueryDays":31,"maxChartPoints":240,"token":"secret"}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load error = %v", err)
	}
}

func TestValidateRejectsUnsafeSamplingInterval(t *testing.T) {
	cfg := Default()
	cfg.SamplingIntervalSecond = 1
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted a one-second sampling interval")
	}
}

func TestValidateRejectsPublicListenAddress(t *testing.T) {
	cfg := Default()
	cfg.ListenAddress = "0.0.0.0:9080"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted a public listen address")
	}
}

func TestValidateApplicationRestrictsHealthChecksToLoopback(t *testing.T) {
	cfg := Default()
	cfg.Applications = []Application{{
		ID:            "console",
		Name:          "Console",
		PublicURL:     "https://console.example.com",
		Services:      []string{"console.service"},
		HealthURL:     "https://example.com/health",
		ReleaseRecord: "/var/lib/origin-ops/releases/console.jsonl",
	}}
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateApplicationAcceptsConfiguredReadOnlySources(t *testing.T) {
	cfg := Default()
	cfg.Applications = []Application{{
		ID:            "console",
		Name:          "Console",
		PublicURL:     "https://console.example.com",
		Services:      []string{"console-api.service", "console-worker.service"},
		HealthURL:     "http://127.0.0.1:9080/health",
		ReleaseRecord: "/var/lib/origin-ops/releases/console.jsonl",
	}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateApplicationAllowsMissingPublicURL(t *testing.T) {
	cfg := Default()
	cfg.Applications = []Application{{
		ID:            "system-services",
		Name:          "System services",
		Services:      []string{"cron.service"},
		ReleaseRecord: "/var/lib/origin-ops/releases/system-services.jsonl",
	}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate error = %v", err)
	}
}
