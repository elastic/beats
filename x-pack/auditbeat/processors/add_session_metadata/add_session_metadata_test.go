// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_session_metadata

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/processdb"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/procfs"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/add_session_metadata/types"

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

	logger = logp.NewLogger("add_session_metadata_test")
)

func TestEnrich(t *testing.T) {
	for _, tt := range enrichTests {
		reader := procfs.NewMockReader()
		db := processdb.NewDB(reader, *logger)

		for _, ev := range tt.mockProcesses {
			err := db.InsertExec(ev)
			assert.NoError(t, err, "%s: inserting exec to db: %w", tt.testName, err)
		}
		s := addSessionMetadata{
			logger: logger,
			db:     db,
			config: tt.config,
		}

		actual, err := s.enrich(&tt.input)
		if tt.expect_error {
			assert.Error(t, err, "%s: error unexpectedly nil", tt.testName)
		} else {
			assert.Nil(t, err, "%s: enrich error: %w", tt.testName, err)
			assert.NotNil(t, actual, "%s: returned nil event", tt.testName)
			//Validate output
			eq, msg := compareMapstr(tt.expected.Fields, actual.Fields)
			assert.True(t, eq, "%s: actual does not match expected: %s\n\nactual: \"%v\"\n\nexpected: \"%v\"", tt.testName, msg, actual, tt.expected)
		}
	}
}

// compareMapstr will compare that all fields in `a` have equal value to the fields in `b`.
// Note: Only fields that exist in `a` are compared; if `b` has additional fields, they do not affect the comparison
func compareMapstr(a mapstr.M, b mapstr.M) (equal bool, msg string) {
	equal = false
	msg = ""

	aFlat := a.Flatten()
	bFlat := b.Flatten()

	for _, key := range *a.FlattenKeys() {
		valA, err := aFlat.GetValue(key)
		if err == nil {
			// FlattenKeys returns inner and leaf nodes, we only need to consider leaf nodes
			// GetValue will return error when attempting to read inner nodes; these keys are ignored
			valB, err := bFlat.GetValue(key)
			if err != nil {
				msg = fmt.Sprintf("%s not found in mapstr b", key)
				return
			}
			if valA != valB {
				msg = fmt.Sprintf("mismatch in key %s: \"%v\" \"%v\"", key, valA, valB)
				return
			}
		}
	}
	equal = true
	return
}
