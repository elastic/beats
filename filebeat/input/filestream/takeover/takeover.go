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

package takeover

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/filebeat/backup"
	cfg "github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/filebeat/input/filestream"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	loginputPrefix = "filebeat::logs::"
)

type filestreamMatchers map[string]func(source string) bool

// TakeOverLogInputStates performs the "take over" action for all filestream inputs
// that have `take_over: true` configuration parameter set to `true`.
//
// `take over` means every state that belongs to a loginput will be converted to a filestream state
// if the source file path matches one of the paths/globs of the filestream input.
//
// This mode is created for a smooth loginput->filestream migration experience, so the filestream
// inputs would pick up ingesting files from the same point where a loginput stopped.
func TakeOverLogInputStates(log *logp.Logger, store backend.Store, backuper backup.Backuper, cfg *cfg.Config) error {
	filestreamMatchers, err := findFilestreams(log, cfg)
	if err != nil {
		return fmt.Errorf("failed to read input configuration: %w", err)
	}
	if len(filestreamMatchers) == 0 {
		return nil
	}

	statesToSet, statesToRemove, err := takeOverStates(log, store, filestreamMatchers)
	if err != nil {
		return fmt.Errorf("failed to take over one of loginput states: %w", err)
	}
	if len(statesToSet) == 0 {
		log.Info("no state to take over")
		return nil
	}

	// before making changes, we backup the registry files for manual rollback if needed
	err = backuper.Backup()
	if err != nil {
		return fmt.Errorf("failed to create backup files: %w", err)
	}

	for key := range statesToSet {
		err = store.Set(key, statesToSet[key])
		if err != nil {
			return fmt.Errorf("failed to set the taken state: %w", err)
		}
	}
	for key := range statesToRemove {
		err = store.Remove(key)
		if err != nil {
			return fmt.Errorf("failed to remove the taken state: %w", err)
		}
	}

	log.Infof("filestream inputs took over %d file(s) from loginputs", len(statesToSet))
	return nil
}

func takeOverStates(log *logp.Logger, store backend.Store, matchers filestreamMatchers) (toSet map[string]mapstr.M, toRemove map[string]struct{}, err error) {
	toSet = make(map[string]mapstr.M)
	toRemove = make(map[string]struct{})

	err = store.Each(func(key string, value statestore.ValueDecoder) (bool, error) {
		if !strings.HasPrefix(key, loginputPrefix) {
			return true, nil
		}
		state := make(mapstr.M)
		err := value.Decode(&state)
		if err != nil {
			return false, err
		}
		sourceValue, err := state.GetValue("source")
		if errors.Is(err, mapstr.ErrKeyNotFound) {
			return true, nil
		}
		if err != nil {
			return false, fmt.Errorf("cannot extract source from the loginput state: %w", err)
		}
		source, ok := sourceValue.(string)
		if !ok {
			return true, nil
		}

		var filestreamID string
		for id, matcher := range matchers {
			if matcher(source) {
				filestreamID = id
				break
			}
		}
		if filestreamID == "" {
			return true, nil
		}

		newKey := loginputToFilestreamKey(key, filestreamID)
		log.Infof("found loginput state `%s` to take over by `%s`", key, newKey)

		newState := loginputToFilestream(state)

		toSet[newKey] = newState
		toRemove[key] = struct{}{}

		return true, nil
	})

	return toSet, toRemove, err
}

// findFilestreams finds filestream inputs that are marked as `take_over: true`
// and creates a file matcher for each such filestream for the future use in state
// processing
func findFilestreams(log *logp.Logger, cfg *cfg.Config) (matchers filestreamMatchers, err error) {
	matchers = make(filestreamMatchers)

	for _, input := range cfg.Inputs {
		inputCfg := defaultInputConfig()
		err := input.Unpack(&inputCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack input configuration: %w", err)
		}
		if inputCfg.Type != "filestream" || !inputCfg.TakeOver {
			continue
		}
		if _, exists := matchers[inputCfg.ID]; exists || inputCfg.ID == "" {
			return matchers, fmt.Errorf("filestream with ID `%s` in `take over` mode requires a unique ID. Add the `id:` key with a unique value.", inputCfg.ID)
		}

		matchers[inputCfg.ID], err = createMatcher(log, inputCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create filestream matcher: %w", err)
		}
	}

	if len(matchers) > 0 {
		log.Infof("found %d filestream inputs in `take over` mode", len(matchers))
	}

	return matchers, nil
}

// createMatcher creates a match function that determines whether the given
// source file matches one of the glob expressions listed in the filestream configuration
func createMatcher(log *logp.Logger, cfg inputConfig) (matcher func(source string) bool, err error) {
	patterns := cfg.Paths

	// see `../fswatch.go` for the similar logic
	if cfg.Prospector.Scanner.RecursiveGlob {
		log.Debugf("recursive glob enabled for filestream `%s`", cfg.ID)
		var newPatterns []string
		for _, pattern := range patterns {
			patterns, err := file.GlobPatterns(pattern, filestream.RecursiveGlobDepth)
			if err != nil {
				return nil, fmt.Errorf("failed to expand recursive globs: %w", err)
			}
			newPatterns = append(newPatterns, patterns...)
		}
		patterns = newPatterns
	} else {
		log.Debugf("recursive glob disabled for filestream `%s`", cfg.ID)
	}

	log.Infof("found %d patterns for filestream `%s`", len(patterns), cfg.ID)

	return func(source string) bool {
		for _, pattern := range patterns {
			matched, err := filepath.Match(pattern, source)
			// the only possible error is ErrBadPattern,
			// should be caught by config validation beforehand
			if err != nil {
				return false
			}
			if matched {
				return true
			}
		}

		return false
	}, nil
}

func loginputToFilestreamKey(key, filestreamID string) string {
	return strings.ReplaceAll(key, loginputPrefix, fmt.Sprintf("filestream::%s::", filestreamID))
}

// conversion from the log input type to the filestream input type
func loginputToFilestream(value mapstr.M) mapstr.M {
	newValue := make(mapstr.M)
	copyMapValue(value, newValue, "ttl", "ttl")
	copyMapValue(value, newValue, "timestamp", "updated")
	copyMapValue(value, newValue, "offset", "cursor.offset")
	copyMapValue(value, newValue, "source", "meta.source")
	copyMapValue(value, newValue, "identifier_name", "meta.identifier_name")
	return newValue
}

func copyMapValue(src, dst mapstr.M, srcKey, dstKey string) {
	value, err := src.GetValue(srcKey)
	if err != nil {
		return
	}
	_, err = dst.Put(dstKey, value)
	if err != nil {
		panic(err)
	}
}
