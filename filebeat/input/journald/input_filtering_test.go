// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux && cgo && withjournald
// +build linux,cgo,withjournald

package journald

import (
	"context"
	"path"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestInputSyslogIdentifier(t *testing.T) {
	tests := map[string]struct {
		identifiers      []string
		expectedMessages []string
	}{
		"one identifier": {
			identifiers: []string{"sudo"},
			expectedMessages: []string{
				"pam_unix(sudo:session): session closed for user root",
			},
		},
		"two identifiers": {
			identifiers: []string{"sudo", "systemd"},
			expectedMessages: []string{
				"pam_unix(sudo:session): session closed for user root",
				"Started Outputs some log lines.",
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			inp := env.mustCreateInput(mapstr.M{
				"paths":              []string{path.Join("testdata", "input-multiline-parser.journal")},
				"syslog_identifiers": testCase.identifiers,
			})

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)
			defer cancelInput()

			env.waitUntilEventCount(len(testCase.expectedMessages))

			for idx, event := range env.pipeline.clients[0].GetEvents() {
				if got, expected := event.Fields["message"], testCase.expectedMessages[idx]; got != expected {
					t.Fatalf("expecting event message %q, got %q", expected, got)
				}
			}
		})
	}
}

func TestInputUnits(t *testing.T) {
	tests := map[string]struct {
		units            []string
		kernel           bool
		expectedMessages []string
	}{
		"one unit": {
			units:  []string{"session-1.scope"},
			kernel: true,
			expectedMessages: []string{
				"pam_unix(sudo:session): session closed for user root",
			},
		},
		"one unit with kernel": {
			units: []string{"session-1.scope"},
			expectedMessages: []string{
				"pam_unix(sudo:session): session closed for user root",
			},
		},
		"two units, all messages": {
			units: []string{"session-1.scope", "user@1000.service"},
			expectedMessages: []string{
				"pam_unix(sudo:session): session closed for user root",
				"Started Outputs some log lines.",
				"1st line",
				"2nd line",
				"3rd line",
				"4th line",
				"5th line",
				"6th line",
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			inp := env.mustCreateInput(mapstr.M{
				"paths":  []string{path.Join("testdata", "input-multiline-parser.journal")},
				"units":  testCase.units,
				"kernel": testCase.kernel,
			})

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)
			defer cancelInput()

			env.waitUntilEventCount(len(testCase.expectedMessages))

			for idx, event := range env.pipeline.clients[0].GetEvents() {
				if got, expected := event.Fields["message"], testCase.expectedMessages[idx]; got != expected {
					t.Fatalf("expecting event message %q, got %q", expected, got)
				}
			}
		})
	}
}

func TestInputIncludeMatches(t *testing.T) {
	tests := map[string]struct {
		includeMatches   map[string]interface{}
		expectedMessages []string
	}{
		"single match condition": {
			includeMatches: map[string]interface{}{
				"match": []string{
					"syslog.facility=3",
				},
			},
			expectedMessages: []string{
				"Started Outputs some log lines.",
				"1st line",
				"2nd line",
				"3rd line",
				"4th line",
				"5th line",
				"6th line",
			},
		},
		"multiple match condition": {
			includeMatches: map[string]interface{}{
				"match": []string{
					"journald.process.name=systemd",
					"syslog.facility=3",
				},
			},
			expectedMessages: []string{
				"Started Outputs some log lines.",
			},
		},
		"and condition": {
			includeMatches: map[string]interface{}{
				"and": []map[string]interface{}{
					{
						"match": []string{
							"syslog.facility=3",
							"message=6th line",
						},
					},
				},
			},
			expectedMessages: []string{
				"6th line",
			},
		},
		"or condition": {
			includeMatches: map[string]interface{}{
				"or": []map[string]interface{}{
					{
						"match": []string{
							"message=5th line",
							"message=6th line",
						},
					},
				},
			},
			expectedMessages: []string{
				"5th line",
				"6th line",
			},
		},
	}

	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)
			inp := env.mustCreateInput(mapstr.M{
				"paths":           []string{path.Join("testdata", "input-multiline-parser.journal")},
				"include_matches": testCase.includeMatches,
			})

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)
			defer cancelInput()

			env.waitUntilEventCount(len(testCase.expectedMessages))

			for idx, event := range env.pipeline.clients[0].GetEvents() {
				if got, expected := event.Fields["message"], testCase.expectedMessages[idx]; got != expected {
					t.Fatalf("expecting event message %q, got %q", expected, got)
				}
			}
		})
	}
}
