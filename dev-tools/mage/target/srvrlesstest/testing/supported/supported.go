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

package supported

import (
	"errors"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/kubernetes"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/linux"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/windows"
)

const (
	Rhel = "rhel"
	// Ubuntu is a Linux distro.
	Ubuntu = "ubuntu"
)

var (
	// ErrOSNotSupported returned when it's an unsupported OS.
	ErrOSNotSupported = errors.New("os/arch not currently supported")
)

var (
	// UbuntuAMD64_2404 - Ubuntu (amd64) 24.04
	UbuntuAMD64_2404 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Linux,
			Arch:    define.AMD64,
			Distro:  Ubuntu,
			Version: "24.04",
		},
		Runner: linux.DebianRunner{},
	}
	// UbuntuAMD64_2204 - Ubuntu (amd64) 22.04
	UbuntuAMD64_2204 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Linux,
			Arch:    define.AMD64,
			Distro:  Ubuntu,
			Version: "22.04",
		},
		Runner: linux.DebianRunner{},
	}
	// UbuntuAMD64_2004 - Ubuntu (amd64) 20.04
	UbuntuAMD64_2004 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Linux,
			Arch:    define.AMD64,
			Distro:  Ubuntu,
			Version: "20.04",
		},
		Runner: linux.DebianRunner{},
	}
	// UbuntuARM64_2404 - Ubuntu (arm64) 24.04
	UbuntuARM64_2404 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Linux,
			Arch:    define.ARM64,
			Distro:  Ubuntu,
			Version: "24.04",
		},
		Runner: linux.DebianRunner{},
	}
	// UbuntuARM64_2204 - Ubuntu (arm64) 22.04
	UbuntuARM64_2204 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Linux,
			Arch:    define.ARM64,
			Distro:  Ubuntu,
			Version: "22.04",
		},
		Runner: linux.DebianRunner{},
	}
	// UbuntuARM64_2004 - Ubuntu (arm64) 20.04
	UbuntuARM64_2004 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Linux,
			Arch:    define.ARM64,
			Distro:  Ubuntu,
			Version: "20.04",
		},
		Runner: linux.DebianRunner{},
	}
	// RhelAMD64_8 - RedHat Enterprise Linux (amd64) 8
	RhelAMD64_8 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Linux,
			Arch:    define.AMD64,
			Distro:  Rhel,
			Version: "8",
		},
		Runner: linux.RhelRunner{},
	}
	// WindowsAMD64_2022 - Windows (amd64) Server 2022
	WindowsAMD64_2022 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Windows,
			Arch:    define.AMD64,
			Version: "2022",
		},
		Runner: windows.WindowsRunner{},
	}
	// WindowsAMD64_2022_Core - Windows (amd64) Server 2022 Core
	WindowsAMD64_2022_Core = common.SupportedOS{
		OS: define.OS{
			Type:    define.Windows,
			Arch:    define.AMD64,
			Version: "2022-core",
		},
		Runner: windows.WindowsRunner{},
	}
	// WindowsAMD64_2019 - Windows (amd64) Server 2019
	WindowsAMD64_2019 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Windows,
			Arch:    define.AMD64,
			Version: "2019",
		},
		Runner: windows.WindowsRunner{},
	}
	// WindowsAMD64_2019_Core - Windows (amd64) Server 2019 Core
	WindowsAMD64_2019_Core = common.SupportedOS{
		OS: define.OS{
			Type:    define.Windows,
			Arch:    define.AMD64,
			Version: "2019-core",
		},
		Runner: windows.WindowsRunner{},
	}
	// WindowsAMD64_2016 - Windows (amd64) Server 2016
	WindowsAMD64_2016 = common.SupportedOS{
		OS: define.OS{
			Type:    define.Windows,
			Arch:    define.AMD64,
			Version: "2016",
		},
		Runner: windows.WindowsRunner{},
	}
	// WindowsAMD64_2016_Core - Windows (amd64) Server 2016 Core
	WindowsAMD64_2016_Core = common.SupportedOS{
		OS: define.OS{
			Type:    define.Windows,
			Arch:    define.AMD64,
			Version: "2016-core",
		},
		Runner: windows.WindowsRunner{},
	}
)

// supported defines the set of supported OS's.
//
// A provisioner might support a lesser number of this OS's, but the following
// are known to be supported by out OS runner logic.
//
// In the case that a batch is not specific on the version and/or distro the first
// one in this list will be picked. So it's best to place the one that we want the
// most testing at the top.
var supported = []common.SupportedOS{
	UbuntuAMD64_2404,
	UbuntuAMD64_2204,
	UbuntuAMD64_2004,
	UbuntuARM64_2404,
	UbuntuARM64_2204,
	UbuntuARM64_2004,
	RhelAMD64_8,
	WindowsAMD64_2022,
	WindowsAMD64_2022_Core,
	WindowsAMD64_2019,
	WindowsAMD64_2019_Core,
	// https://github.com/elastic/ingest-dev/issues/3484
	// WindowsAMD64_2016,
	// WindowsAMD64_2016_Core,
}

// init injects the kubernetes support list into the support list above
func init() {
	for _, k8sSupport := range kubernetes.GetSupported() {
		supported = append(supported, common.SupportedOS{
			OS:     k8sSupport,
			Runner: kubernetes.Runner{},
		})
	}
}

// osMatch returns true when the specific OS is a match for a non-specific OS.
func osMatch(specific define.OS, notSpecific define.OS) bool {
	if specific.Type != notSpecific.Type || specific.Arch != notSpecific.Arch {
		return false
	}
	if notSpecific.Distro != "" && specific.Distro != notSpecific.Distro {
		return false
	}
	if notSpecific.Version != "" && specific.Version != notSpecific.Version {
		return false
	}
	if notSpecific.DockerVariant != "" && specific.DockerVariant != notSpecific.DockerVariant {
		return false
	}
	return true
}

// getSupported returns all the supported based on the provided OS profile while using
// the provided platforms as a filter.
func getSupported(os define.OS, platforms []define.OS) ([]common.SupportedOS, error) {
	var match []common.SupportedOS
	for _, s := range supported {
		if osMatch(s.OS, os) && allowedByPlatforms(s.OS, platforms) {
			match = append(match, s)
		}
	}
	if len(match) > 0 {
		return match, nil
	}
	return nil, fmt.Errorf("%w: %s/%s", ErrOSNotSupported, os.Type, os.Arch)
}

// allowedByPlatforms determines if the os is in the allowed list of platforms.
func allowedByPlatforms(os define.OS, platforms []define.OS) bool {
	if len(platforms) == 0 {
		return true
	}
	for _, platform := range platforms {
		if ok := allowedByPlatform(os, platform); ok {
			return true
		}
	}
	return false
}

// allowedByPlatform determines if the platform allows this os.
func allowedByPlatform(os define.OS, platform define.OS) bool {
	if os.Type != platform.Type {
		return false
	}
	if platform.Arch == "" {
		// not specific on arch
		return true
	}
	if os.Arch != platform.Arch {
		return false
	}
	if platform.Type == define.Linux {
		// on linux distro is supported
		if platform.Distro == "" {
			// not specific on distro
			return true
		}
		if os.Distro != platform.Distro {
			return false
		}
	}
	if platform.Version == "" {
		// not specific on version
		return true
	}
	if os.Version != platform.Version {
		return false
	}
	if platform.Type == define.Kubernetes {
		// on kubernetes docker variant is supported
		if platform.DockerVariant == "" {
			return true
		}
		if os.DockerVariant != platform.DockerVariant {
			return false
		}
	}
	return true
}
