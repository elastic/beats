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

package define

import (
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/component"
)

const (
	// Default constant can be used as the default group for tests.
	Default = "default"
)

const (
	// Darwin is macOS platform
	Darwin = component.Darwin
	// Linux is Linux platform
	Linux = component.Linux
	// Windows is Windows platform
	Windows = component.Windows
	// Kubernetes is Kubernetes platform
	Kubernetes = "kubernetes"
)

const (
	// AMD64 is amd64 architecture
	AMD64 = component.AMD64
	// ARM64 is arm64 architecture
	ARM64 = component.ARM64
)

// OS defines an operating system, architecture, version and distribution combination.
type OS struct {
	// Type is the operating system type (darwin, linux or windows).
	//
	// This is always required to be defined on the OS structure.
	// If it is not defined the test runner will error.
	Type string `json:"type"`
	// Arch is the architecture type (amd64 or arm64).
	//
	// In the case that it's not provided the test will run on every
	// architecture that is supported.
	Arch string `json:"arch"`
	// Version is a specific version of the OS type to run this test on
	//
	// When defined the test runs on this specific version only. When not
	// defined the test is run on a selected version for this operating system.
	Version string `json:"version"`
	// Distro allows in the Linux case for a specific distribution to be
	// selected for running on. Example would be "ubuntu". In the Kubernetes case
	// for a specific distribution of kubernetes. Example would be "kind".
	Distro string `json:"distro"`
	// DockerVariant allows in the Kubernetes case for a specific variant to
	// be selected for running with. Example would be "wolfi".
	DockerVariant string `json:"docker_variant"`
}

// Validate returns an error if not valid.
func (o OS) Validate() error {
	if o.Type == "" {
		return errors.New("type must be defined")
	}
	if o.Type != Darwin && o.Type != Linux && o.Type != Windows && o.Type != Kubernetes {
		return errors.New("type must be either darwin, linux, windows, or kubernetes")
	}
	if o.Arch != "" {
		if o.Arch != AMD64 && o.Arch != ARM64 {
			return errors.New("arch must be either amd64 or arm64")
		}
		if o.Type == Windows && o.Arch == ARM64 {
			return errors.New("windows on arm64 not supported")
		}
	}
	if o.Distro != "" && (o.Type != Linux && o.Type != Kubernetes) {
		return errors.New("distro can only be set when type is linux or kubernetes")
	}
	if o.DockerVariant != "" && o.Type != Kubernetes {
		return errors.New("docker variant can only be set when type is kubernetes")
	}
	return nil
}

// Stack defines the stack required for the test.
type Stack struct {
	// Version defines a specific stack version to create for this test.
	//
	// In the case that no version is provided the same version being used for
	// the current test execution is used.
	Version string `json:"version"`
}

// Requirements defines the testing requirements for the test to run.
type Requirements struct {
	// Group must be set on each test to define which group the tests belongs to.
	// Tests that are in the same group are executed on the same runner.
	//
	// Useful when tests take a long time to complete and sharding them across multiple
	// hosts can improve the total amount of time to complete all the tests.
	Group string `json:"group"`

	// OS defines the operating systems this test can run on. In the case
	// multiple are provided the test is ran multiple times one time on each
	// combination.
	OS []OS `json:"os,omitempty"`

	// Stack defines the stack required for the test.
	Stack *Stack `json:"stack,omitempty"`

	// Local defines if this test can safely be performed on a local development machine.
	// If not set then the test will not be performed when local only testing is performed.
	//
	// This doesn't mean this test can only run locally. It will still run on defined OS's
	// when a full test run is performed.
	Local bool `json:"local"`

	// Sudo defines that this test must run under superuser permissions. On Mac and Linux the
	// test gets executed under sudo and on Windows it gets run under Administrator.
	Sudo bool `json:"sudo"`
}

// Validate returns an error if not valid.
func (r Requirements) Validate() error {
	if r.Group == "" {
		return errors.New("group is required")
	}
	for i, o := range r.OS {
		if err := o.Validate(); err != nil {
			return fmt.Errorf("invalid os %d: %w", i, err)
		}
	}
	return nil
}

// runtimeAllowed returns true if the runtime matches a valid OS.
func (r Requirements) runtimeAllowed(os string, arch string, version string, distro string) bool {
	if len(r.OS) == 0 {
		// all allowed
		return true
	}
	for _, o := range r.OS {
		if o.Type != Kubernetes && o.Type != os {
			// not valid on this runtime
			continue
		}
		if o.Arch != "" && o.Arch != arch {
			// not allowed on specific architecture
			continue
		}
		if o.Version != "" && o.Version != version {
			// not allowed on specific version
			continue
		}
		if o.Distro != "" && o.Distro != distro {
			// not allowed on specific distro
			continue
		}
		// allowed
		return true
	}
	// made it this far, not allowed
	return false
}
