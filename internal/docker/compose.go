package docker

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ComposeRunner executes docker compose commands
type ComposeRunner struct {
	composeBinary string
}

// NewComposeRunner creates a new compose runner
func NewComposeRunner() *ComposeRunner {
	// Try to find docker compose (v2 plugin style) first, fall back to docker-compose
	composeBinary := "docker"
	if _, err := exec.LookPath("docker-compose"); err == nil {
		composeBinary = "docker-compose"
	}

	return &ComposeRunner{
		composeBinary: composeBinary,
	}
}

// buildCommand builds a docker compose command
func (r *ComposeRunner) buildCommand(ctx context.Context, composePath string, args ...string) *exec.Cmd {
	return r.buildCommandWithFiles(ctx, []string{composePath}, args...)
}

// buildCommandWithFiles builds a docker compose command with one or more compose files.
func (r *ComposeRunner) buildCommandWithFiles(ctx context.Context, composePaths []string, args ...string) *exec.Cmd {
	if len(composePaths) == 0 {
		return exec.CommandContext(ctx, "false")
	}

	var cmdArgs []string

	if r.composeBinary == "docker" {
		// Docker v2 plugin style: docker compose
		cmdArgs = append(cmdArgs, "compose")
		for _, composePath := range composePaths {
			cmdArgs = append(cmdArgs, "-f", composePath)
		}
		cmdArgs = append(cmdArgs, args...)
	} else {
		// Legacy docker-compose
		for _, composePath := range composePaths {
			cmdArgs = append(cmdArgs, "-f", composePath)
		}
		cmdArgs = append(cmdArgs, args...)
	}

	cmd := exec.CommandContext(ctx, r.composeBinary, cmdArgs...)
	cmd.Dir = filepath.Dir(composePaths[0])

	return cmd
}

// Pull pulls images for a service
func (r *ComposeRunner) Pull(ctx context.Context, composePath, service string) error {
	args := []string{"pull"}
	if service != "" {
		args = append(args, service)
	}

	cmd := r.buildCommand(ctx, composePath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

// Up starts services
func (r *ComposeRunner) Up(ctx context.Context, composePath, service string, forceRecreate bool) error {
	return r.upWithFiles(ctx, []string{composePath}, service, forceRecreate)
}

// UpWithOverride starts services using a base compose file plus a one-off override file.
func (r *ComposeRunner) UpWithOverride(ctx context.Context, composePath, overridePath, service string, forceRecreate bool) error {
	files := []string{composePath}
	if strings.TrimSpace(overridePath) != "" {
		files = append(files, overridePath)
	}
	return r.upWithFiles(ctx, files, service, forceRecreate)
}

func (r *ComposeRunner) upWithFiles(ctx context.Context, composeFiles []string, service string, forceRecreate bool) error {
	args := []string{"up", "-d"}

	if forceRecreate {
		args = append(args, "--force-recreate")
	}

	if service != "" {
		args = append(args, "--no-deps", service)
	}

	cmd := r.buildCommandWithFiles(ctx, composeFiles, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to up: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

// Down stops services
func (r *ComposeRunner) Down(ctx context.Context, composePath string) error {
	cmd := r.buildCommand(ctx, composePath, "down")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to down: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	return nil
}

// Ps lists services
func (r *ComposeRunner) Ps(ctx context.Context, composePath string) ([]string, error) {
	cmd := r.buildCommand(ctx, composePath, "ps", "--services", "--filter", "status=running")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to ps: %w\nstderr: %s", err, stderr.String())
	}

	services := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	return services, nil
}

// Config validates and parses compose file
func (r *ComposeRunner) Config(ctx context.Context, composePath string) (string, error) {
	cmd := r.buildCommand(ctx, composePath, "config")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to config: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// FindComposeFiles finds all docker-compose files in a directory tree
func FindComposeFiles(root string) ([]string, error) {
	var composeFiles []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip permission errors and other access issues, continue walking
			if os.IsPermission(err) || os.IsNotExist(err) {
				return filepath.SkipDir
			}
			// For other errors, skip this path but continue
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Match docker-compose*.yml, docker-compose*.yaml, compose.yml, compose.yaml
		name := info.Name()
		if strings.HasPrefix(name, "docker-compose") && (strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) {
			composeFiles = append(composeFiles, path)
		} else if name == "compose.yml" || name == "compose.yaml" {
			composeFiles = append(composeFiles, path)
		}

		return nil
	})

	// Note: err can still be non-nil for fatal errors, but we've handled
	// permission errors gracefully above
	if err != nil {
		return composeFiles, fmt.Errorf("walk completed with errors: %w", err)
	}

	return composeFiles, nil
}
