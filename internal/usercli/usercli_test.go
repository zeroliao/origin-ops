package usercli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAddsAndListsUserWithoutPrintingPassword(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "config.json")
	configBody := `{"authFile":"` + filepath.ToSlash(filepath.Join(directory, "users.json")) + `","listenAddress":"127.0.0.1:9080","metricsDir":"data/metrics","samplingIntervalSeconds":60,"maxQueryDays":31,"maxChartPoints":240,"applications":[]}`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	password := "secret-password"
	if err := Run([]string{"add", "--config", configPath, "--username", "operator", "--password-stdin"}, strings.NewReader(password+"\n"), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), password) {
		t.Fatal("command output contains the password")
	}
	output.Reset()
	if err := Run([]string{"list", "--config", configPath}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "operator\tenabled\n" {
		t.Fatalf("list output = %q", output.String())
	}
}
