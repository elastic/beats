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
	"fmt"
	"strconv"
	"strings"
	"time"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
)

const (
	// highCardinalityHarvesterLimit is the harvester limit of the
	// high_cardinality preset when 'harvester_limit' is not set.
	highCardinalityHarvesterLimit = 500
	// highCardinalityCheckInterval is the scan interval of the
	// high_cardinality preset when 'prospector.scanner.check_interval' is
	// not set.
	highCardinalityCheckInterval = 30 * time.Second
)

// resolvedPreset is the preset an input runs and the values it implies.
// It is written once in fileProspector.Init, before Run and before any
// harvester exists, and is read-only afterwards, so it needs no lock.
type resolvedPreset struct {
	highCardinality bool
	// harvesterLimit is the maximum number of open harvesters, 0 means no
	// limit.
	harvesterLimit uint64
	checkInterval  time.Duration
}

// name returns the name of the resolved preset.
func (r resolvedPreset) name() string {
	if r.highCardinality {
		return presetHighCardinality
	}
	return presetDefault
}

// presetInputs is what the configuration contributes to the preset
// resolution.
type presetInputs struct {
	selection presetSelection
	// forceDefault is set for the copytruncate prospector, which always
	// runs the default preset, whatever the configuration selects.
	forceDefault bool
	// harvesterLimit is the configured 'harvester_limit', 0 means not set.
	harvesterLimit uint64
	// checkInterval is the configured 'prospector.scanner.check_interval'
	// or its default.
	checkInterval time.Duration
	// checkIntervalSet is true when the user wrote
	// 'prospector.scanner.check_interval', even with the default value.
	checkIntervalSet bool
	// closeOnEOF is the configured 'close.reader.on_eof', which the
	// high_cardinality preset ignores.
	closeOnEOF bool
}

// fileCountDecides reports whether the number of files found at input start
// selects the preset, which only happens for the auto preset, set or not.
func (in presetInputs) fileCountDecides() bool {
	return !in.forceDefault && in.selection.name == presetAuto
}

// resolvePreset returns the preset an input runs given its configuration and
// the number of files found at input start. default keeps the configured
// values; high_cardinality keeps the configured ones and fills in its own
// for those that are not set.
func resolvePreset(in presetInputs, fileCount int) resolvedPreset {
	var highCardinality bool
	if in.fileCountDecides() {
		highCardinality = fileCount >= in.selection.threshold
	} else {
		highCardinality = !in.forceDefault && in.selection.name == presetHighCardinality
	}

	if !highCardinality {
		return resolvedPreset{
			harvesterLimit: in.harvesterLimit,
			checkInterval:  in.checkInterval,
		}
	}

	res := resolvedPreset{
		highCardinality: true,
		harvesterLimit:  highCardinalityHarvesterLimit,
		checkInterval:   highCardinalityCheckInterval,
	}
	if in.harvesterLimit > 0 {
		res.harvesterLimit = in.harvesterLimit
	}
	if in.checkIntervalSet {
		res.checkInterval = in.checkInterval
	}

	return res
}

// startAction is what the high_cardinality preset does with a file the
// prospector would start a harvester for.
type startAction uint8

const (
	// startSkip does not start a harvester.
	startSkip startAction = iota
	// startCloseAtEOF starts a harvester that closes the file at EOF.
	startCloseAtEOF
	// startKeepOpen starts a harvester that keeps the file open after EOF.
	startKeepOpen
)

// highCardinalityAction decides from the registry cursor of a file what the
// high_cardinality preset does with it, so that idle files hold no harvester.
// A fully read file is not started. A file is only kept open when it grew
// after its last harvester read it to EOF, which means it is being written
// to; any other file is read to EOF and closed. GZIP offsets are on the
// decompressed stream, so the cursor's EOF flag replaces the size comparison,
// and a GZIP file is always closed at EOF by the reader. A cursor past the
// file size means the file was truncated, which openFile handles.
func highCardinalityAction(info loginp.CursorInfo, cur state, size int64, gzip bool) startAction {
	switch {
	case !info.Found:
		return startCloseAtEOF
	case gzip && cur.EOF:
		return startSkip
	case gzip:
		return startCloseAtEOF
	case cur.Offset == size:
		return startSkip
	case cur.Offset < size && info.EndedAtEOF:
		return startKeepOpen
	default:
		return startCloseAtEOF
	}
}

// presetMessage describes how the preset of an input was resolved, for the
// log line written at input start.
func presetMessage(in presetInputs, fileCount int, res resolvedPreset) string {
	parts := []string{"preset resolved: " + res.name()}
	switch {
	case in.selection.explicit:
		parts = append(parts, fmt.Sprintf("selected by 'preset.%s'", in.selection.name))
	case in.selection.forcedDefaultReason != "":
		parts = append(parts, in.selection.forcedDefaultReason)
	default:
		parts = append(parts, "'preset' is not set")
	}
	if in.forceDefault && in.selection.name != presetDefault {
		parts = append(parts, "'rotation.external.strategy.copytruncate' always runs the default preset")
	}
	parts = append(parts, fmt.Sprintf("%d files found", fileCount))
	if in.fileCountDecides() {
		parts = append(parts, fmt.Sprintf("threshold: %d files", in.selection.threshold))
	}

	limit := "no limit"
	if res.harvesterLimit > 0 {
		limit = strconv.FormatUint(res.harvesterLimit, 10)
	}
	parts = append(parts,
		"harvester limit: "+limit,
		"check interval: "+res.checkInterval.String())

	return strings.Join(parts, "; ")
}
