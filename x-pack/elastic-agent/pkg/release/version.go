// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package release

import (
	"strconv"
	"strings"
	"time"

	libbeatVersion "github.com/elastic/beats/v7/libbeat/version"
)

const (
	hashLen = 6
)

// snapshot is a flag marking build as a snapshot.
var snapshot = ""

// allowEmptyPgp is used as a debug flag and allows working
// without valid pgp
var allowEmptyPgp string

// allowUpgrade is used as a debug flag and allows working
// with upgrade without requiring Agent to be installed correctly
var allowUpgrade string

// Commit returns the current build hash or unknown if it was not injected in the build process.
func Commit() string {
	return libbeatVersion.Commit()
}

// ShortCommit returns commit up to 6 characters.
func ShortCommit() string {
	hash := Commit()
	if len(hash) > hashLen {
		hash = hash[:hashLen]
	}
	return hash
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

// VersionInfo is structure used by `version --yaml`.
type VersionInfo struct {
	Version   string    `yaml:"version"`
	Commit    string    `yaml:"commit"`
	BuildTime time.Time `yaml:"build_time"`
	Snapshot  bool      `yaml:"snapshot"`
}

// Info returns current version information.
func Info() VersionInfo {
	return VersionInfo{
		Version:   Version(),
		Commit:    Commit(),
		BuildTime: BuildTime(),
		Snapshot:  Snapshot(),
	}
}

// String returns the string format for the version information.
func (v VersionInfo) String() string {
	var sb strings.Builder

	sb.WriteString(v.Version)
	if v.Snapshot {
		sb.WriteString("-SNAPSHOT")
	}
	sb.WriteString(" (build: ")
	sb.WriteString(v.Commit)
	sb.WriteString(" at ")
	sb.WriteString(v.BuildTime.Format("2006-01-02 15:04:05 -0700 MST"))
	sb.WriteString(")")
	return sb.String()
}
