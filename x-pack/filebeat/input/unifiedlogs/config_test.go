// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build darwin

package unifiedlogs

import (
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConfig(t *testing.T) {
	const cfgYaml = `
archive_file: /path/to/file.logarchive
trace_file: /path/to/file.tracev3
start: 2024-12-04 13:46:00+0200
end: 2024-12-04 13:46:00+0200
predicate:
- pid == 1
process:
- sudo
source: true
info: true
debug: true
backtrace: true
signpost: true
unreliable: true
mach_continuous_time: true
backfill: true
`

	expected := config{
		ShowConfig: showConfig{
			ArchiveFile: "/path/to/file.logarchive",
			TraceFile:   "/path/to/file.tracev3",
			Start:       "2024-12-04 13:46:00+0200",
			End:         "2024-12-04 13:46:00+0200",
		},
		CommonConfig: commonConfig{
			Predicate:          []string{"pid == 1"},
			Process:            []string{"sudo"},
			Source:             true,
			Info:               true,
			Debug:              true,
			Backtrace:          true,
			Signpost:           true,
			Unreliable:         true,
			MachContinuousTime: true,
		},
		Backfill: true,
	}

	c := conf.MustNewConfigFrom(cfgYaml)
	cfg := defaultConfig()
	assert.NoError(t, c.Unpack(&cfg))
	assert.EqualValues(t, expected, cfg)
}
