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

package common

import (
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"strings"
)

// Config provides the configuration for running the runner.
type Config struct {
	AgentVersion   string
	StateDir       string
	ReleaseVersion string
	StackVersion   string
	BuildDir       string
	GOVersion      string
	RepoDir        string
	DiagnosticsDir string

	// Platforms filters the tests to only run on the provided list
	// of platforms even if the tests supports more than what is
	// defined in this list.
	Platforms []string

	// Packages filters the tests to only run on the provided list
	// of platforms even if the tests supports more than what is
	// defined in this list.
	Packages []string

	// BinaryName is the name of the binary package under test, i.e, elastic-agent, metricbeat, etc
	// this is used to copy the .tar.gz to the remote host
	BinaryName string

	// Groups filters the tests to only run tests that are part of
	// the groups defined in this list.
	Groups []string

	// Matrix enables matrix testing. This explodes each test to
	// run on all supported platforms the runner supports.
	Matrix bool

	// SingleTest only has the runner run that specific test.
	SingleTest string

	// VerboseMode passed along a verbose mode flag to tests
	VerboseMode bool

	// Timestamp enables timestamps on the console output.
	Timestamp bool

	// Testflags contains extra go test flags to be set when running tests
	TestFlags string

	// ExtraEnv contains extra environment flags to pass to the runner.
	ExtraEnv map[string]string
}

// Validate returns an error if the information is invalid.
func (c *Config) Validate() error {
	if c.AgentVersion == "" {
		return errors.New("field AgentVersion must be set")
	}
	if c.StackVersion == "" {
		return errors.New("field StackVersion must be set")
	}
	if c.BuildDir == "" {
		return errors.New("field BuildDir must be set")
	}
	if c.GOVersion == "" {
		return errors.New("field GOVersion must be set")
	}
	if c.RepoDir == "" {
		return errors.New("field RepoDir must be set")
	}
	if c.StateDir == "" {
		return errors.New("field StateDir must be set")
	}
	_, err := c.GetPlatforms()
	if err != nil {
		return err
	}
	return nil
}

// GetPlatforms returns the defined platforms for the configuration.
func (c *Config) GetPlatforms() ([]define.OS, error) {
	var each []define.OS
	for _, platform := range c.Platforms {
		o, err := parsePlatform(platform)
		if err != nil {
			return nil, err
		}
		each = append(each, o)
	}
	return each, nil
}

func parsePlatform(platform string) (define.OS, error) {
	separated := strings.Split(platform, "/")
	var os define.OS
	switch len(separated) {
	case 0:
		return define.OS{}, fmt.Errorf("failed to parse platform string %q: empty string", platform)
	case 1:
		os = define.OS{Type: separated[0]}
	case 2:
		os = define.OS{Type: separated[0], Arch: separated[1]}
	case 3:
		if separated[0] == define.Linux {
			os = define.OS{Type: separated[0], Arch: separated[1], Distro: separated[2]}
		} else {
			os = define.OS{Type: separated[0], Arch: separated[1], Version: separated[2]}
		}
	case 4:
		if separated[0] == define.Linux {
			os = define.OS{Type: separated[0], Arch: separated[1], Distro: separated[2], Version: separated[3]}
		} else if separated[0] == define.Kubernetes {
			os = define.OS{Type: separated[0], Arch: separated[1], Version: separated[2], DockerVariant: separated[3]}
		} else {
			return define.OS{}, fmt.Errorf("failed to parse platform string %q: more than 2 separators", platform)
		}
	default:
		return define.OS{}, fmt.Errorf("failed to parse platform string %q: more than 3 separators", platform)
	}
	if err := os.Validate(); err != nil {
		return define.OS{}, fmt.Errorf("failed to parse platform string %q: %w", platform, err)
	}
	return os, nil
}
