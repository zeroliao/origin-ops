package inventory

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"origin-ops/internal/config"
)

const maxReleaseRecords = 100

type Service struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LoadState   string `json:"loadState"`
	ActiveState string `json:"activeState"`
	SubState    string `json:"subState"`
	Status      string `json:"status"`
}

type Health struct {
	Configured bool      `json:"configured"`
	Status     string    `json:"status"`
	CheckedAt  time.Time `json:"checkedAt,omitempty"`
}

type Application struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PublicURL   string    `json:"publicUrl"`
	Services    []Service `json:"services"`
	Health      Health    `json:"health"`
}

type Release struct {
	Version        string    `json:"version"`
	Commit         string    `json:"commit"`
	Status         string    `json:"status"`
	StartedAt      time.Time `json:"startedAt"`
	FinishedAt     time.Time `json:"finishedAt"`
	Actor          string    `json:"actor"`
	RollbackTarget string    `json:"rollbackTarget"`
}

type commandRunner func(context.Context, string) (string, error)

type Provider struct {
	applications map[string]config.Application
	orderedIDs   []string
	client       *http.Client
	runSystemctl commandRunner
	now          func() time.Time
}

func New(applications []config.Application) *Provider {
	entries := make(map[string]config.Application, len(applications))
	orderedIDs := make([]string, 0, len(applications))
	for _, application := range applications {
		entries[application.ID] = application
		orderedIDs = append(orderedIDs, application.ID)
	}
	return &Provider{
		applications: entries,
		orderedIDs:   orderedIDs,
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		runSystemctl: systemctlShow,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (p *Provider) List(ctx context.Context) []Application {
	applications := make([]Application, 0, len(p.orderedIDs))
	for _, id := range p.orderedIDs {
		applications = append(applications, p.applicationStatus(ctx, p.applications[id]))
	}
	return applications
}

func (p *Provider) Releases(ctx context.Context, id string) ([]Release, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	application, exists := p.applications[id]
	if !exists {
		return nil, ErrApplicationNotFound
	}
	return readReleases(application.ReleaseRecord)
}

var ErrApplicationNotFound = errors.New("application not found")

func (p *Provider) applicationStatus(ctx context.Context, configured config.Application) Application {
	services := make([]Service, 0, len(configured.Services))
	for _, service := range configured.Services {
		services = append(services, p.serviceStatus(ctx, service))
	}
	return Application{
		ID:          configured.ID,
		Name:        configured.Name,
		Description: configured.Description,
		PublicURL:   configured.PublicURL,
		Services:    services,
		Health:      p.healthStatus(ctx, configured.HealthURL),
	}
}

func (p *Provider) serviceStatus(ctx context.Context, unit string) Service {
	output, err := p.runSystemctl(ctx, unit)
	if err != nil {
		return Service{Name: unit, Status: "unknown"}
	}
	values := strings.Split(strings.TrimSpace(output), "\n")
	service := Service{Name: unit, Status: "unknown"}
	if len(values) > 0 {
		service.Description = values[0]
	}
	if len(values) > 1 {
		service.LoadState = values[1]
	}
	if len(values) > 2 {
		service.ActiveState = values[2]
	}
	if len(values) > 3 {
		service.SubState = values[3]
	}
	switch service.ActiveState {
	case "active":
		service.Status = "running"
	case "inactive", "failed", "activating", "deactivating":
		service.Status = service.ActiveState
	}
	return service
}

func (p *Provider) healthStatus(ctx context.Context, rawURL string) Health {
	if rawURL == "" {
		return Health{Status: "not_configured"}
	}
	checkedAt := p.now()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Health{Configured: true, Status: "unavailable", CheckedAt: checkedAt}
	}
	response, err := p.client.Do(request)
	if err != nil {
		return Health{Configured: true, Status: "unavailable", CheckedAt: checkedAt}
	}
	response.Body.Close()
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return Health{Configured: true, Status: "healthy", CheckedAt: checkedAt}
	}
	return Health{Configured: true, Status: "unhealthy", CheckedAt: checkedAt}
}

func systemctlShow(ctx context.Context, unit string) (string, error) {
	command := exec.CommandContext(ctx, "systemctl", "show", unit, "--property=Description,LoadState,ActiveState,SubState", "--value")
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func readReleases(path string) ([]Release, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Release{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open release record: %w", err)
	}
	defer file.Close()

	releases := make([]Release, 0, maxReleaseRecords)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 64<<10)
	line := 0
	for scanner.Scan() {
		line++
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var release Release
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&release); err != nil {
			return nil, fmt.Errorf("decode release record line %d: %w", line, err)
		}
		if err := validateRelease(release); err != nil {
			return nil, fmt.Errorf("release record line %d: %w", line, err)
		}
		if len(releases) == maxReleaseRecords {
			copy(releases, releases[1:])
			releases[len(releases)-1] = release
		} else {
			releases = append(releases, release)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read release record: %w", err)
	}
	return releases, nil
}

func AppendRelease(path string, release Release) error {
	if err := validateRelease(release); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create release record directory: %w", err)
	}
	line, err := json.Marshal(release)
	if err != nil {
		return fmt.Errorf("encode release record: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return fmt.Errorf("open release record: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append release record: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync release record: %w", err)
	}
	return nil
}

func validateRelease(release Release) error {
	if release.Version == "" || release.Commit == "" || release.Status == "" || release.StartedAt.IsZero() || release.FinishedAt.IsZero() {
		return fmt.Errorf("version, commit, status, startedAt, and finishedAt are required")
	}
	if release.FinishedAt.Before(release.StartedAt) {
		return fmt.Errorf("finishedAt must not be before startedAt")
	}
	switch release.Status {
	case "success", "failed", "rolled_back":
	default:
		return fmt.Errorf("status must be success, failed, or rolled_back")
	}
	if len(release.Version) > 128 || len(release.Commit) > 128 || len(release.Actor) > 128 || len(release.RollbackTarget) > 128 {
		return fmt.Errorf("release record fields must not exceed 128 characters")
	}
	return nil
}
