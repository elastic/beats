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

package builder

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const logName = "autodiscover.builder"

// GetContainerID returns the id of a container
func GetContainerID(container mapstr.M) string {
	id, _ := container["id"].(string)
	return id
}

// GetContainerName returns the name of a container
func GetContainerName(container mapstr.M) string {
	name, _ := container["name"].(string)
	return name
}

// GetHintString takes a hint and returns its value as a string
func GetHintString(hints mapstr.M, key, config string) string {
	base := config
	if base == "" {
		base = key
	} else if key != "" {
		base = fmt.Sprint(key, ".", config)
	}
	if iface, err := hints.GetValue(base); err == nil {
		if str, ok := iface.(string); ok {
			return str
		}
	}

	return ""
}

// GetHintMapStr takes a hint and returns a MapStr
func GetHintMapStr(hints mapstr.M, key, config string) mapstr.M {
	base := config
	if base == "" {
		base = key
	} else if key != "" {
		base = fmt.Sprint(key, ".", config)
	}
	if iface, err := hints.GetValue(base); err == nil {
		if mapstr, ok := iface.(mapstr.M); ok {
			return mapstr
		}
	}

	return nil
}

// GetHintAsList takes a hint and returns the value as lists.
func GetHintAsList(hints mapstr.M, key, config string) []string {
	if str := GetHintString(hints, key, config); str != "" {
		return getStringAsList(str)
	}

	return nil
}

// GetProcessors gets processor definitions from the hints and returns a list of configs as a MapStr
func GetProcessors(hints mapstr.M, key string) []mapstr.M {
	processors := GetConfigs(hints, key, "processors")
	for _, proc := range processors {
		for key, value := range proc {
			if str, ok := value.(string); ok {
				cfg := mapstr.M{}
				if err := json.Unmarshal([]byte(str), &cfg); err != nil {
					logp.NewLogger(logName).Debugw("Unable to unmarshal json due to error", "error", err)
					continue
				}
				proc[key] = cfg
			}
		}
	}
	return processors
}

// GetConfigs takes in a key and returns a list of configs as a slice of MapStr
func GetConfigs(hints mapstr.M, key, name string) []mapstr.M {
	raw := GetHintMapStr(hints, key, name)
	if raw == nil {
		return nil
	}

	var words, nums []string

	for key := range raw {
		if _, err := strconv.Atoi(key); err != nil {
			words = append(words, key)
			continue
		} else {
			nums = append(nums, key)
		}
	}

	sort.Strings(nums)

	var configs []mapstr.M
	for _, key := range nums {
		rawCfg := raw[key]
		if config, ok := rawCfg.(mapstr.M); ok {
			configs = append(configs, config)
		}
	}

	for _, word := range words {
		configs = append(configs, mapstr.M{
			word: raw[word],
		})
	}

	return configs
}

func getStringAsList(input string) []string {
	if input == "" {
		return []string{}
	}
	list := strings.Split(input, ",")

	for i := 0; i < len(list); i++ {
		list[i] = strings.TrimSpace(list[i])
	}

	return list
}

// GetHintAsConfigs can read a hint in the form of a stringified JSON and return a mapstr.M
func GetHintAsConfigs(hints mapstr.M, key string) []mapstr.M {
	if str := GetHintString(hints, key, "raw"); str != "" {
		// check if it is a single config
		if str[0] != '[' {
			cfg := mapstr.M{}
			if err := json.Unmarshal([]byte(str), &cfg); err != nil {
				logp.NewLogger(logName).Debugw("Unable to unmarshal json due to error", "error", err)
				return nil
			}
			return []mapstr.M{cfg}
		}

		var cfg []mapstr.M
		if err := json.Unmarshal([]byte(str), &cfg); err != nil {
			logp.NewLogger(logName).Debugw("Unable to unmarshal json due to error", "error", err)
			return nil
		}
		return cfg
	}
	return nil
}

// IsEnabled will return true when 'enabled' is **explicitly** set to true.
func IsEnabled(hints mapstr.M, key string) bool {
	if value, err := hints.GetValue(fmt.Sprintf("%s.enabled", key)); err == nil {
		enabled, _ := strconv.ParseBool(value.(string))
		return enabled
	}

	return false
}

// IsDisabled will return true when 'enabled' is **explicitly** set to false.
func IsDisabled(hints mapstr.M, key string) bool {
	if value, err := hints.GetValue(fmt.Sprintf("%s.enabled", key)); err == nil {
		enabled, err := strconv.ParseBool(value.(string))
		if err != nil {
			logp.NewLogger(logName).Debugw("Error parsing 'enabled' hint.",
				"error", err, "autodiscover.hints", hints)
			return false
		}
		return !enabled
	}

	// keep reading disable (deprecated) for backwards compatibility
	if value, err := hints.GetValue(fmt.Sprintf("%s.disable", key)); err == nil {
		cfgwarn.Deprecate("8.0.0", "disable hint is deprecated. Use `enabled: false` instead.")
		disabled, _ := strconv.ParseBool(value.(string))
		return disabled
	}

	return false
}

// GenerateHints parses annotations based on a prefix and sets up hints that can be picked up by individual Beats.
func GenerateHints(annotations mapstr.M, container, prefix string) mapstr.M {
	hints := mapstr.M{}
	if rawEntries, err := annotations.GetValue(prefix); err == nil {
		if entries, ok := rawEntries.(mapstr.M); ok {
			for key, rawValue := range entries {
				// If there are top level hints like co.elastic.logs/ then just add the values after the /
				// Only consider namespaced annotations
				parts := strings.Split(key, "/")
				if len(parts) == 2 {
					hintKey := fmt.Sprintf("%s.%s", parts[0], parts[1])
					// Insert only if there is no entry already. container level annotations take
					// higher priority.
					if _, err := hints.GetValue(hintKey); err != nil {
						hints.Put(hintKey, rawValue)
					}
				} else if container != "" {
					// Only consider annotations that are of type mapstr.M as we are looking for
					// container level nesting
					builderHints, ok := rawValue.(mapstr.M)
					if !ok {
						continue
					}

					// Check for <containerName>/ prefix
					for hintKey, rawVal := range builderHints {
						if strings.HasPrefix(hintKey, container) {
							// Split the key to get part[1] to be the hint
							parts := strings.Split(hintKey, "/")
							if len(parts) == 2 {
								// key will be the hint type
								hintKey := fmt.Sprintf("%s.%s", key, parts[1])
								hints.Put(hintKey, rawVal)
							}
						}
					}
				}
			}
		}
	}

	return hints
}

// GetHintsAsList gets a set of hints and tries to convert them into a list of hints
func GetHintsAsList(hints mapstr.M, key string) []mapstr.M {
	raw := GetHintMapStr(hints, key, "")
	if raw == nil {
		return nil
	}

	var words, nums []string

	for key := range raw {
		if _, err := strconv.Atoi(key); err != nil {
			words = append(words, key)
			continue
		} else {
			nums = append(nums, key)
		}
	}

	sort.Strings(nums)

	var configs []mapstr.M
	for _, key := range nums {
		rawCfg := raw[key]
		if config, ok := rawCfg.(mapstr.M); ok {
			configs = append(configs, config)
		}
	}

	defaultMap := mapstr.M{}
	for _, word := range words {
		defaultMap[word] = raw[word]
	}

	if len(defaultMap) != 0 {
		configs = append(configs, defaultMap)
	}
	return configs
}
