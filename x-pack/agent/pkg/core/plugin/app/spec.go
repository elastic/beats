// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app

import (
	"os/user"
)

const (
	// ConfigurableGrpc is a flag telling agent that program has capability of Grpc server with a Config endpoint
	ConfigurableGrpc = "grpc"
	// ConfigurableFile is a flag telling agent that program has capability of being configured by accepting `-c filepath`
	// argument with a configuration file provided
	ConfigurableFile = "file"
)

// Specifier returns a process specification.
type Specifier interface {
	Spec() ProcessSpec
}

// ProcessSpec specifies a way of running a process
type ProcessSpec struct {
	// Binary path.
	BinaryPath string

	// Set of arguments.
	Args []string

	// Allows running third party application without
	// the requirement for Config endpoint
	// recognized options are: [grpc, file]
	Configurable  string
	Configuration map[string]interface{}

	// Under what user we can run the program. (example: apm-server is not running as root, isolation and cgroup)
	User  user.User
	Group user.Group

	// TODO: mapping transformation rules for configuration between agent.yml and to the beats.
}
