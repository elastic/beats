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

package filestream

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
)

func TestResolvePreset(t *testing.T) {
	explicitAuto := func(threshold int) presetSelection {
		return presetSelection{name: presetAuto, explicit: true, threshold: threshold}
	}
	explicitHighCardinality := presetSelection{name: presetHighCardinality, explicit: true}
	explicitDefault := presetSelection{name: presetDefault, explicit: true}
	defaultValues := resolvedPreset{checkInterval: 10 * time.Second}
	highCardinalityValues := resolvedPreset{
		highCardinality: true,
		harvesterLimit:  highCardinalityHarvesterLimit,
		checkInterval:   highCardinalityCheckInterval,
	}

	cases := []struct {
		name      string
		in        presetInputs
		fileCount int
		want      resolvedPreset
	}{
		{
			name:      "auto with file count below the threshold runs default",
			in:        presetInputs{selection: explicitAuto(5), checkInterval: 10 * time.Second},
			fileCount: 4,
			want:      defaultValues,
		},
		{
			name:      "auto with file count equal to the threshold runs high_cardinality",
			in:        presetInputs{selection: explicitAuto(5), checkInterval: 10 * time.Second},
			fileCount: 5,
			want:      highCardinalityValues,
		},
		{
			name:      "auto with threshold 0 runs high_cardinality without files",
			in:        presetInputs{selection: explicitAuto(0), checkInterval: 10 * time.Second},
			fileCount: 0,
			want:      highCardinalityValues,
		},
		{
			name:      "high_cardinality runs high_cardinality without files",
			in:        presetInputs{selection: explicitHighCardinality, checkInterval: 10 * time.Second},
			fileCount: 0,
			want:      highCardinalityValues,
		},
		{
			name:      "default runs default far above the threshold",
			in:        presetInputs{selection: explicitDefault, checkInterval: 10 * time.Second},
			fileCount: 1_000_000,
			want:      defaultValues,
		},
		{
			name: "high_cardinality keeps a configured harvester_limit",
			in: presetInputs{
				selection:      explicitHighCardinality,
				harvesterLimit: 100,
				checkInterval:  10 * time.Second,
			},
			want: resolvedPreset{
				highCardinality: true,
				harvesterLimit:  100,
				checkInterval:   highCardinalityCheckInterval,
			},
		},
		{
			name: "high_cardinality keeps an explicit check_interval equal to the default",
			in: presetInputs{
				selection:        explicitHighCardinality,
				checkInterval:    10 * time.Second,
				checkIntervalSet: true,
			},
			want: resolvedPreset{
				highCardinality: true,
				harvesterLimit:  highCardinalityHarvesterLimit,
				checkInterval:   10 * time.Second,
			},
		},
		{
			name: "high_cardinality keeps an explicit check_interval",
			in: presetInputs{
				selection:        explicitHighCardinality,
				checkInterval:    5 * time.Minute,
				checkIntervalSet: true,
			},
			want: resolvedPreset{
				highCardinality: true,
				harvesterLimit:  highCardinalityHarvesterLimit,
				checkInterval:   5 * time.Minute,
			},
		},
		{
			name: "default keeps the configured values when they are set",
			in: presetInputs{
				selection:        explicitDefault,
				harvesterLimit:   100,
				checkInterval:    5 * time.Minute,
				checkIntervalSet: true,
			},
			want: resolvedPreset{harvesterLimit: 100, checkInterval: 5 * time.Minute},
		},
		{
			// Changes to high_cardinality when autoByDefault is flipped.
			name: "auto not set runs default above the threshold",
			in: presetInputs{
				selection:     presetSelection{name: presetAuto, threshold: 5},
				checkInterval: 10 * time.Second,
			},
			fileCount: 6,
			want:      defaultValues,
		},
		{
			// The copytruncate prospector with 'preset' not set. Once
			// autoByDefault is flipped only forceDefault keeps it on default.
			name: "forceDefault with auto not set runs default above the threshold",
			in: presetInputs{
				selection:     presetSelection{name: presetAuto, threshold: 5},
				forceDefault:  true,
				checkInterval: 10 * time.Second,
			},
			fileCount: 6,
			want:      defaultValues,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, resolvePreset(tc.in, tc.fileCount), "unexpected resolved preset")
		})
	}
}

func TestHighCardinalityAction(t *testing.T) {
	found := loginp.CursorInfo{Found: true}
	endedAtEOF := loginp.CursorInfo{Found: true, EndedAtEOF: true}

	cases := []struct {
		name string
		info loginp.CursorInfo
		cur  state
		size int64
		gzip bool
		want startAction
	}{
		{
			name: "no cursor",
			size: 10,
			want: startCloseAtEOF,
		},
		{
			name: "GZIP read to EOF",
			info: found,
			cur:  state{Offset: 100, EOF: true},
			size: 10,
			gzip: true,
			want: startSkip,
		},
		{
			// Kept open if it were not GZIP.
			name: "GZIP not read to EOF",
			info: endedAtEOF,
			cur:  state{Offset: 5},
			size: 10,
			gzip: true,
			want: startCloseAtEOF,
		},
		{
			name: "offset equal to size",
			info: endedAtEOF,
			cur:  state{Offset: 10},
			size: 10,
			want: startSkip,
		},
		{
			name: "offset above size",
			info: endedAtEOF,
			cur:  state{Offset: 20},
			size: 10,
			want: startCloseAtEOF,
		},
		{
			name: "offset below size, last harvester did not end at EOF",
			info: found,
			cur:  state{Offset: 5},
			size: 10,
			want: startCloseAtEOF,
		},
		{
			name: "offset below size, last harvester ended at EOF",
			info: endedAtEOF,
			cur:  state{Offset: 5},
			size: 10,
			want: startKeepOpen,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, highCardinalityAction(tc.info, tc.cur, tc.size, tc.gzip),
				"unexpected action for cursor %+v, %+v, size %d, gzip %t", tc.info, tc.cur, tc.size, tc.gzip)
		})
	}
}

func TestPresetMessage(t *testing.T) {
	const reason = "'preset' is not set and 'delete.enabled' cannot be used with 'preset.auto' or 'preset.high_cardinality'"
	const copytruncateNote = "'rotation.external.strategy.copytruncate' always runs the default preset"
	notSet := presetSelection{name: presetAuto, threshold: 5000}
	defaultValues := resolvedPreset{checkInterval: 10 * time.Second}
	highCardinalityValues := resolvedPreset{
		highCardinality: true,
		harvesterLimit:  500,
		checkInterval:   30 * time.Second,
	}

	cases := []struct {
		name        string
		in          presetInputs
		fileCount   int
		res         resolvedPreset
		contains    []string
		notContains []string
		notSetOnce  bool
	}{
		{
			name:      "not set",
			in:        presetInputs{selection: notSet},
			fileCount: 6000,
			res:       defaultValues,
			contains: []string{
				"preset resolved: default",
				"'preset' is not set",
				"6000 files found",
				"harvester limit: no limit",
				"check interval: 10s",
			},
			notContains: []string{"threshold", "selected by"},
		},
		{
			name: "explicit auto shows the threshold",
			in: presetInputs{
				selection: presetSelection{name: presetAuto, explicit: true, threshold: 3},
			},
			fileCount: 4,
			res:       highCardinalityValues,
			contains: []string{
				"preset resolved: high_cardinality",
				"selected by 'preset.auto'",
				"4 files found",
				"threshold: 3 files",
				"harvester limit: 500",
				"check interval: 30s",
			},
			notContains: []string{"not set"},
		},
		{
			name:        "explicit high_cardinality",
			in:          presetInputs{selection: presetSelection{name: presetHighCardinality, explicit: true}},
			res:         highCardinalityValues,
			contains:    []string{"preset resolved: high_cardinality", "selected by 'preset.high_cardinality'", "0 files found"},
			notContains: []string{"threshold"},
		},
		{
			name: "forced to default by an option",
			in: presetInputs{
				selection: presetSelection{name: presetDefault, threshold: 5000, forcedDefaultReason: reason},
			},
			fileCount:   3,
			res:         defaultValues,
			contains:    []string{"preset resolved: default", reason, "3 files found"},
			notContains: []string{"threshold", "copytruncate"},
			notSetOnce:  true,
		},
		{
			name:        "not set with copytruncate",
			in:          presetInputs{selection: notSet, forceDefault: true},
			fileCount:   3,
			res:         defaultValues,
			contains:    []string{"preset resolved: default", "'preset' is not set", copytruncateNote, "3 files found"},
			notContains: []string{"threshold"},
		},
		{
			name: "explicit default with copytruncate",
			in: presetInputs{
				selection:    presetSelection{name: presetDefault, explicit: true},
				forceDefault: true,
			},
			fileCount:   3,
			res:         defaultValues,
			contains:    []string{"preset resolved: default", "selected by 'preset.default'", "3 files found"},
			notContains: []string{"threshold", "copytruncate"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := presetMessage(tc.in, tc.fileCount, tc.res)
			for _, want := range tc.contains {
				assert.Contains(t, msg, want, "the preset message must contain %q", want)
			}
			for _, unwanted := range tc.notContains {
				assert.NotContains(t, msg, unwanted, "the preset message must not contain %q", unwanted)
			}
			if tc.notSetOnce {
				assert.Equal(t, 1, strings.Count(msg, "'preset' is not set"),
					"the preset message must say once that 'preset' is not set: %s", msg)
			}
		})
	}
}
