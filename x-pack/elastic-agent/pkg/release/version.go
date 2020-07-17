// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package release

import (
	"strconv"
	"time"

	libbeatVersion "github.com/elastic/beats/v7/libbeat/version"
)

// snapshot is a flag marking build as a snapshot.
var snapshot = ""

// Commit returns the current build hash or unknown if it was not injected in the build process.
func Commit() string {
	return libbeatVersion.Commit()
}

// BuildTime returns the build time of the binaries.
func BuildTime() time.Time {
	return libbeatVersion.BuildTime()
}

// Version returns the version of the application.
func Version() string {
	return libbeatVersion.GetDefaultVersion()
}

// Snapshot returns true if binary was built as snapshot.
func Snapshot() bool {
	val, err := strconv.ParseBool(snapshot)
	return err == nil && val
}
