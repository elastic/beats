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
	"fmt"
	"os"
	"strings"
	"time"
)

type CreateOptions struct {
	ForceBuild bool
}

type UpOptions struct {
	Create CreateOptions
}

type Filter map[string]string

type Driver interface {
	Up(ctx context.Context, opts UpOptions, service string) error
	Kill(ctx context.Context, signal string, service string) error
	Ps(ctx context.Context, filter ...string) ([]map[string]string, error)
	Containers(ctx context.Context, projectFilter Filter, filter ...string) ([]string, error)

	LockFile() string
}

// docker-compose project wrapper
type Project struct {
	Driver
}

func NewProject(name string, files []string) (*Project, error) {
	if len(files) == 0 {
		return nil, errors.New("project needs at least one file")
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
			ForceBuild: true,
		},
	}, service)
}

// Ensure all wanted services are healthy. Wait loop (60s timeout)
func (c *Project) Wait(seconds int, services ...string) error {
	healthy := false
	for !healthy && seconds > 0 {
		healthy = true

		servicesStatus, err := c.getServices(services...)
		if err != nil {
			return err
		}

		for _, s := range servicesStatus {
			if !s.Healthy {
				healthy = false
				break
			}
		}

		time.Sleep(1 * time.Second)
		seconds--
	}

	if !healthy {
		return errors.New("Timeout waiting for services to be healthy")
	}
	return nil
}

func (c *Project) Kill(service string) error {
	c.Lock()
	defer c.Unlock()

	return c.Driver.Kill(context.Background(), "KILL", service)
}

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
func (c *Project) Lock() {
	seconds := 300
	for seconds > 0 {
		file, err := os.OpenFile(c.LockFile(), os.O_CREATE|os.O_EXCL, 0500)
		file.Close()
		if err != nil {
			fmt.Println("docker-compose.yml is locked, waiting")
			time.Sleep(1 * time.Second)
			seconds--
			continue
		}
		return
	}

	// This should rarely happen as we lock for start only, less than a second
	panic(errors.New("Timeout waiting for lock, please remove docker-compose.yml.lock"))
}

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

	containers, err := c.Driver.Containers(context.Background(), Filter{"State": "Running"}, filter...)
	if err != nil {
		return nil, err
	}

	for _, c := range services {
		name := strings.Split(c["Name"], "_")[1]
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
		service.Healthy = strings.Contains(c["State"], "(healthy)")
		service.Running = contains(containers, c["Id"])
		if service.Healthy {
			service.Old = oldRegexp.MatchString(c["State"])
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
