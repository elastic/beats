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
	"path/filepath"
	"syscall"
	"time"

	"github.com/pkg/errors"

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
	// Containers(ctx context.Context, projectFilter Filter, filter ...string) ([]string, error)

	LockFile() string
}

// ContainerStatus is an interface to obtain the status of a container
type ContainerStatus interface {
	ServiceName() string
	Healthy() bool
	Running() bool
	Old() bool
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
	return &Project{
		&wrapperDriver{
			Name:  name,
			Files: files,
		},
	}, nil
}

// Start the container, unless it's running already
func (c *Project) Start(service string) error {
	servicesStatus, err := c.getServices(service)
	if err != nil {
		return err
	}

	if servicesStatus[service] != nil {
		if servicesStatus[service].Running {
			// Someone is running it
			return nil
		}
	}

	c.Lock()
	defer c.Unlock()

	return c.Driver.Up(context.Background(), UpOptions{
		Create: CreateOptions{
			Build:         true,
			ForceRecreate: true,
		},
	}, service)
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
			if !s.Healthy {
				healthy = false
				break
			}
		}

		time.Sleep(1 * time.Second)
	}

	if !healthy {
		return errors.New("timeout waiting for services to be healthy")
	}
	return nil
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
		if contains(except, s.Name) {
			continue
		}

		if s.Old {
			err = c.Kill(s.Name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Lock acquires the lock (300s) timeout
// Normally it should only be seconds that the lock is used, but in some cases it can take longer.
// Pid is written to the lock file, and it is used to check if process holding the process is still
// alive to avoid deadlocks on unexpected finalizations.
func (c *Project) Lock() {
	timeout := time.Now().Add(300 * time.Second)
	infoShown := false
	for time.Now().Before(timeout) {
		if acquireLock(c.LockFile()) {
			if infoShown {
				logp.Info("%s lock acquired", c.LockFile())
			}
			return
		}

		if stalledLock(c.LockFile()) {
			if err := os.Remove(c.LockFile()); err == nil {
				logp.Info("Stalled lockfile %s removed", c.LockFile())
				continue
			}
		}

		if !infoShown {
			logp.Info("%s is locked, waiting", c.LockFile())
			infoShown = true
		}
		time.Sleep(1 * time.Second)
	}

	// This should rarely happen as we lock for start only, less than a second
	panic(errors.New("Timeout waiting for lock"))
}

func acquireLock(path string) bool {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0700)
	if err != nil {
		return false
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%d", os.Getpid())
	if err != nil {
		panic(errors.Wrap(err, "Failed to write pid to lock file"))
	}
	return true
}

// stalledLock checks if process holding the lock is still alive
func stalledLock(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 0500)
	if err != nil {
		return false
	}
	defer file.Close()

	var pid int
	fmt.Fscanf(file, "%d", &pid)

	return !processExists(pid)
}

func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if process == nil || err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

// Unlock releases the project lock
func (c *Project) Unlock() {
	os.Remove(c.LockFile())
}

type serviceInfo struct {
	Name    string
	Running bool
	Healthy bool

	// Has been up for too long?:
	Old bool
}

func (c *Project) getServices(filter ...string) (map[string]*serviceInfo, error) {
	c.Lock()
	defer c.Unlock()

	result := make(map[string]*serviceInfo)
	services, err := c.Driver.Ps(context.Background(), filter...)
	if err != nil {
		return nil, err
	}

	for _, c := range services {
		name := c.ServiceName()

		// In case of several (stopped) containers, always prefer info about running ones
		if result[name] != nil {
			if result[name].Running {
				continue
			}
		}

		service := &serviceInfo{
			Name: name,
		}
		// fill details:
		service.Healthy = c.Healthy()
		service.Running = c.Running()
		if service.Healthy {
			service.Old = c.Old()
		}
		result[name] = service
	}

	return result, nil
}

func contains(list []string, item string) bool {
	for _, i := range list {
		if item == i {
			return true
		}
	}
	return false
}
