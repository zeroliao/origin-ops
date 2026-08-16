package releasecli

import (
	"flag"
	"fmt"
	"io"
	"time"

	"origin-ops/internal/config"
	"origin-ops/internal/inventory"
)

func Run(args []string, stdout io.Writer) error {
	if len(args) == 0 || args[0] != "append" {
		return fmt.Errorf("release action must be append")
	}
	flags := flag.NewFlagSet("origin-ops release append", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", "", "configuration file")
	applicationID := flags.String("application", "", "configured application id")
	version := flags.String("version", "", "release version")
	commit := flags.String("commit", "", "release commit")
	status := flags.String("status", "", "success, failed, or rolled_back")
	startedAt := flags.String("started-at", "", "RFC3339 start time")
	finishedAt := flags.String("finished-at", "", "RFC3339 finish time")
	actor := flags.String("actor", "", "release actor")
	rollbackTarget := flags.String("rollback-target", "", "rollback target")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	recordPath := ""
	for _, application := range cfg.Applications {
		if application.ID == *applicationID {
			recordPath = application.ReleaseRecord
			break
		}
	}
	if recordPath == "" {
		return fmt.Errorf("application %q is not configured", *applicationID)
	}
	start, err := time.Parse(time.RFC3339, *startedAt)
	if err != nil {
		return fmt.Errorf("started-at must use RFC3339")
	}
	finish, err := time.Parse(time.RFC3339, *finishedAt)
	if err != nil {
		return fmt.Errorf("finished-at must use RFC3339")
	}
	release := inventory.Release{
		Version: *version, Commit: *commit, Status: *status,
		StartedAt: start, FinishedAt: finish, Actor: *actor, RollbackTarget: *rollbackTarget,
	}
	if err := inventory.AppendRelease(recordPath, release); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "release recorded for %s\n", *applicationID)
	return nil
}
