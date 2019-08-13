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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// HostInfo exposes information about started scenario
type HostInfo interface {
	// Host returns an address as host:port that can be used to connect to
	// a running service.
	Host() string

	// HostForPort returns an address as host:port that can be used to
	// connect to a running service that has multiple exposed ports. The
	// address returned is the one that can be used to connect to the
	// indicated exposed port.
	HostForPort(port int) string
}

// EnsureUp starts all the requested services (must be defined in docker-compose.yml)
// with a default timeout of 300 seconds
func EnsureUp(t testing.TB, service string, options ...UpOption) HostInfo {
	t.Helper()

	if hostInfo := HostInfoFromEnv(t, service); hostInfo != nil {
		return hostInfo
	}

	compose, err := getComposeProject(os.Getenv("DOCKER_COMPOSE_PROJECT_NAME"))
	if err != nil {
		t.Fatal(err)
	}
	defer compose.Close()

	// Kill no longer used containers
	err = compose.KillOld([]string{service})
	if err != nil {
		t.Fatal(err)
	}

	upOptions := UpOptions{
		Timeout: 60 * time.Second,
		Create: CreateOptions{
			Build:         true,
			ForceRecreate: true,
		},
	}
	for _, option := range options {
		option(&upOptions)
	}

	// Start container
	err = compose.Start(service, upOptions)
	if err != nil {
		t.Fatal("failed to start service", service, err)
	}

	// Wait for health
	err = compose.Wait(upOptions.Timeout, service)
	if err != nil {
		t.Fatal(err)
	}

	// Get host information
	host, err := compose.HostInformation(service)
	if err != nil {
		t.Fatalf("getting host for %s", service)
	}

	return host
}

// EnsureUpWithTimeout starts all the requested services (must be defined in docker-compose.yml)
// Wait for `timeout` seconds for health
func EnsureUpWithTimeout(t testing.TB, timeout int, service string) HostInfo {
	return EnsureUp(t, service, UpWithTimeout(time.Duration(timeout)*time.Second))
}

// HostInfoFromEnv gets the host information to use for the test from environment variables.
func HostInfoFromEnv(t testing.TB, service string) HostInfo {
	// If an environment variable with the form <SERVICE>_HOST is used, its value
	// is used as host instead of starting a new service.
	envVar := fmt.Sprintf("%s_HOST", strings.ToUpper(service))
	host := os.Getenv(envVar)
	if host != "" {
		return &staticHostInfo{host: host}
	}

	// The NO_COMPOSE env variables makes it possible to skip the starting of the environment.
	// This is useful if the service is already running locally.
	// Kept for historical reasons, now it only complains if the host environment
	// variable is not set.
	noCompose, err := strconv.ParseBool(os.Getenv("NO_COMPOSE"))
	if err == nil && noCompose {
		t.Fatalf("%s environment variable must be set as the host:port where %s is running", envVar, service)
	}

	return nil
}

type staticHostInfo struct {
	host string
}

func (i *staticHostInfo) Host() string {
	return i.host
}

func (i *staticHostInfo) HostForPort(int) string {
	return i.host
}

func findComposePath() (string, error) {
	// find docker-compose
	path, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if path == "/" {
			break
		}

		composePath := filepath.Join(path, "docker-compose.yml")
		if _, err = os.Stat(composePath); err == nil {
			return composePath, nil
		}
		path = filepath.Dir(path)
	}

	return "", errors.New("docker-compose.yml not found")
}

func getComposeProject(name string) (*Project, error) {
	path, err := findComposePath()
	if err != nil {
		return nil, err
	}

	return NewProject(
		name,
		[]string{
			path,
		},
	)
}
