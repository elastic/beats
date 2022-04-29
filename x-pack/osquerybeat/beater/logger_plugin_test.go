// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/osquery/osquery-go/plugin/logger"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/testutil"
)

func TestLoggerPlugin_New(t *testing.T) {
	validLogger := logp.NewLogger("logger_test")

	tests := []struct {
		name          string
		log           *logp.Logger
		logSnapshotFn HandleSnapshotResultFunc
		shouldPanic   bool
	}{
		{
			name:          "invalid",
			log:           nil,
			logSnapshotFn: nil,
			shouldPanic:   true,
		},
		{
			name:          "nologfunc",
			log:           validLogger,
			logSnapshotFn: nil,
		},
		{
			name:          "nonempty",
			log:           validLogger,
			logSnapshotFn: func(res SnapshotResult) {},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				testutil.AssertPanic(t, func() { NewLoggerPlugin(tc.log, tc.logSnapshotFn) })
				return
			}

			p := NewLoggerPlugin(tc.log, tc.logSnapshotFn)
			if p == nil {
				t.Error("expected nil logger pluggin")
			}
		})
	}
}

func TestLoggerPlugin_Log(t *testing.T) {
	validLogger := logp.NewLogger("logger_test")

	snapshotFn := func(res SnapshotResult) {
	}

	result := SnapshotResult{
		Action: "foo",
		Name:   "bar",
		Hits: []map[string]string{
			{
				"testkey": "testval",
			},
			{
				"testkey2": "testval2",
				"testkey3": "testval3",
			},
		},
	}
	resultbytes, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		logSnapshotFn HandleSnapshotResultFunc
		logType       logger.LogType
		logMessage    string
		err           string
	}{
		{
			name:          "nosnapshot",
			logSnapshotFn: snapshotFn,
			logType:       logger.LogTypeString,
			logMessage:    "",
		},
		{
			name:          "snapshot invalid",
			logSnapshotFn: snapshotFn,
			logType:       logger.LogTypeSnapshot,
			logMessage:    "",
			err:           "unexpected end of JSON input",
		},
		{
			name:          "snapshot empty",
			logSnapshotFn: snapshotFn,
			logType:       logger.LogTypeSnapshot,
			logMessage:    "{}",
		},
		{
			name:          "snapshot nonempty",
			logSnapshotFn: snapshotFn,
			logType:       logger.LogTypeSnapshot,
			logMessage:    string(resultbytes),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedSnapshot *SnapshotResult
			p := NewLoggerPlugin(validLogger, func(res SnapshotResult) {
				capturedSnapshot = &res
			})
			err := p.Log(context.Background(), tc.logType, tc.logMessage)
			if err != nil {
				if tc.err == "" {
					t.Errorf("unexpected error: %v", err)
				} else {
					diff := cmp.Diff(err.Error(), tc.err)
					if diff != "" {
						t.Error(diff)
					}
				}
			} else {
				if tc.err != "" {
					t.Errorf("expected error: %v", tc.err)
				}
				if tc.logType == logger.LogTypeSnapshot && tc.logMessage == string(resultbytes) {
					diff := cmp.Diff(capturedSnapshot, &result)
					if diff != "" {
						t.Error(diff)
					}
				}
			}
		})
	}
}
