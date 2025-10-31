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
