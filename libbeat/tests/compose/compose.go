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
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// EnsureUp starts all the requested services (must be defined in docker-compose.yml)
// with a default timeout of 300 seconds
func EnsureUp(t *testing.T, services ...string) {
	EnsureUpWithTimeout(t, 300, services...)
}

// EnsureUpWithTimeout starts all the requested services (must be defined in docker-compose.yml)
// Wait for `timeout` seconds for health
func EnsureUpWithTimeout(t *testing.T, timeout int, services ...string) {
	// The NO_COMPOSE env variables makes it possible to skip the starting of the environment.
	// This is useful if the service is already running locally.
	if noCompose, err := strconv.ParseBool(os.Getenv("NO_COMPOSE")); err == nil && noCompose {
		return
	}

	compose, err := getComposeProject()
	if err != nil {
		t.Fatal(err)
	}

	// Kill no longer used containers
	err = compose.KillOld(services)
	if err != nil {
		t.Fatal(err)
	}

	for _, service := range services {
		err = compose.Start(service)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Wait for health
	err = compose.Wait(timeout, services...)
	if err != nil {
		t.Fatal(err)
	}
}

func getComposeProject() (*Project, error) {
	// find docker-compose
	path, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		if path == "/" {
			return nil, errors.New("docker-compose.yml not found")
		}

		if _, err = os.Stat(filepath.Join(path, "docker-compose.yml")); err != nil {
			path = filepath.Dir(path)
		} else {
			break
		}
	}

	return NewProject(
		os.Getenv("DOCKER_COMPOSE_PROJECT_NAME"),
		[]string{
			filepath.Join(path, "docker-compose.yml"),
		},
	)
}
