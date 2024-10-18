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

package systemlogs

import (
	"os"
	"testing"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/input/log"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func generateFile(t *testing.T) string {
	// Create a know file for testing, the content is not relevant
	// it just needs to exist
	knwonFile, err := os.CreateTemp(t.TempDir(), t.Name()+"knwonFile*")
	if err != nil {
		t.Fatalf("cannot create temporary file: %s", err)
	}

	if _, err := knwonFile.WriteString("Bowties are cool"); err != nil {
		t.Fatalf("cannot write to temporary file '%s': %s", knwonFile.Name(), err)
	}
	knwonFile.Close()

	return knwonFile.Name()
}

func TestUseJournald(t *testing.T) {
	filename := generateFile(t)

	testCases := map[string]struct {
		cfg         map[string]any
		useJournald bool
		expectErr   bool
	}{
		"No files found": {
			cfg: map[string]any{
				"files.paths": []string{"/file/does/not/exist"},
			},
			useJournald: true,
		},
		"File exists": {
			cfg: map[string]any{
				"files.paths": []string{filename},
			},
			useJournald: false,
		},
		"use_journald is true": {
			cfg: map[string]any{
				"use_journald": true,
				"journald":     struct{}{},
			},
			useJournald: true,
		},
		"use_files is true": {
			cfg: map[string]any{
				"use_files": true,
				"journald":  nil,
				"files":     struct{}{},
			},
			useJournald: false,
		},
		"use_journald and use_files are true": {
			cfg: map[string]any{
				"use_files":    true,
				"use_journald": true,
				"journald":     struct{}{},
			},
			useJournald: false,
			expectErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(tc.cfg)

			useJournald, err := useJournald(cfg)
			if !tc.expectErr && err != nil {
				t.Fatalf("did not expect an error calling 'useJournald': %s", err)
			}
			if tc.expectErr && err == nil {
				t.Fatal("expecting an error when calling 'userJournald', got none")
			}

			if useJournald != tc.useJournald {
				t.Fatalf("expecting 'useJournald' to be %t, got %t",
					tc.useJournald, useJournald)
			}
		})
	}
}

func TestLogInputIsInstantiated(t *testing.T) {
	filename := generateFile(t)
	c := map[string]any{
		"files.paths": []string{filename},
	}

	cfg := conf.MustNewConfigFrom(c)

	inp, err := newV1Input(cfg, connectorMock{}, input.Context{})
	if err != nil {
		t.Fatalf("did not expect an error calling newV1Input: %s", err)
	}
	_, isLogInput := inp.(*log.Input)
	if !isLogInput {
		t.Fatalf("expecting an instance of *log.Input, got '%T' instead", inp)
	}
}

type connectorMock struct{}

func (mock connectorMock) Connect(c *conf.C) (channel.Outleter, error) {
	return outleterMock{}, nil
}

func (mock connectorMock) ConnectWith(c *conf.C, clientConfig beat.ClientConfig) (channel.Outleter, error) {
	return outleterMock{}, nil
}

type outleterMock struct{}

func (o outleterMock) Close() error            { return nil }
func (o outleterMock) Done() <-chan struct{}   { return make(chan struct{}) }
func (o outleterMock) OnEvent(beat.Event) bool { return false }
