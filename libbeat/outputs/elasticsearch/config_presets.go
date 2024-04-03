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

const (
	presetNone       = ""
	presetCustom     = "custom"
	presetBalanced   = "balanced"
	presetThroughput = "throughput"
	presetScale      = "scale"
	presetLatency    = "latency"
)

var presetConfigs = map[string]*config.C{
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

// Given a user config, check its preset field and apply any corresponding
// config overrides.
// Returns a list of the user fields that were overwritten, and the full
// preset config that was applied.
func applyPreset(preset string, userConfig *config.C) ([]string, *config.C, error) {
	presetConfig := presetConfigs[preset]
	if presetConfig == nil {
		return nil, nil, fmt.Errorf("unknown preset value %v", preset)
	}

	// Check for any user-provided fields that overlap with the preset.
	// Queue parameters have special handling since they must be applied
	// as a group so all queue parameters conflict with each other.
	// User config keys must be fetch from a special helper since they
	// unpredictably contain an implicit prefix, see flattenedKeysForConfig.
	presetKeys := presetConfig.FlattenedKeys()
	userKeys, err := flattenedKeysForConfig(userConfig)
	if err != nil {
		return nil, nil, err
	}
	presetConfiguresQueue := listContainsPrefix(presetKeys, "queue.")
	overridden := []string{}
	for _, key := range userKeys {
		if strings.HasPrefix(key, "queue.") && presetConfiguresQueue {
			overridden = append(overridden, key)
		} else if listContainsStr(presetKeys, key) {
			overridden = append(overridden, key)
		}
	}
	// Remove the queue parameters if needed, then merge the preset
	// config on top of the user config.
	if presetConfiguresQueue {
		_, _ = userConfig.Remove("queue", -1)
	}
	err = userConfig.Merge(presetConfig)
	if err != nil {
		return nil, nil, err
	}
	return overridden, presetConfig, nil
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

// Configs passed to the Elasticsearch output are often (but not always)
// created via a config.Namespace, where the top-level namespace
// "elasticsearch" has all the output's config fields underneath. This
// usually doesn't matter, since we're just given the config object for
// the top-level keys underneath "elasticsearch". However, when collecting
// the full flattened key list for a config via FlattenedKeys(), a config
// object that was created via config.Namespace still "remembers" and
// includes the top-level "elasticsearch" key even though it is omitted
// when otherwise modifying or unpacking keys.
// Since we need the actual effective flattened key within the elasticsearch
// namespace, and we don't want it to behave differently depending on how
// our config object was originally generated, we use this workaround:
// unpack the config into a bare map, then repack it into a config object.
// This way it "forgets" about the namespace metadata since it is not
// included when unpacking into map for.
// It would be nice to not have to do this (perhaps config.Namespace.Unpack
// should strip the extra prefix somehow when it extracts the config subtree?)
func flattenedKeysForConfig(cfg *config.C) ([]string, error) {
	rawMap := make(map[string]interface{})
	err := cfg.Unpack(rawMap)
	if err != nil {
		return nil, err
	}
	strippedCfg, err := config.NewConfigFrom(rawMap)
	if err != nil {
		return nil, err
	}
	return strippedCfg.FlattenedKeys(), nil
}
