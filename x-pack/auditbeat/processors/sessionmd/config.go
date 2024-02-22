// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package sessionmd

// Config for add_session_metadata processor.
type config struct {
	Backend       string `config:"backend"`
	ReplaceFields bool   `config:"replace_fields"`
	PIDField      string `config:"pid_field"`
}

func defaultConfig() config {
	return config{
		Backend:       "auto",
		ReplaceFields: false,
		PIDField:      "process.pid",
	}
}
