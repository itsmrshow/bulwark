package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/moby/moby/api/types"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

// Client wraps the Docker API client
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Client{cli: cli}, nil
}

// Close closes the Docker client
func (c *Client) Close() error {
	return c.cli.Close()
}

// Ping verifies connectivity to Docker daemon
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

// ListContainers lists all containers
func (c *Client) ListContainers(ctx context.Context, all bool) ([]types.Container, error) {
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	return containers, nil
}

// InspectContainer gets detailed container information
func (c *Client) InspectContainer(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	return inspect, nil
}

// ImagePull pulls an image from a registry
func (c *Client) ImagePull(ctx context.Context, ref string) error {
	out, err := c.cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", ref, err)
	}
	defer out.Close()

	// Read the output to ensure the pull completes
	_, err = io.Copy(io.Discard, out)
	if err != nil {
		return fmt.Errorf("failed to read pull output: %w", err)
	}

	return nil
}

// ImageTag tags an image
func (c *Client) ImageTag(ctx context.Context, source, target string) error {
	err := c.cli.ImageTag(ctx, source, target)
	if err != nil {
		return fmt.Errorf("failed to tag image %s as %s: %w", source, target, err)
	}
	return nil
}

// ImageInspect gets detailed image information
func (c *Client) ImageInspect(ctx context.Context, imageID string) (types.ImageInspect, error) {
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return types.ImageInspect{}, fmt.Errorf("failed to inspect image %s: %w", imageID, err)
	}
	return inspect, nil
}

// ContainerRestart restarts a container
func (c *Client) ContainerRestart(ctx context.Context, containerID string) error {
	timeout := 10 // seconds
	options := container.StopOptions{
		Timeout: &timeout,
	}

	err := c.cli.ContainerRestart(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}
	return nil
}

// ContainerLogs gets container logs
func (c *Client) ContainerLogs(ctx context.Context, containerID string, tail string) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
	}

	logs, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs for container %s: %w", containerID, err)
	}

	return logs, nil
}

// ContainerStats gets container stats (for health checking)
func (c *Client) ContainerStats(ctx context.Context, containerID string) (types.Stats, error) {
	stats, err := c.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return types.Stats{}, fmt.Errorf("failed to get stats for container %s: %w", containerID, err)
	}
	defer stats.Body.Close()

	var result types.Stats
	// Read the stats (first response is the latest)
	decoder := io.NopCloser(stats.Body)
	defer decoder.Close()

	return result, nil
}
