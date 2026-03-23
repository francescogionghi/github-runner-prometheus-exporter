package validator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thineshsubramani/github-runner-prometheus-exporter/config"
	"github.com/thineshsubramani/github-runner-prometheus-exporter/internal/platform"
)

// Check if process exists
func ValidateRunnerProcess(processName string) error {
	if !platform.IsRunnerProcessRunning(processName) {
		return fmt.Errorf("runner process %q not running", processName)
	}
	return nil
}

// Validate directory and required files exist
func ValidatePaths(basePath string) error {
	// Check if base path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return fmt.Errorf("base path does not exist: %s", basePath)
	}

	// Validate .runner
	runnerPath := filepath.Join(basePath, ".runner")
	if _, err := os.Stat(runnerPath); os.IsNotExist(err) {
		return fmt.Errorf("missing .runner config at: %s", runnerPath)
	}

	// Validate _temp/_github_workflow/event.json
	// eventPath := filepath.Join(basePath, "_temp/_github_workflow/event.json")
	// if _, err := os.Stat(eventPath); os.IsNotExist(err) {
	// 	return fmt.Errorf("missing event.json at: %s", eventPath)
	// }

	return nil
}

func ValidateConfig(cfg *config.Config) error {
	runner, err := cfg.SelectedRunner()
	if err != nil {
		return err
	}

	if runner.Metrics.EnableJob {
		if runner.Logs.Worker == "" {
			return fmt.Errorf("runner %q: logs.worker is required when metrics.enable_job is true", runner.Name)
		}
		info, err := os.Stat(runner.Logs.Worker)
		if err != nil {
			return fmt.Errorf("runner %q: worker log path not accessible: %w", runner.Name, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("runner %q: logs.worker must be a directory", runner.Name)
		}
	}

	if runner.Metrics.EnableEvent {
		var eventPath string
		switch runner.Mode {
		case "test":
			eventPath = runner.Test.EventPath
		default:
			eventPath = runner.Logs.Event
		}
		if eventPath == "" {
			return fmt.Errorf("runner %q: event path is required when metrics.enable_event is true", runner.Name)
		}
		eventDir := filepath.Dir(eventPath)
		info, err := os.Stat(eventDir)
		if err != nil {
			return fmt.Errorf("runner %q: event directory not accessible: %w", runner.Name, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("runner %q: event path parent must be a directory", runner.Name)
		}
	}

	return nil
}
