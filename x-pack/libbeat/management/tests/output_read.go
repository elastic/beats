// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tests

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type findFieldsMode string

const ALL findFieldsMode = "all"
const ONCE findFieldsMode = "once"

// ReadEvents reads the ndjson output we get from the beats file output
func ReadEvents(t *testing.T, path string) []mapstr.M {
	files, err := filepath.Glob(filepath.Join(path, "*.ndjson"))
	require.NoError(t, err)

	events := []mapstr.M{}
	for _, file := range files {
		rawFile, err := os.ReadFile(file)
		require.NoError(t, err)
		lines := strings.Split(string(rawFile), "\n")
		for _, line := range lines {
			var event = mapstr.M{}
			// skip newlines that appear at the end of files
			if len(line) < 2 {
				continue
			}
			err = json.Unmarshal([]byte(line), &event)
			require.NoError(t, err)
			events = append(events, event)
		}
	}
	return events
}

// ValuesExist verifies that the given fields exist in the events.
// the values map takes keys in the form of keys in the events map, which may be in dot form: "system.cpu.cores", etc.
// The value for the map should be the expected value, or a `nil` if you merely want to check for the presence of a field.
// the mode determines if `ValuesExist` must exist in all events, or just one.
func ValuesExist(t *testing.T, values map[string]interface{}, events []mapstr.M, mode findFieldsMode) {
	for searchKey, val := range values {
		var foundCount = 0
		for eventIter, event := range events {
			evt, err := event.GetValue(searchKey)
			if errors.Is(err, mapstr.ErrKeyNotFound) {
				continue
			}
			if val == nil {
				foundCount++
			} else {
				if val == evt {
					foundCount++
				} else if val != evt && mode == ALL {
					t.Errorf("Key %s was found in event %d, but value was unexpected. Expected %#v, got %#v", searchKey, eventIter, val, evt)
				}
			}
		}
		if mode == ALL {
			if foundCount != len(events) {
				t.Errorf("Expected to find key %s in all %d events, but key was only found %d times.", searchKey, len(events), foundCount)
			}
		}
		if mode == ONCE {
			if foundCount == 0 {
				t.Errorf("Did not find key %s in any events", searchKey)
			}
		}
	}
}

// ValuesDoNotExist checks to make sure that the given keys do not exist in any events.
func ValuesDoNotExist(t *testing.T, values []string, events []mapstr.M) {
	for _, key := range values {
		for eventIter, event := range events {
			evt, _ := event.GetValue(key)
			if evt != nil {
				t.Errorf("key %s with value %#v was found in event %d in the output", key, evt, eventIter)
			}
		}
	}
}
