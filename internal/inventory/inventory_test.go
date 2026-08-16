package inventory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"origin-ops/internal/config"
)

func TestListReportsServiceAndHealthStates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	provider := New([]config.Application{{
		ID: "console", Name: "Console", Group: "平台工具", PublicURL: "https://console.example.com",
		Services: []string{"console.service", "worker.service"}, HealthURL: server.URL,
		ReleaseRecord: filepath.Join(t.TempDir(), "releases.jsonl"),
	}})
	provider.runSystemctl = func(_ context.Context, unit string) (string, error) {
		if unit == "worker.service" {
			return "", errors.New("unit unavailable")
		}
		return "Origin Ops console\nloaded\nactive\nrunning\n", nil
	}
	provider.now = func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) }

	applications := provider.List(context.Background())
	if len(applications) != 1 {
		t.Fatalf("application count = %d", len(applications))
	}
	application := applications[0]
	if application.Group != "平台工具" {
		t.Fatalf("application group = %q", application.Group)
	}
	if application.Services[0].Status != "running" || application.Services[1].Status != "unknown" {
		t.Fatalf("service states = %+v", application.Services)
	}
	if application.Services[0].Description != "Origin Ops console" {
		t.Fatalf("service description = %q", application.Services[0].Description)
	}
	if application.Health.Status != "healthy" || application.Health.CheckedAt.IsZero() {
		t.Fatalf("health = %+v", application.Health)
	}
}

func TestReleasesReadsNewestRecordsAndReportsUnknownApplication(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "releases.jsonl")
	content := "{\"version\":\"v1\",\"commit\":\"abc123\",\"status\":\"success\",\"startedAt\":\"2026-08-16T00:00:00Z\",\"finishedAt\":\"2026-08-16T00:01:00Z\",\"actor\":\"manual\",\"rollbackTarget\":\"v0\"}\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	provider := New([]config.Application{{ID: "console", Name: "Console", PublicURL: "https://console.example.com", ReleaseRecord: path}})
	releases, err := provider.Releases(context.Background(), "console")
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].Version != "v1" {
		t.Fatalf("releases = %+v", releases)
	}
	if _, err := provider.Releases(context.Background(), "missing"); !errors.Is(err, ErrApplicationNotFound) {
		t.Fatalf("missing application error = %v", err)
	}
}

func TestAppendReleaseWritesValidatedJSONLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "releases.jsonl")
	release := Release{
		Version: "v2", Commit: "def456", Status: "success",
		StartedAt:  time.Date(2026, 8, 16, 1, 0, 0, 0, time.UTC),
		FinishedAt: time.Date(2026, 8, 16, 1, 1, 0, 0, time.UTC),
		Actor:      "deploy-script", RollbackTarget: "v1",
	}
	if err := AppendRelease(path, release); err != nil {
		t.Fatal(err)
	}
	releases, err := readReleases(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || releases[0].Version != "v2" {
		t.Fatalf("releases = %+v", releases)
	}
}
