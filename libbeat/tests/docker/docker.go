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
	"io"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/menderesk/beats/v7/libbeat/common/docker"
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
		return "", errors.Wrap(err, "creating container")
	}

	if err := c.cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", errors.Wrap(err, "starting container")
	}

	return resp.ID, nil
}

// imagePull pulls an image
func (c Client) imagePull(image string) (err error) {
	ctx := context.Background()
	_, _, err = c.cli.ImageInspectWithRaw(ctx, image)
	if err == nil {
		// Image already available, do nothing
		return nil
	}
	for retry := 0; retry < 3; retry++ {
		err = func() error {
			respBody, err := c.cli.ImagePull(ctx, image, types.ImagePullOptions{})
			if err != nil {
				return errors.Wrapf(err, "pullling image %s", image)
			}
			defer respBody.Close()

			// Read all the response, to be sure that the pull has finished before returning.
			_, err = io.Copy(ioutil.Discard, respBody)
			if err != nil {
				return errors.Wrapf(err, "reading response for image %s", image)
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
	return c.cli.ContainerRemove(ctx, ID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
}

// Close closes the underlying client
func (c *Client) Close() error {
	return c.cli.Close()
}
