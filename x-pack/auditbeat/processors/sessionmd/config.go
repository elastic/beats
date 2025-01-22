// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package sessionmd

import "time"

// Config for add_session_metadata processor.
type config struct {
	Backend        string        `config:"backend"`
	PIDField       string        `config:"pid_field"`
	DBReaperPeriod time.Duration `config:"db_reaper_period"`
}

func defaultConfig() config {
	return config{
		Backend:        "auto",
		PIDField:       "process.pid",
		DBReaperPeriod: time.Second * 30,
	}
}
