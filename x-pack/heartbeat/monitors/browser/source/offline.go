// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || darwin || synthetics

package source

import "os"

var offlineEnvVar = "ELASTIC_SYNTHETICS_OFFLINE"

// Offline checks whether sources should act in offline mode, where
// calls to NPM are forbidden.
func Offline() bool {
	return os.Getenv(offlineEnvVar) == "true"
}

// GoOffline switches our current state to offline. Primarily for tests.
func GoOffline() {
	e := os.Setenv(offlineEnvVar, "true")
	if e != nil {
		panic("could not set offline env var!")
	}
}

// GoOffline switches our current state to offline. Primarily for tests.
func GoOnline() {
	e := os.Setenv(offlineEnvVar, "false")
	if e != nil {
		panic("could not set offline env var!")
	}
}
