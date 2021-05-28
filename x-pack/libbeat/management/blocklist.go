// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/x-pack/libbeat/management/api"
)

// ConfigBlocklist takes a ConfigBlocks object and filter it based on the given
// blocklist settings
type ConfigBlocklist struct {
	patterns map[string]match.Matcher
}

// ConfigBlocklistSettings holds a list of fields and regular expressions to blocklist
type ConfigBlocklistSettings struct {
	Patterns map[string]string `yaml:",inline"`
}

// Unpack unpacks nested fields set with dot notation like foo.bar into the proper nesting
// in a nested map/slice structure.
func (f *ConfigBlocklistSettings) Unpack(from interface{}) error {
	m, ok := from.(map[string]interface{})
	if !ok {
		return fmt.Errorf("wrong type, map is expected")
	}

	f.Patterns = map[string]string{}
	for k, v := range common.MapStr(m).Flatten() {
		f.Patterns[k] = fmt.Sprintf("%s", v)
	}

	return nil
}

// NewConfigBlocklist filters configs from CM according to a given blocklist
func NewConfigBlocklist(cfg ConfigBlocklistSettings) (*ConfigBlocklist, error) {
	list := ConfigBlocklist{
		patterns: map[string]match.Matcher{},
	}

	for field, pattern := range cfg.Patterns {
		exp, err := match.Compile(pattern)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("Given expression is not a valid regexp: %s", pattern))
		}

		list.patterns[field] = exp
	}

	return &list, nil
}

// Detect an error if any of the given config blocks is blocklisted
func (c *ConfigBlocklist) Detect(configBlocks api.ConfigBlocks) Errors {
	var errs Errors
	for _, configs := range configBlocks {
		for _, block := range configs.Blocks {
			if c.isBlocklisted(configs.Type, block) {
				errs = append(errs, &Error{
					Type: ConfigError,
					Err:  fmt.Errorf("Config for '%s' is blocklisted", configs.Type),
				})
			}
		}
	}
	return errs
}

func (c *ConfigBlocklist) isBlocklisted(blockType string, block *api.ConfigBlock) bool {
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
			if c.isBlocklistedBlock(pattern, segments, cfg.Config) {
				return true
			}
		}
	}

	return false
}

func (c *ConfigBlocklist) isBlocklistedBlock(pattern match.Matcher, segments []string, current *common.Config) bool {
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
			return child != nil && c.isBlocklistedBlock(pattern, segments[1:], child)

		default:
			// traverse the tree
			child, _ := current.Child(segments[0], -1)
			return child != nil && c.isBlocklistedBlock(pattern, segments[1:], child)

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
					if c.isBlocklistedBlock(pattern, segments, child) {
						return true
					}
				}
			}

		default:
			// List of elements, explode traversal to all of them
			for count, _ := current.CountField(""); count > 0; count-- {
				child, _ := current.Child("", count-1)
				if child != nil && c.isBlocklistedBlock(pattern, segments, child) {
					return true
				}
			}
		}
	}

	return false
}
