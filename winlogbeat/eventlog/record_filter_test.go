//go:build windows

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

package eventlog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
)

func TestParseLevels(t *testing.T) {
	levels, err := parseLevels("info, warning, 2, crit")
	require.NoError(t, err)

	assert.Contains(t, levels, uint8(0))
	assert.Contains(t, levels, uint8(4))
	assert.Contains(t, levels, uint8(3))
	assert.Contains(t, levels, uint8(2))
	assert.Contains(t, levels, uint8(1))
	assert.NotContains(t, levels, uint8(5))
}

func TestParseLevelsInvalid(t *testing.T) {
	_, err := parseLevels("warning, potato")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid level")
}

func TestParseEventIDRanges(t *testing.T) {
	includes, excludes, err := parseEventIDRanges("1, 100-200, -17, -300-303")
	require.NoError(t, err)

	assert.Equal(t, []eventIDRange{
		{start: 1, end: 1},
		{start: 100, end: 200},
	}, includes)
	assert.Equal(t, []eventIDRange{
		{start: 17, end: 17},
		{start: 300, end: 303},
	}, excludes)
}

func TestParseEventIDRangesInvalid(t *testing.T) {
	tests := []string{
		"foo",
		"7-3",
		",",
		"-",
	}

	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			_, _, err := parseEventIDRanges(in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid")
		})
	}
}

func TestRecordFilterMatch(t *testing.T) {
	f, err := newRecordFilter(query{
		IgnoreOlder: time.Hour,
		Provider:    []string{"MyProvider"},
		Level:       "warning",
		EventID:     "100, 200-210, -205",
	})
	require.NoError(t, err)

	assert.True(t, f.match(testRecord(time.Now().Add(-30*time.Minute), "MyProvider", "", 3, 100)))

	assert.False(t, f.match(testRecord(time.Now().Add(-2*time.Hour), "myprovider", "", 3, 100)))
	assert.False(t, f.match(testRecord(time.Now().Add(-30*time.Minute), "other", "", 3, 100)))
	assert.False(t, f.match(testRecord(time.Now().Add(-30*time.Minute), "myprovider", "", 3, 100)))
	assert.False(t, f.match(testRecord(time.Now().Add(-30*time.Minute), "", "MyProvider", 3, 201)))
	assert.False(t, f.match(testRecord(time.Now().Add(-30*time.Minute), "myprovider", "", 2, 100)))
	assert.False(t, f.match(testRecord(time.Now().Add(-30*time.Minute), "myprovider", "", 3, 300)))

	// Excludes take precedence over includes.
	assert.False(t, f.match(testRecord(time.Now().Add(-30*time.Minute), "myprovider", "", 3, 205)))
}

func TestRecordFilterMatchNil(t *testing.T) {
	var f *recordFilter
	assert.True(t, f.match(nil))
}

func TestRecordFilterIgnoreOlderZeroTimestamp(t *testing.T) {
	f, err := newRecordFilter(query{IgnoreOlder: time.Nanosecond})
	require.NoError(t, err)

	// Zero event timestamp should not be dropped by ignore_older.
	assert.True(t, f.match(testRecord(time.Time{}, "provider", "", 4, 1)))
}

func testRecord(ts time.Time, providerName, sourceName string, level uint8, id uint32) *Record {
	return &Record{
		Event: winevent.Event{
			Provider: winevent.Provider{
				Name:            providerName,
				EventSourceName: sourceName,
			},
			LevelRaw: level,
			EventIdentifier: winevent.EventIdentifier{
				ID: id,
			},
			TimeCreated: winevent.TimeCreated{
				SystemTime: ts,
			},
		},
	}
}
