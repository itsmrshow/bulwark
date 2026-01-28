package docker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Container represents a Docker container
type Container struct {
	ID      string
	Names   []string
	Image   string
	ImageID string
	Labels  map[string]string
	State   string
	Status  string
	Created int64
}

// ContainerJSON represents detailed container information
type ContainerJSON struct {
	ID              string
	Name            string
	Image           string
	State           ContainerState
	Config          *ContainerConfig
	NetworkSettings *NetworkSettings
}

// ContainerState represents container state
type ContainerState struct {
	Status     string
	Running    bool
	Paused     bool
	Restarting bool
	Health     *Health
}

// Health represents container health status
type Health struct {
	Status        string
	FailingStreak int
}

// ContainerConfig represents container configuration
type ContainerConfig struct {
	Image       string
	Labels      map[string]string
	Healthcheck *Healthcheck
}

// Healthcheck represents healthcheck configuration
type Healthcheck struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	Retries     int
	StartPeriod time.Duration
}

// NetworkSettings represents network configuration
type NetworkSettings struct {
	IPAddress string
	Ports     map[string][]PortBinding
}

// PortBinding represents port binding
type PortBinding struct {
	HostIP   string
	HostPort string
}

// ImageInspect represents image information
type ImageInspect struct {
	ID          string
	RepoTags    []string
	RepoDigests []string
	Created     string
	Size        int64
}

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
func (c *Client) ListContainers(ctx context.Context, all bool) ([]Container, error) {
	options := types.ContainerListOptions{
		All: all,
	}

	containers, err := c.cli.ContainerList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]Container, len(containers))
	for i, cont := range containers {
		result[i] = Container{
			ID:      cont.ID,
			Names:   cont.Names,
			Image:   cont.Image,
			ImageID: cont.ImageID,
			Labels:  cont.Labels,
			State:   cont.State,
			Status:  cont.Status,
			Created: cont.Created,
		}
	}

	return result, nil
}

// InspectContainer gets detailed container information
func (c *Client) InspectContainer(ctx context.Context, containerID string) (ContainerJSON, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return ContainerJSON{}, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	result := ContainerJSON{
		ID:    inspect.ID,
		Name:  inspect.Name,
		Image: inspect.Image,
		State: ContainerState{
			Status:     inspect.State.Status,
			Running:    inspect.State.Running,
			Paused:     inspect.State.Paused,
			Restarting: inspect.State.Restarting,
		},
	}

	if inspect.State.Health != nil {
		result.State.Health = &Health{
			Status:        inspect.State.Health.Status,
			FailingStreak: inspect.State.Health.FailingStreak,
		}
	}

	if inspect.Config != nil {
		result.Config = &ContainerConfig{
			Image:  inspect.Config.Image,
			Labels: inspect.Config.Labels,
		}
		if inspect.Config.Healthcheck != nil {
			result.Config.Healthcheck = &Healthcheck{
				Test:        inspect.Config.Healthcheck.Test,
				Interval:    inspect.Config.Healthcheck.Interval,
				Timeout:     inspect.Config.Healthcheck.Timeout,
				Retries:     inspect.Config.Healthcheck.Retries,
				StartPeriod: inspect.Config.Healthcheck.StartPeriod,
			}
		}
	}

	return result, nil
}

// ImagePull pulls an image from a registry
func (c *Client) ImagePull(ctx context.Context, ref string) error {
	out, err := c.cli.ImagePull(ctx, ref, types.ImagePullOptions{})
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
func (c *Client) ImageInspect(ctx context.Context, imageID string) (ImageInspect, error) {
	inspect, _, err := c.cli.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return ImageInspect{}, fmt.Errorf("failed to inspect image %s: %w", imageID, err)
	}

	return ImageInspect{
		ID:          inspect.ID,
		RepoTags:    inspect.RepoTags,
		RepoDigests: inspect.RepoDigests,
		Created:     inspect.Created,
		Size:        inspect.Size,
	}, nil
}

// ContainerRestart restarts a container
func (c *Client) ContainerRestart(ctx context.Context, containerID string) error {
	timeout := int(10) // seconds
	options := container.StopOptions{
		Timeout: &timeout,
	}
	if err := c.cli.ContainerRestart(ctx, containerID, options); err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}
	return nil
}

// ContainerLogs gets container logs
func (c *Client) ContainerLogs(ctx context.Context, containerID string, tail string) (io.ReadCloser, error) {
	options := types.ContainerLogsOptions{
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

// ListImages lists Docker images
func (c *Client) ListImages(ctx context.Context) ([]image.Summary, error) {
	images, err := c.cli.ImageList(ctx, types.ImageListOptions{
		All: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}
	return images, nil
}

// RemoveImage removes an image
func (c *Client) RemoveImage(ctx context.Context, imageID string, force bool) error {
	options := types.ImageRemoveOptions{
		Force: force,
	}
	_, err := c.cli.ImageRemove(ctx, imageID, options)
	if err != nil {
		return fmt.Errorf("failed to remove image %s: %w", imageID, err)
	}
	return nil
}

// PruneImages prunes unused images
func (c *Client) PruneImages(ctx context.Context) error {
	_, err := c.cli.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return fmt.Errorf("failed to prune images: %w", err)
	}
	return nil
}
