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
	"fmt"
	"io"
	"io/ioutil"

	"github.com/elastic/elastic-agent-autodiscover/docker"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// Client for Docker
type Client struct {
	cli *client.Client
}

// NewClient builds and returns a docker Client
func NewClient() (Client, error) {
	c, err := docker.NewClient(client.DefaultDockerHost, nil, nil)
	return Client{cli: c}, err
}

// ContainerStart pulls and starts the given container
func (c Client) ContainerStart(image string, cmd []string, labels map[string]string) (string, error) {
	err := c.imagePull(image)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	resp, err := c.cli.ContainerCreate(ctx, &container.Config{
		Image:  image,
		Cmd:    cmd,
		Labels: labels,
	}, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("starting container: %w", err)
	}

	return resp.ID, nil
}

// imagePull pulls an image
func (c Client) imagePull(img string) (err error) {
	ctx := context.Background()
	_, _, err = c.cli.ImageInspectWithRaw(ctx, img)
	if err == nil {
		// Image already available, do nothing
		return nil
	}
	for retry := 0; retry < 3; retry++ {
		err = func() error {
			respBody, err := c.cli.ImagePull(ctx, img, image.PullOptions{})
			if err != nil {
				return fmt.Errorf("pullling image %s: %w", img, err)
			}
			defer respBody.Close()

			// Read all the response, to be sure that the pull has finished before returning.
			_, err = io.Copy(ioutil.Discard, respBody)
			if err != nil {
				return fmt.Errorf("reading response for image %s: %w", img, err)
			}
			return nil
		}()
		if err == nil {
			break
		}
	}
	return
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

// ContainerInspect recovers information of the container
func (c Client) ContainerInspect(ID string) (types.ContainerJSON, error) {
	ctx := context.Background()
	return c.cli.ContainerInspect(ctx, ID)
}

// ContainerKill kills the given container
func (c Client) ContainerKill(ID string) error {
	ctx := context.Background()
	return c.cli.ContainerKill(ctx, ID, "KILL")
}

// ContainerRemove kills and removes the given container
func (c Client) ContainerRemove(ID string) error {
	ctx := context.Background()
	return c.cli.ContainerRemove(ctx, ID, container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

// Close closes the underlying client
func (c *Client) Close() error {
	return c.cli.Close()
}
