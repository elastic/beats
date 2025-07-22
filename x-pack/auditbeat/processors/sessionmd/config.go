// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package sessionmd

import "time"

// Config for add_session_metadata processor.
type config struct {
	// Backend specifies the data source for the processor. Possible values are `auto`, `procfs`, and `kernel_tracing`
	Backend string `config:"backend"`
	// PIDField specifies the event field used to locate the process ID
	PIDField string `config:"pid_field"`
	/// DBReaperPeriod specifies the interval of how often the backing process DB should remove orphaned and exited events.
	// Only valid for the `procfs` backend, or if `auto` falls back to `procfs`
	DBReaperPeriod time.Duration `config:"db_reaper_period"`
	// ReapProcesses, if enabled, will tell the process DB reaper thread to also remove orphaned process exec events, in addition to orphaned exit events and compleated process events.
	// This can result in data loss if auditbeat is running in an environment where it can't properly talk to procfs, but it can also reduce the memory footprint of auditbeat.
	// Only valid for the `procfs` backend.
	ReapProcesses bool `config:"reap_processes"`
}

func defaultConfig() config {
	return config{
		Backend:        "auto",
		PIDField:       "process.pid",
		DBReaperPeriod: time.Second * 30,
		ReapProcesses:  false,
	}
}
