// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package sessionmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	enrichTests = []struct {
		testName      string
		mockProcesses []types.ProcessExecEvent
		config        Config
		input         beat.Event
		expected      beat.Event
		expect_error  bool
	}{
		{
			testName: "Enrich Process",
			config: Config{
				ReplaceFields: false,
				PidField:      "process.pid",
			},
			mockProcesses: []types.ProcessExecEvent{
				{
					Pids: types.PidInfo{
						Tid:  uint32(100),
						Tgid: uint32(100),
						Ppid: uint32(50),
						Pgid: uint32(100),
						Sid:  uint32(40),
					},
					Cwd:      "/",
					Filename: "/bin/ls",
				},
				{
					Pids: types.PidInfo{
						Tid:  uint32(50),
						Tgid: uint32(50),
						Ppid: uint32(40),
						Sid:  uint32(40),
					},
				},
				{
					Pids: types.PidInfo{
						Tid:  uint32(40),
						Tgid: uint32(40),
						Ppid: uint32(1),
						Sid:  uint32(1),
					},
				},
			},
			input: beat.Event{
				Fields: mapstr.M{
					"process": mapstr.M{
						"pid": uint32(100),
					},
				},
			},
			expected: beat.Event{
				Fields: mapstr.M{
					"process": mapstr.M{
						"executable":        "/bin/ls",
						"working_directory": "/",
						"pid":               uint32(100),
						"parent": mapstr.M{
							"pid": uint32(50),
						},
						"session_leader": mapstr.M{
							"pid": uint32(40),
						},
						"group_leader": mapstr.M{
							"pid": uint32(100),
						},
					},
				},
			},
			expect_error: false,
		},
		{
			testName: "No Pid Field in Event",
			config: Config{
				ReplaceFields: false,
				PidField:      "process.pid",
			},
			input: beat.Event{
				Fields: mapstr.M{
					"process": mapstr.M{
						"executable":        "ls",
						"working_directory": "/",
						"parent": mapstr.M{
							"pid": uint32(100),
						},
					},
				},
			},
			expect_error: true,
		},
		{
			testName: "Pid Not Number",
			config: Config{
				ReplaceFields: false,
				PidField:      "process.pid",
			},
			input: beat.Event{
				Fields: mapstr.M{
					"process": mapstr.M{
						"pid":               "xyz",
						"executable":        "ls",
						"working_directory": "/",
						"parent": mapstr.M{
							"pid": uint32(50),
						},
					},
				},
			},
			expect_error: true,
		},
		{
			testName: "PID not in DB",
			config: Config{
				ReplaceFields: false,
				PidField:      "process.pid",
			},
			input: beat.Event{
				Fields: mapstr.M{
					"process": mapstr.M{
						"pid":               "100",
						"executable":        "ls",
						"working_directory": "/",
						"parent": mapstr.M{
							"pid": uint32(100),
						},
					},
				},
			},
			expect_error: true,
		},
	}

	filterTests = []struct {
		testName string
		mx       mapstr.M
		my       mapstr.M
		expected bool
	}{
		{
			testName: "equal",
			mx: mapstr.M{
				"key1": "A",
				"key2": mapstr.M{
					"key2_2": 2.0,
				},
				"key3": 1,
			},
			my: mapstr.M{
				"key1": "A",
				"key2": mapstr.M{
					"key2_2": 2.0,
				},
				"key3": 1,
			},
			expected: true,
		},
		{
			testName: "mismatched values",
			mx: mapstr.M{
				"key1": "A",
				"key2": "B",
				"key3": "C",
			},
			my: mapstr.M{
				"key1": "A",
				"key2": "X",
				"key3": "C",
			},
			expected: false,
		},
		{
			testName: "ignore key only in 2nd map",
			mx: mapstr.M{
				"key1": "A",
				"key2": "B",
			},
			my: mapstr.M{
				"key1": "A",
				"key2": "B",
				"key3": "C",
			},
			expected: true,
		},
		{
			testName: "nested mismatch",
			mx: mapstr.M{
				"key1": "A",
				"key2": mapstr.M{
					"key2_2": "B",
				},
			},
			my: mapstr.M{
				"key1": "A",
				"key2": mapstr.M{
					"key2_2": 2.0,
				},
				"key3": 1,
			},
			expected: false,
		},
	}

	logger = logp.NewLogger("add_session_metadata_test")
)

func TestEnrich(t *testing.T) {
	for _, tt := range enrichTests {
		reader := procfs.NewMockReader()
		db := processdb.NewDB(reader, *logger)

		for _, ev := range tt.mockProcesses {
			db.InsertExec(ev)
		}
		s := addSessionMetadata{
			logger: logger,
			db:     db,
			config: tt.config,
		}

		// avoid taking address of loop variable
		i := tt.input
		actual, err := s.enrich(&i)
		if tt.expect_error {
			assert.Error(t, err, "%s: error unexpectedly nil", tt.testName)
		} else {
			assert.Nil(t, err, "%s: enrich error: %w", tt.testName, err)
			assert.NotNil(t, actual, "%s: returned nil event", tt.testName)

			//Validate output
			if diff := cmp.Diff(tt.expected.Fields, actual.Fields, ignoreMissingFrom(tt.expected.Fields)); diff != "" {
				t.Errorf("field mismatch:\n%s", diff)
			}
		}
	}
}

// IgnoreMissingFrom returns a filter that will ignore all fields missing from m
func ignoreMissingFrom(m mapstr.M) cmp.Option {
	return cmp.FilterPath(func(p cmp.Path) bool {
		mi, ok := p.Index(-1).(cmp.MapIndex)
		if !ok {
			return false
		}
		vx, _ := mi.Values()
		return !vx.IsValid()
	}, cmp.Ignore())
}

// TestFilter ensures `ignoreMissingFrom` filter is working as expected
// Note: This validates test code only
func TestFilter(t *testing.T) {
	for _, tt := range filterTests {
		if eq := cmp.Equal(tt.mx, tt.my, ignoreMissingFrom(tt.mx)); eq != tt.expected {
			t.Errorf("%s: unexpected comparator result", tt.testName)
		}
	}
}
