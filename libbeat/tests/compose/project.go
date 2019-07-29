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
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

// CreateOptions are the options when containers are created
type CreateOptions struct {
	Build         bool
	ForceRecreate bool
}

// UpOptions are the options when containers are started
type UpOptions struct {
	Create CreateOptions
}

// UpOption customizes UpOptions
type UpOption func(*UpOptions)

// RecreateOnUp enables or disables recreation of container on startup
func RecreateOnUp(recreate bool) UpOption {
	return func(opts *UpOptions) {
		opts.Create.ForceRecreate = recreate
	}
}

// Filter options for services
type Filter struct {
	State State
}

// State of a service for filtering
type State string

// Possible states of a service for filtering
const (
	AnyState     = State("")
	RunningState = State("running")
	StoppedState = State("stopped")
)

// Driver is the interface of docker compose implementations
type Driver interface {
	Up(ctx context.Context, opts UpOptions, service string) error
	Kill(ctx context.Context, signal string, service string) error
	Ps(ctx context.Context, filter ...string) ([]ContainerStatus, error)

	LockFile() string

	Close() error
}

// ContainerStatus is an interface to obtain the status of a container
type ContainerStatus interface {
	ServiceName() string
	Healthy() bool
	Running() bool
	Old() bool
	Host() string
	HostForPort(int) string
}

// Project is a docker-compose project
type Project struct {
	Driver
}

// NewProject creates a new docker-compose project
func NewProject(name string, files []string) (*Project, error) {
	if len(files) == 0 {
		return nil, errors.New("project needs at least one file")
	}
	if name == "" {
		name = filepath.Base(filepath.Dir(files[0]))
	}
	driver, err := NewWrapperDriver()
	if err != nil {
		return nil, err
	}
	driver.Name = name
	driver.Files = files
	return &Project{Driver: driver}, nil
}

// Start the container, unless it's running already
func (c *Project) Start(service string, upOptions ...UpOption) error {
	c.Lock()
	defer c.Unlock()

	options := UpOptions{
		Create: CreateOptions{
			Build:         true,
			ForceRecreate: true,
		},
	}

	for _, option := range upOptions {
		option(&options)
	}

	return c.Driver.Up(context.Background(), options, service)
}

// Wait ensures all wanted services are healthy. Wait loop (60s timeout)
func (c *Project) Wait(seconds int, services ...string) error {
	healthy := false
	timeout := time.Now().Add(time.Duration(seconds) * time.Second)
	for !healthy && time.Now().Before(timeout) {
		healthy = true

		servicesStatus, err := c.getServices(services...)
		if err != nil {
			return err
		}

		if len(servicesStatus) == 0 {
			healthy = false
		}

		for _, s := range servicesStatus {
			if !s.Healthy() {
				healthy = false
				break
			}
		}

		if healthy {
			break
		}

		time.Sleep(1 * time.Second)
	}

	if !healthy {
		return errors.New("Timeout waiting for services to be healthy")
	}
	return nil
}

// HostInformation gets the host information of a service
func (c *Project) HostInformation(service string) (ServiceInfo, error) {
	servicesStatus, err := c.getServices(service)
	if err != nil {
		return nil, err
	}

	if len(servicesStatus) == 0 {
		return nil, errors.New("no container running for service")
	}

	status, ok := servicesStatus[service]
	if !ok || status.Host() == "" {
		return nil, errors.New("unknown host:port for service")
	}

	return status, nil
}

// Kill a container
func (c *Project) Kill(service string) error {
	c.Lock()
	defer c.Unlock()

	return c.Driver.Kill(context.Background(), "KILL", service)
}

// KillOld kills old containers
func (c *Project) KillOld(except []string) error {
	// Do not kill ourselves ;)
	except = append(except, "beat")

	// These services take very long to start up and stop. If they are stopped
	// it can happen that an other package tries to start them at the same time
	// which leads to a conflict. We need a better solution long term but that should
	// solve the problem for now.
	except = append(except, "elasticsearch", "kibana", "logstash", "kubernetes")

	servicesStatus, err := c.getServices()
	if err != nil {
		return err
	}

	for _, s := range servicesStatus {
		// Ignore the ones we want
		if contains(except, s.Name()) {
			continue
		}

		if s.Running() && s.Old() {
			err = c.Kill(s.Name())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Lock acquires the lock (300s) timeout
// Normally it should only be seconds that the lock is used, but in some cases it can take longer.
func (c *Project) Lock() {
	timeout := time.Now().Add(300 * time.Second)
	infoShown := false
	for time.Now().Before(timeout) {
		file, err := os.OpenFile(c.LockFile(), os.O_CREATE|os.O_EXCL, 0500)
		file.Close()
		if err != nil {
			if !infoShown {
				logp.Info("docker-compose.yml is locked, waiting")
				infoShown = true
			}
			time.Sleep(1 * time.Second)
			continue
		}
		if infoShown {
			logp.Info("docker-compose.yml lock acquired")
		}
		return
	}

	// This should rarely happen as we lock for start only, less than a second
	panic(errors.New("Timeout waiting for lock, please remove docker-compose.yml.lock"))
}

// Unlock releases the project lock
func (c *Project) Unlock() {
	os.Remove(c.LockFile())
}

// ServiceInfo is an interface for objects that give information about running services
type ServiceInfo interface {
	Name() string
	Running() bool
	Healthy() bool

	// Has been up for too long?:
	Old() bool

	Host() string
	HostForPort(int) string
}

func (c *Project) getServices(filter ...string) (map[string]ServiceInfo, error) {
	c.Lock()
	defer c.Unlock()

	result := make(map[string]ServiceInfo)
	services, err := c.Driver.Ps(context.Background(), filter...)
	if err != nil {
		return nil, err
	}

	for _, c := range services {
		name := c.ServiceName()

		// In case of several (stopped) containers, always prefer info about running ones
		if r := result[name]; r != nil && r.Running() {
			continue
		}

		result[name] = &containerServiceInfo{c}
	}

	return result, nil
}

// containerServiceInfo wrapps a container status to provide information
// about the service implemented by the container
type containerServiceInfo struct {
	ContainerStatus
}

func (i *containerServiceInfo) Name() string {
	return i.ContainerStatus.ServiceName()
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if item == i {
			return true
		}
	}
	return false
}
