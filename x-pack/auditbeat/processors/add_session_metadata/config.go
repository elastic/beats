// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package add_session_metadata

// Config for add_session_metadata processor.
type Config struct {
	Backend       string `config:"backend"`
	ReplaceFields bool   `config:"replace_fields"`
	PidField      string `config:"pid_field"`
}

func defaultConfig() Config {
	return Config{
		Backend:       "ebpf",
		ReplaceFields: false,
		PidField:      "process.pid",
	}
}
