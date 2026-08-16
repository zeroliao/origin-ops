package releasecli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAppendsOnlyToConfiguredApplication(t *testing.T) {
	directory := t.TempDir()
	recordPath := filepath.Join(directory, "releases", "console.jsonl")
	configPath := filepath.Join(directory, "config.json")
	configBody := `{"listenAddress":"127.0.0.1:9080","metricsDir":"metrics","authFile":"auth/users.json","samplingIntervalSeconds":60,"maxQueryDays":31,"maxChartPoints":240,"applications":[{"id":"console","name":"Console","description":"Operations console","publicUrl":"","services":[],"healthUrl":"","releaseRecord":"` + filepath.ToSlash(recordPath) + `"}]}`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"append", "--config", configPath, "--application", "console",
		"--version", "v1", "--commit", "abc123", "--status", "success",
		"--started-at", "2026-08-16T01:00:00Z", "--finished-at", "2026-08-16T01:01:00Z",
		"--actor", "deploy-script", "--rollback-target", "v0",
	}
	var output bytes.Buffer
	if err := Run(args, &output); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"version":"v1"`) || output.String() != "release recorded for console\n" {
		t.Fatalf("record = %s, output = %q", data, output.String())
	}
	args[4] = "missing"
	if err := Run(args, &output); err == nil {
		t.Fatal("unconfigured application was accepted")
	}
}
