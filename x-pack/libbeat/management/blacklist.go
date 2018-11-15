// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

// ConfigBlacklist takes a ConfigBlocks object and filter it based on the given
// blacklist settings
type ConfigBlacklist struct {
	patterns map[string]*regexp.Regexp
}

// ConfigBlacklistSettings holds a list of fields and regular expressions to blacklist
type ConfigBlacklistSettings map[string]string

// Unpack unpacks nested fields set with dot notation like foo.bar into the proper nesting
// in a nested map/slice structure.
func (f ConfigBlacklistSettings) Unpack(to interface{}) error {
	m, ok := to.(map[string]interface{})
	if !ok {
		return fmt.Errorf("wrong type, expect map")
	}

	var expand func(key string, value interface{})

	expand = func(key string, value interface{}) {
		switch v := value.(type) {
		case map[string]interface{}:
			for k, val := range v {
				expand(fmt.Sprintf("%v.%v", key, k), val)
			}
		case []interface{}:
			for i := range v {
				expand(fmt.Sprintf("%v.%v", key, i), v[i])
			}
		default:
			m[key] = fmt.Sprintf("%s", value)
		}
	}

	for k, val := range m {
		expand(k, val)
	}
	return nil
}

// NewConfigBlacklist filters configs from CM according to a given blacklist
func NewConfigBlacklist(patterns ConfigBlacklistSettings) (*ConfigBlacklist, error) {
	list := ConfigBlacklist{
		patterns: map[string]*regexp.Regexp{},
	}

	for field, pattern := range patterns {
		exp, err := regexp.Compile(pattern)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Given expression is not a valid regexp: %s", pattern))
		}

		list.patterns[field] = exp
	}

	return &list, nil
}

// Filter returns a copy of the given ConfigBlocks with the
func (c *ConfigBlacklist) Filter(configBlocks api.ConfigBlocks) api.ConfigBlocks {
	var result api.ConfigBlocks

	for _, configs := range configBlocks {
		newConfigs := api.ConfigBlocksWithType{Type: configs.Type}

		for _, block := range configs.Blocks {
			if c.isBlacklisted(configs.Type, block) {
				logp.Err("Got a blacklisted configuration, ignoring it")
				continue
			}

			newConfigs.Blocks = append(newConfigs.Blocks, block)
		}

		if len(newConfigs.Blocks) > 0 {
			result = append(result, newConfigs)
		}
	}

	return result
}

func (c *ConfigBlacklist) isBlacklisted(blockType string, block *api.ConfigBlock) bool {
	cfg, err := block.ConfigWithMeta()
	if err != nil {
		return false
	}

	for field, pattern := range c.patterns {
		prefix := blockType
		if strings.Contains(field, ".") {
			prefix += "."
		}

		if strings.HasPrefix(field, prefix) {
			// This pattern affects a field on this block type
			field = field[len(prefix):]
			var segments []string
			if len(field) > 0 {
				segments = strings.Split(field, ".")
			}
			if c.isBlacklistedBlock(pattern, segments, cfg.Config) {
				return true
			}
		}
	}

	return false
}

func (c *ConfigBlacklist) isBlacklistedBlock(pattern *regexp.Regexp, segments []string, current *common.Config) bool {
	if current.IsDict() {
		switch len(segments) {
		case 0:
			for _, field := range current.GetFields() {
				if pattern.MatchString(field) {
					return true
				}
			}

		case 1:
			// Check field in the dict
			val, err := current.String(segments[0], -1)
			if err == nil {
				return pattern.MatchString(val)
			}
			// not a string, traverse
			child, _ := current.Child(segments[0], -1)
			return child != nil && c.isBlacklistedBlock(pattern, segments[1:], child)

		default:
			// traverse the tree
			child, _ := current.Child(segments[0], -1)
			return child != nil && c.isBlacklistedBlock(pattern, segments[1:], child)

		}
	}

	if current.IsArray() {
		switch len(segments) {
		case 0:
			// List of elements, match strings
			for count, _ := current.CountField(""); count > 0; count-- {
				val, err := current.String("", count-1)
				if err == nil && pattern.MatchString(val) {
					return true
				}

				// not a string, traverse
				child, _ := current.Child("", count-1)
				if child != nil {
					if c.isBlacklistedBlock(pattern, segments, child) {
						return true
					}
				}
			}

		default:
			// List of elements, explode traversal to all of them
			for count, _ := current.CountField(""); count > 0; count-- {
				child, _ := current.Child("", count-1)
				if child != nil && c.isBlacklistedBlock(pattern, segments, child) {
					return true
				}
			}
		}
	}

	return false
}
