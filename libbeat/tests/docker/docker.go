package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Client for Docker
type Client struct {
	cli *client.Client
}

// NewClient builds and returns a docker Client
func NewClient() (Client, error) {
	c, err := client.NewEnvClient()
	return Client{cli: c}, err
}

// ContainerStart pulls and starts the given container
func (c Client) ContainerStart(image string, cmd []string, labels map[string]string) (string, error) {
	ctx := context.Background()
	if _, err := c.cli.ImagePull(ctx, image, types.ImagePullOptions{}); err != nil {
		return "", err
	}

	resp, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image:  image,
		Cmd:    cmd,
		Labels: labels,
	}, nil, nil, "")
	if err != nil {
		return "", err
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	return resp.ID, nil
}

// ContainerWait waits for a container to finish
func (c Client) ContainerWait(ID string) error {
	ctx := context.Background()
	_, err := c.cli.ContainerWait(ctx, ID)
	return err
}

// ContainerKill kills the given container
func (c Client) ContainerKill(ID string) error {
	ctx := context.Background()
	return c.cli.ContainerKill(ctx, ID, "KILL")
}
