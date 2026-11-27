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

// Package checkpoint holds the event log state types and reads the state
// file written by Winlogbeat versions that kept their own registry.
package checkpoint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"go.yaml.in/yaml/v2"
)

// PersistedState represents the format of the data persisted to disk.
type PersistedState struct {
	UpdateTime time.Time       `yaml:"update_time"`
	States     []EventLogState `yaml:"event_logs"`
}

// EventLogState represents the state of an individual event log.
type EventLogState struct {
	Name         string    `yaml:"name"`
	RecordNumber uint64    `yaml:"record_number"`
	Timestamp    time.Time `yaml:"timestamp"`
	Bookmark     string    `yaml:"bookmark,omitempty"`
}

// Read loads the state file written by Winlogbeat versions that kept their
// own registry. A missing file returns an empty map and no error.
func Read(file string) (map[string]EventLogState, error) {
	states := make(map[string]EventLogState)
	contents, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return states, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read persisted state from %s: %w", file, err)
	}

	var ps PersistedState
	err = yaml.Unmarshal(contents, &ps)
	if err != nil {
		return nil, fmt.Errorf("failed to read persisted state from %s: %w", file, err)
	}
	for _, s := range ps.States {
		states[s.Name] = s
	}
	return states, nil
}
