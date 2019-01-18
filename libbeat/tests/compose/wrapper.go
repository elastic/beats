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

package compose

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
)

const (
	labelComposeService = "com.docker.compose.service"
	labelComposeProject = "com.docker.compose.project"
)

type wrapperDriver struct {
	Name  string
	Files []string
}

type wrapperContainer struct {
	info types.Container
}

func (c *wrapperContainer) ServiceName() string {
	return c.info.Labels[labelComposeService]
}

func (c *wrapperContainer) Healthy() bool {
	return strings.Contains(c.info.Status, "(healthy)")
}

func (c *wrapperContainer) Running() bool {
	return c.info.State == "running"
}

func (c *wrapperContainer) Old() bool {
	return strings.Contains(c.info.Status, "minute")
}

func (d *wrapperDriver) LockFile() string {
	return d.Files[0] + ".lock"
}

func (d *wrapperDriver) cmd(ctx context.Context, command string, arg ...string) *exec.Cmd {
	var args []string
	args = append(args, "--project-name", d.Name)
	for _, f := range d.Files {
		args = append(args, "--file", f)
	}
	args = append(args, command)
	args = append(args, arg...)
	cmd := exec.CommandContext(ctx, "docker-compose", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func (d *wrapperDriver) Up(ctx context.Context, opts UpOptions, service string) error {
	var args []string

	args = append(args, "-d")

	if opts.Create.Build {
		args = append(args, "--build")
	}

	if opts.Create.ForceRecreate {
		args = append(args, "--force-recreate")
	}

	if service != "" {
		args = append(args, service)
	}

	return d.cmd(ctx, "up", args...).Run()
}

func (d *wrapperDriver) Kill(ctx context.Context, signal string, service string) error {
	var args []string

	if signal != "" {
		args = append(args, "-s", signal)
	}

	if service != "" {
		args = append(args, service)
	}

	return d.cmd(ctx, "kill", args...).Run()
}

func (d *wrapperDriver) Ps(ctx context.Context, filter ...string) ([]ContainerStatus, error) {
	containers, err := d.containers(ctx, Filter{State: AnyState}, filter...)
	if err != nil {
		return nil, errors.Wrap(err, "ps")
	}

	ps := make([]ContainerStatus, len(containers))
	for i, c := range containers {
		ps[i] = &wrapperContainer{info: c}
	}
	return ps, nil
}

func (d *wrapperDriver) Containers(ctx context.Context, projectFilter Filter, filter ...string) ([]string, error) {
	containers, err := d.containers(ctx, projectFilter, filter...)
	if err != nil {
		return nil, errors.Wrap(err, "containers")
	}

	ids := make([]string, len(containers))
	for i := range containers {
		ids[i] = containers[i].ID
	}
	return ids, nil
}

func (d *wrapperDriver) containers(ctx context.Context, projectFilter Filter, filter ...string) ([]types.Container, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to start docker client")
	}

	var serviceFilters []filters.Args
	if len(filter) == 0 {
		f := makeFilter(d.Name, "", projectFilter)
		serviceFilters = append(serviceFilters, f)
	} else {
		for _, service := range filter {
			f := makeFilter(d.Name, service, projectFilter)
			serviceFilters = append(serviceFilters, f)
		}
	}

	var containers []types.Container
	for _, f := range serviceFilters {
		c, err := cli.ContainerList(ctx, types.ContainerListOptions{
			All:     true,
			Filters: f,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get container list")
		}
		containers = append(containers, c...)
	}

	return containers, nil
}

func makeFilter(project, service string, projectFilter Filter) filters.Args {
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("%s=%s", labelComposeProject, project))

	if service != "" {
		f.Add("label", fmt.Sprintf("%s=%s", labelComposeService, service))
	}

	switch projectFilter.State {
	case AnyState:
		// No filter
	case RunningState:
		f.Add("status", "running")
	case StoppedState:
		f.Add("status", "exited")
	}

	return f
}
