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

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/testutil"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestLoggerPlugin_New(t *testing.T) {
	validLogger := logp.NewLogger("logger_test")

	tests := []struct {
		name             string
		log              *logp.Logger
		logQueryResultFn HandleQueryResultFunc
		shouldPanic      bool
	}{
		{
			name:             "invalid",
			log:              nil,
			logQueryResultFn: nil,
			shouldPanic:      true,
		},
		{
			name:             "nologfunc",
			log:              validLogger,
			logQueryResultFn: nil,
		},
		{
			name:             "nonempty",
			log:              validLogger,
			logQueryResultFn: func(res QueryResult) {},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				testutil.AssertPanic(t, func() { NewLoggerPlugin(tc.log, tc.logQueryResultFn) })
				return
			}

			p := NewLoggerPlugin(tc.log, tc.logQueryResultFn)
			if p == nil {
				t.Error("expected nil logger pluggin")
			}
		})
	}
}

func TestLoggerPlugin_Log(t *testing.T) {
	validLogger := logp.NewLogger("logger_test")

	queryResultFn := func(res QueryResult) {
	}

	result := QueryResult{
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
		name             string
		logQueryResultFn HandleQueryResultFunc
		logType          logger.LogType
		logMessage       string
		err              string
	}{
		{
			name:             "nosnapshot",
			logQueryResultFn: queryResultFn,
			logType:          logger.LogTypeString,
			logMessage:       "{}",
		},
		{
			name:             "snapshot invalid",
			logQueryResultFn: queryResultFn,
			logType:          logger.LogTypeSnapshot,
			logMessage:       "",
			err:              "unexpected end of JSON input",
		},
		{
			name:             "snapshot empty",
			logQueryResultFn: queryResultFn,
			logType:          logger.LogTypeSnapshot,
			logMessage:       "{}",
		},
		{
			name:             "snapshot nonempty",
			logQueryResultFn: queryResultFn,
			logType:          logger.LogTypeSnapshot,
			logMessage:       string(resultbytes),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var capturedQueryResult *QueryResult
			p := NewLoggerPlugin(validLogger, func(res QueryResult) {
				capturedQueryResult = &res
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
					diff := cmp.Diff(capturedQueryResult, &result)
					if diff != "" {
						t.Error(diff)
					}
				}
			}
		})
	}
}
