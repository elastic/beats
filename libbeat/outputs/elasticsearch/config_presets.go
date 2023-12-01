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

package elasticsearch

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
)

type preset string

const (
	presetNone       preset = ""
	presetCustom     preset = "custom"
	presetBalanced   preset = "balanced"
	presetThroughput preset = "throughput"
	presetScale      preset = "scale"
	presetLatency    preset = "latency"
)

var configMaps = map[preset]*config.C{
	presetNone:   config.MustNewConfigFrom(map[string]interface{}{}),
	presetCustom: config.MustNewConfigFrom(map[string]interface{}{}),
	presetBalanced: config.MustNewConfigFrom(map[string]interface{}{
		"bulk_max_size":              1600,
		"worker":                     1,
		"queue.mem.events":           3200,
		"queue.mem.flush.min_events": 1600,
		"queue.mem.flush.timeout":    10 * time.Second,
		"compression_level":          1,
		"idle_connection_timeout":    3 * time.Second,
	}),
	presetThroughput: config.MustNewConfigFrom(map[string]interface{}{
		"bulk_max_size":              1600,
		"worker":                     4,
		"queue.mem.events":           12800,
		"queue.mem.flush.min_events": 1600,
		"queue.mem.flush.timeout":    5 * time.Second,
		"compression_level":          1,
		"idle_connection_timeout":    15 * time.Second,
	}),
	presetScale: config.MustNewConfigFrom(map[string]interface{}{
		"bulk_max_size":              1600,
		"worker":                     1,
		"queue.mem.events":           3200,
		"queue.mem.flush.min_events": 1600,
		"queue.mem.flush.timeout":    20 * time.Second,
		"compression_level":          1,
		"idle_connection_timeout":    1 * time.Second,
	}),
	presetLatency: config.MustNewConfigFrom(map[string]interface{}{
		"bulk_max_size":              50,
		"worker":                     1,
		"queue.mem.events":           4100,
		"queue.mem.flush.min_events": 2050,
		"queue.mem.flush.timeout":    1 * time.Second,
		"compression_level":          1,
		"idle_connection_timeout":    60 * time.Second,
	}),
}

// Make sure unpacked preset names are valid
func (p *preset) Unpack(s string) error {
	value := preset(s)
	if err := value.Validate(); err != nil {
		return err
	}
	*p = value
	return nil
}

// Return an error if the preset name is unknown
func (p *preset) Validate() error {
	if configMaps[*p] == nil {
		return fmt.Errorf("unknown preset name '%v'", p)
	}
	return nil
}

// Apply the configuration for c's "preset" field, returning a list of fields
// in the provided user config that were overwritten.
func applyPreset(
	target *elasticsearchConfig, userConfig *config.C,
) ([]string, error) {
	presetConfig := configMaps[target.Preset]
	if presetConfig == nil {
		// This should never happen because the Preset field is validated
		// when it's first unpacked, but let's be cautious.
		return nil, fmt.Errorf("unknown preset value %v", target.Preset)
	}

	// Check for any user-provided fields that overlap with the preset.
	// Queue parameters have special handling since they must be applied
	// as a group so all queue parameters conflict with each other.
	userKeys := userConfig.FlattenedKeys()
	presetKeys := presetConfig.FlattenedKeys()
	presetConfiguresQueue := listContainsPrefix(presetKeys, "queue.")
	queueConflict := false
	overridden := []string{}
	for _, key := range userKeys {
		if strings.HasPrefix(key, "queue.") && presetConfiguresQueue {
			overridden = append(overridden, key)
			queueConflict = true
		} else if listContainsStr(presetKeys, key) {
			overridden = append(overridden, key)
		}
	}
	if queueConflict {
		// If both the preset and the user config have queue parameters,
		// we need to explicitly clear the elasticsearchConfig's queue
		// field before applying the preset, since config.Unpack gives
		// an error when unpacking namespace types twice even if the
		// names match.
		target.Queue = config.Namespace{}
	}

	// Apply the preset to the ES config
	err := presetConfig.Unpack(target)
	if err != nil {
		return nil, err
	}

	return overridden, nil
}

// TODO: Replace this with slices.Contains once we hit Go 1.21.
func listContainsStr(list []string, str string) bool {
	for _, s := range list {
		if s == str {
			return true
		}
	}
	return false
}

func listContainsPrefix(list []string, prefix string) bool {
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
