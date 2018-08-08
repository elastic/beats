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

package dns

import (
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

// Config defines the configuration options for the DNS processor.
type Config struct {
	CacheConfig
	Nameservers []string        `config:"nameservers"`                // Required on Windows. /etc/resolv.conf is used if none are given.
	Timeout     time.Duration   `conifg:"timeout"`                    // Per request timeout (with 2 nameservers the total timeout would be 2x).
	Lookup      []*LookupConfig `config:"lookup" validate:"required"` // List lookups to perform.
}

// LookupConfig defines what type of lookup is performed on which fields.
type LookupConfig struct {
	Type        string        `config:"type" validate:"required"` // Reverse is the only supported type currently.
	Action      FieldAction   `config:"action"`                   // Append or replace (defaults to append) when target exists.
	Fields      common.MapStr `config:"fields"`                   // Mapping of source fields to target fields.
	reverseFlat map[string]string
}

// FieldAction defines the behavior when the target field exists.
type FieldAction uint8

// List of FieldAction types.
const (
	ActionAppend FieldAction = iota
	ActionReplace
)

// Unpack unpacks a string to a FieldAction.
func (fa *FieldAction) Unpack(v string) error {
	switch strings.ToLower(v) {
	case "", "append":
		*fa = ActionAppend
	case "replace":
		*fa = ActionReplace
	default:
		return errors.Errorf("invalid dns lookup action value '%v'", v)
	}
	return nil
}

// CacheConfig defines the success and failure caching parameters.
type CacheConfig struct {
	SuccessCache CacheSettings `config:"success_cache"`
	FailureCache CacheSettings `config:"failure_cache"`
}

// CacheSettings define the caching behavior for an individual cache.
type CacheSettings struct {
	// TTL value for items in cache. Not used for success because we use TTL
	// from the DNS record.
	TTL time.Duration `config:"ttl"`
	// Initial capacity. How much space is allocated at initialization.
	InitialCapacity int `config:"capacity.initial" validate:"min=0"`
	// Max capacity of the cache. When capacity is reached a random item is
	// evicted from the cache.
	MaxCapacity int `config:"capacity.max"     validate:"min=0"`
}

// Validate validates the data contained in the config.
func (c *Config) Validate() error {
	for _, l := range c.Lookup {
		l.Type = strings.ToLower(l.Type)
		switch l.Type {
		case "reverse", "ptr":
		default:
			return errors.Errorf("invalid dns lookup type '%v' specified in "+
				"config (valid values are reverse or ptr)", l.Type)
		}

		l.reverseFlat = map[string]string{}
		for k, v := range l.Fields.Flatten() {
			target, ok := v.(string)
			if !ok {
				return errors.Errorf("target field for dns lookup of %v "+
					"must be a string but got %T", k, v)
			}
			l.reverseFlat[k] = target
		}
	}

	if c.SuccessCache.MaxCapacity != 0 && c.SuccessCache.MaxCapacity < c.SuccessCache.InitialCapacity {
		return errors.Errorf("success_cache.capacity.max must be >= success_cache.capacity.initial")
	}
	if c.FailureCache.MaxCapacity != 0 && c.FailureCache.MaxCapacity < c.FailureCache.InitialCapacity {
		return errors.Errorf("failure_cache.capacity.max must be >= failure_cache.capacity.initial")
	}
	return nil
}

var defaultConfig = Config{
	CacheConfig: CacheConfig{
		SuccessCache: CacheSettings{
			InitialCapacity: 1000,
			MaxCapacity:     10000,
		},
		FailureCache: CacheSettings{
			TTL:             time.Minute,
			InitialCapacity: 1000,
			MaxCapacity:     10000,
		},
	},
	Timeout: 500 * time.Millisecond,
}
