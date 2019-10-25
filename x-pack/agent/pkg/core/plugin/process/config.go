// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package process

import "time"

// Config for fine tuning new process
type Config struct {
	MinPortNumber int           `yaml:"min_port" config:"min_port"`
	MaxPortNumber int           `yaml:"max_port" config:"max_port"`
	SpawnTimeout  time.Duration `yaml:"spawn_timeout" config:"spawn_timeout"`

	// Transport is one of `unix` or `tcp`. `unix` uses unix sockets and is not supported on windows.
	// Windows falls back to `tcp` regardless of configuration.
	// With invalid configuration fallback to `tcp` is used as well.
	Transport string

	// TODO: cgroups and namespaces
}
