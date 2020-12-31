// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"os/user"
)

// ProcessSpec specifies a way of running a process
type ProcessSpec struct {
	// Binary path.
	BinaryPath string

	// Set of arguments.
	Args          []string
	Configuration map[string]interface{}

	// Under what user we can run the program. (example: apm-server is not running as root, isolation and cgroup)
	User  user.User
	Group user.Group

	// TODO: mapping transformation rules for configuration between elastic-agent.yml and to the beats.
}
