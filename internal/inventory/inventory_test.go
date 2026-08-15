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
		ID: "console", Name: "Console", PublicURL: "https://console.example.com",
		Services: []string{"console.service", "worker.service"}, HealthURL: server.URL,
		ReleaseRecord: filepath.Join(t.TempDir(), "releases.jsonl"),
	}})
	provider.runSystemctl = func(_ context.Context, unit string) (string, error) {
		if unit == "worker.service" {
			return "", errors.New("unit unavailable")
		}
		return "loaded\nactive\nrunning\n", nil
	}
	provider.now = func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) }

	applications := provider.List(context.Background())
	if len(applications) != 1 {
		t.Fatalf("application count = %d", len(applications))
	}
	application := applications[0]
	if application.Services[0].Status != "running" || application.Services[1].Status != "unknown" {
		t.Fatalf("service states = %+v", application.Services)
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
