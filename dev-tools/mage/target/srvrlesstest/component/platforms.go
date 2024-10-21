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

package component

import (
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/utils"
	goruntime "runtime"
	"strings"

	"github.com/elastic/go-sysinfo"
)

const (
	// Container represents running inside a container
	Container = "container"
	// Darwin represents running on Mac OSX
	Darwin = "darwin"
	// Linux represents running on Linux
	Linux = "linux"
	// Windows represents running on Windows
	Windows = "windows"
)

const (
	// AMD64 represents the amd64 architecture
	AMD64 = "amd64"
	// ARM64 represents the arm64 architecture
	ARM64 = "arm64"
)

// Platform defines the platform that a component can support
type Platform struct {
	OS   string
	Arch string
	GOOS string
}

// Platforms is an array of platforms.
type Platforms []Platform

// GlobalPlatforms defines the platforms that a component can support
var GlobalPlatforms = Platforms{
	{
		OS:   Container,
		Arch: AMD64,
		GOOS: Linux,
	},
	{
		OS:   Container,
		Arch: ARM64,
		GOOS: Linux,
	},
	{
		OS:   Darwin,
		Arch: AMD64,
		GOOS: Darwin,
	},
	{
		OS:   Darwin,
		Arch: ARM64,
		GOOS: Darwin,
	},
	{
		OS:   Linux,
		Arch: AMD64,
		GOOS: Linux,
	},
	{
		OS:   Linux,
		Arch: ARM64,
		GOOS: Linux,
	},
	{
		OS:   Windows,
		Arch: AMD64,
		GOOS: Windows,
	},
}

// String returns the platform string identifier.
func (p *Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Arch)
}

// Exists returns true if the
func (p Platforms) Exists(platform string) bool {
	pieces := strings.SplitN(platform, "/", 2)
	if len(pieces) != 2 {
		return false
	}
	for _, platform := range p {
		if platform.OS == pieces[0] && platform.Arch == pieces[1] {
			return true
		}
	}
	return false
}

// UserDetail provides user specific information on the running platform.
type UserDetail struct {
	Root bool
}

// PlatformDetail is platform that has more detail information about the running platform.
type PlatformDetail struct {
	Platform

	NativeArch string
	Family     string
	Major      int
	Minor      int

	User UserDetail
}

// PlatformModifier can modify the platform details before the runtime specifications are loaded.
type PlatformModifier func(detail PlatformDetail) PlatformDetail

// LoadPlatformDetail loads the platform details for the current system.
func LoadPlatformDetail(modifiers ...PlatformModifier) (PlatformDetail, error) {
	hasRoot, err := utils.HasRoot()
	if err != nil {
		return PlatformDetail{}, err
	}
	info, err := sysinfo.Host()
	if err != nil {
		return PlatformDetail{}, err
	}
	os := info.Info().OS
	nativeArch := info.Info().NativeArchitecture
	if nativeArch == "x86_64" {
		// go-sysinfo Architecture and NativeArchitecture prefer x64_64
		// but GOARCH prefers amd64
		nativeArch = "amd64"
	}
	if nativeArch == "aarch64" {
		// go-sysinfo Architecture and NativeArchitecture prefer aarch64
		// but GOARCH prefers arm64
		nativeArch = "arm64"
	}
	detail := PlatformDetail{
		Platform: Platform{
			OS:   goruntime.GOOS,
			Arch: goruntime.GOARCH,
			GOOS: goruntime.GOOS,
		},
		NativeArch: nativeArch,
		Family:     os.Family,
		Major:      os.Major,
		Minor:      os.Minor,
		User: UserDetail{
			Root: hasRoot,
		},
	}
	for _, modifier := range modifiers {
		detail = modifier(detail)
	}
	return detail, nil
}
