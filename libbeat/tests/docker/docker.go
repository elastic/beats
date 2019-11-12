// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
	waitC, errC := c.cli.ContainerWait(ctx, ID, container.WaitConditionNotRunning)
	select {
	case <-waitC:
	case err := <-errC:
		return err
	}
	return nil
}

// ContainerKill kills the given container
func (c Client) ContainerKill(ID string) error {
	ctx := context.Background()
	return c.cli.ContainerKill(ctx, ID, "KILL")
}
