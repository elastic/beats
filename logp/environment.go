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

package logp

import "strings"

// Environment indicates the environment the logger is supped to be run in.
// The default logger configuration may be different for different environments.
type Environment int

const (
	// DefaultEnvironment is used if the environment the process runs in is not known.
	DefaultEnvironment Environment = iota

	// SystemdEnvironment indicates that the process is started and managed by systemd.
	SystemdEnvironment

	// ContainerEnvironment indicates that the process is running within a container (docker, k8s, rkt, ...).
	ContainerEnvironment

	// MacOSServiceEnvironment indicates that the process is running as a daemon on macOS (e.g. managed via launchctl).
	MacOSServiceEnvironment

	// WindowsServiceEnvironment indicates the the process is run as a windows service.
	WindowsServiceEnvironment

	// InvalidEnvironment indicates that the environment name given is unknown or invalid.
	InvalidEnvironment
)

const (
	defaultEnvironmentString        = "default"
	systemdEnvironmentString        = "systemd"
	containerEnvironmentString      = "container"
	macOSServiceEnvironmentString   = "macOS_service"
	windowsServiceEnvironmentString = "windows_service"
	invalidEnvironmentString        = "<invalid>"
)

// String returns the string representation the configured environment
func (v Environment) String() string {
	switch v {
	case DefaultEnvironment:
		return defaultEnvironmentString
	case SystemdEnvironment:
		return systemdEnvironmentString
	case ContainerEnvironment:
		return containerEnvironmentString
	case MacOSServiceEnvironment:
		return macOSServiceEnvironmentString
	case WindowsServiceEnvironment:
		return windowsServiceEnvironmentString
	default:
		return invalidEnvironmentString
	}
}

// ParseEnvironment returns the environment type by name.
// The parse is case insensitive.
// InvalidEnvironment is returned if the environment type is unknown.
func ParseEnvironment(in string) Environment {
	switch strings.ToLower(in) {
	case defaultEnvironmentString:
		return DefaultEnvironment
	case systemdEnvironmentString:
		return SystemdEnvironment
	case containerEnvironmentString:
		return ContainerEnvironment
	case macOSServiceEnvironmentString:
		return MacOSServiceEnvironment
	case windowsServiceEnvironmentString:
		return WindowsServiceEnvironment
	default:
		return InvalidEnvironment
	}
}
