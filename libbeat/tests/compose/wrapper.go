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
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

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

	Environment []string
}

type wrapperContainer struct {
	info types.Container
}

func (c *wrapperContainer) Name() string {
	return c.ServiceName()
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

var statusOldRe = regexp.MustCompile(`(\d+) (minutes|hours)`)

func (c *wrapperContainer) Old() bool {
	match := statusOldRe.FindStringSubmatch(c.info.Status)
	if len(match) < 3 {
		return false
	}
	n, _ := strconv.Atoi(match[1])
	unit := match[2]
	switch unit {
	case "minutes":
		return n > 5
	default:
		return true
	}
}

// privateHost returns the address of the container, it should be reachable
// from the host if docker is being run natively. To be used when the tests
// are run from another container in the same network. It also works when
// running from the hoist network if the docker daemon runs natively.
func (c *wrapperContainer) privateHost(port int) string {
	var ip string
	for _, net := range c.info.NetworkSettings.Networks {
		if len(net.IPAddress) > 0 {
			ip = net.IPAddress
			break
		}
	}
	if len(ip) == 0 {
		return ""
	}

	for _, info := range c.info.Ports {
		if info.PublicPort != uint16(0) && (port == 0 || info.PrivatePort == uint16(port)) {
			return net.JoinHostPort(ip, strconv.Itoa(int(info.PrivatePort)))
		}
	}
	return ""
}

// exposedHost returns the exposed address in the host, can be used when the
// test is run from the host network. Recommended when using docker machines.
func (c *wrapperContainer) exposedHost(port int) string {
	for _, info := range c.info.Ports {
		if info.PublicPort != uint16(0) && (port == 0 || info.PrivatePort == uint16(port)) {
			return net.JoinHostPort("localhost", strconv.Itoa(int(info.PublicPort)))
		}
	}
	return ""
}

func (c *wrapperContainer) Host() string {
	return c.HostForPort(0)
}

func (c *wrapperContainer) HostForPort(port int) string {
	// TODO: Support multiple networks/ports
	if runtime.GOOS == "linux" {
		return c.privateHost(port)
	}
	// We can use `exposedHost()` in all platforms when we can use host
	// network in the metricbeat container
	return c.exposedHost(port)
}

func (d *wrapperDriver) LockFile() string {
	return d.Files[0] + ".lock"
}

func (d *wrapperDriver) cmd(ctx context.Context, command string, arg ...string) *exec.Cmd {
	var args []string
	args = append(args, "--no-ansi", "--project-name", d.Name)
	for _, f := range d.Files {
		args = append(args, "--file", f)
	}
	args = append(args, command)
	args = append(args, arg...)
	cmd := exec.CommandContext(ctx, "docker-compose", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if len(d.Environment) > 0 {
		cmd.Env = append(os.Environ(), d.Environment...)
	}
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

	for {
		// It can fail if we have reached some system limit, specially
		// number of networks, retry while the context is not done
		err := d.cmd(ctx, "up", args...).Run()
		if err == nil {
			return d.setupAdvertisedHost(ctx, service)
		}

		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return err
		}
	}
}

func writeToContainer(ctx context.Context, id, filename, content string) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	now := time.Now()
	err := tw.WriteHeader(&tar.Header{
		Typeflag:   tar.TypeReg,
		Name:       filepath.Base(filename),
		Mode:       0100644,
		Size:       int64(len(content)),
		ModTime:    now,
		AccessTime: now,
		ChangeTime: now,
	})
	if err != nil {
		return errors.Wrap(err, "failed to write tar header")
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return errors.Wrap(err, "failed to write tar file")
	}
	if err := tw.Close(); err != nil {
		return errors.Wrap(err, "failed to close tar")
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		return errors.Wrap(err, "failed to start docker client")
	}
	defer cli.Close()

	opts := types.CopyToContainerOptions{}
	err = cli.CopyToContainer(ctx, id, filepath.Dir(filename), bytes.NewReader(buf.Bytes()), opts)
	if err != nil {
		return errors.Wrapf(err, "failed to copy environment to container %s", id)
	}
	return nil
}

// setupAdvertisedHost adds a file to a container with its address, this can
// be used in services that need to configure an address to be advertised to
// clients.
func (d *wrapperDriver) setupAdvertisedHost(ctx context.Context, service string) error {
	containers, err := d.containers(ctx, Filter{State: AnyState}, service)
	if err != nil {
		return errors.Wrap(err, "setupAdvertisedHost")
	}
	if len(containers) == 0 {
		return errors.Errorf("no containers for service %s", service)
	}

	for _, c := range containers {
		w := &wrapperContainer{info: c}
		content := fmt.Sprintf("SERVICE_HOST=%s", w.Host())

		err := writeToContainer(ctx, c.ID, "/run/compose_env", content)
		if err != nil {
			return err
		}
	}
	return nil
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
	defer cli.Close()

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
