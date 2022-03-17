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

package ecs

import (
	"time"
)

// These fields contain information about an installed software package. It
// contains general information about a package, such as name, version or size.
// It also contains installation details, such as time or location.
type Package struct {
	// Package name
	Name string `ecs:"name"`

	// Package version
	Version string `ecs:"version"`

	// Additional information about the build version of the installed package.
	// For example use the commit SHA of a non-released package.
	BuildVersion string `ecs:"build_version"`

	// Description of the package.
	Description string `ecs:"description"`

	// Package size in bytes.
	Size int64 `ecs:"size"`

	// Time when package was installed.
	Installed time.Time `ecs:"installed"`

	// Path where the package is installed.
	Path string `ecs:"path"`

	// Package architecture.
	Architecture string `ecs:"architecture"`

	// Checksum of the installed package for verification.
	Checksum string `ecs:"checksum"`

	// Indicating how the package was installed, e.g. user-local, global.
	InstallScope string `ecs:"install_scope"`

	// License under which the package was released.
	// Use a short name, e.g. the license identifier from SPDX License List
	// where possible (https://spdx.org/licenses/).
	License string `ecs:"license"`

	// Home page or reference URL of the software in this package, if
	// available.
	Reference string `ecs:"reference"`

	// Type of package.
	// This should contain the package file type, rather than the package
	// manager name. Examples: rpm, dpkg, brew, npm, gem, nupkg, jar.
	Type string `ecs:"type"`
}
