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

package version

import (
	"runtime/debug"
	"sync/atomic"
	"time"
)

var (
	packageVersion atomic.Value
	vcsRevision    string
	vcsTime        time.Time
	qualifier      = ""
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.time":
				vcsTime, _ = time.Parse(time.RFC3339, setting.Value)
			case "vcs.revision":
				vcsRevision = setting.Value
			}
		}
	}
}

// VCSRevision returns the hash of the git commit used for the build.
func VCSRevision() string {
	return vcsRevision
}

// VCSTime returns the timestamp of the commit used for the build.
func VCSTime() time.Time {
	return vcsTime
}

// GetDefaultVersion returns the current version.
// If running in stand-alone mode, it's the libbeat version. If running in
// managed mode, a.k.a under the agent, it's the package version set using
// SetPackageVersion. If SetPackageVersion haven't been called, it reports the
// libbeat version
//
// This method is in a separate file as the version.go file is auto-generated.
func GetDefaultVersion() string {
	if v, ok := packageVersion.Load().(string); ok && v != "" {
		return v
	}

	if qualifier == "" {
		return defaultBeatVersion
	}
	return defaultBeatVersion + "-" + qualifier
}

// BuildTime exposes the compile-time build time information.
// It will represent the zero time instant if parsing fails.
//
// Deprecated: Use VCSTime
func BuildTime() time.Time {
	return VCSTime()
}

// Commit exposes the compile-time commit hash.
//
// Deprecated: Use VCSRevision
func Commit() string {
	return VCSRevision()
}

// SetPackageVersion sets the package version, overriding the defaultBeatVersion.
func SetPackageVersion(version string) {
	// Currently, the Elastic Agent does not perform any validation on the
	// package version, therefore, no validation is done here either.
	packageVersion.Store(version)
}
