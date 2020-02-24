// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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
