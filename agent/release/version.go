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

package release

import "time"

// version is the current version of the agent.
var version = "8.0.0"

// buildHash is the hash of the current build.
var commit = "<unknown>"

// buildTime when the binary was build
var buildTime = "<unknown>"

// qualifier returns the version qualifier like alpha1.
var qualifier = ""

// Commit returns the current build hash or unkown if it was not injected in the build process.
func Commit() string {
	return commit
}

// BuildTime returns the build time of the binaries.
func BuildTime() time.Time {
	t, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		return time.Time{}
	}
	return t
}

// Version returns the version of the application.
func Version() string {
	if qualifier == "" {
		return version
	}
	return version + "-" + qualifier
}
