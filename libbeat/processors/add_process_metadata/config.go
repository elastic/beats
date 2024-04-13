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

package add_process_metadata

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// defaultCgroupRegex captures 64-character lowercase hexadecimal container IDs found in cgroup paths.
var defaultCgroupRegex = regexp.MustCompile(`[-/]([0-9a-f]{64})(\.scope)?$`)

type config struct {
	// IgnoreMissing: Ignore errors if event has no PID field.
	IgnoreMissing bool `config:"ignore_missing"`

	// OverwriteKeys allow target_fields to overwrite existing fields.
	OverwriteKeys bool `config:"overwrite_keys"`

	// RestrictedFields make restricted fields available (i.e. env).
	RestrictedFields bool `config:"restricted_fields"`

	// MatchPIDs fields containing the PID to lookup.
	MatchPIDs []string `config:"match_pids" validate:"required"`

	// Target is the destination root where fields will be added.
	Target string `config:"target"`

	// Fields is the list of fields to add to target.
	Fields []string `config:"include_fields"`

	// HostPath is the path where /proc reside
	HostPath string `config:"host_path"`

	// CgroupPrefix is the prefix where the container id is inside cgroup
	CgroupPrefixes []string `config:"cgroup_prefixes"`

	// CgroupRegex is the regular expression that captures the container ID from a cgroup path.
	CgroupRegex *regexp.Regexp `config:"cgroup_regex"`

	// CgroupCacheExpireTime is the length of time before cgroup cache elements expire in seconds,
	// set to 0 to disable the cgroup cache
	CgroupCacheExpireTime time.Duration `config:"cgroup_cache_expire_time"`
}

func (c *config) Validate() error {
	if c.CgroupRegex != nil && c.CgroupRegex.NumSubexp() != 1 {
		return fmt.Errorf("cgroup_regexp must contain exactly one capturing group for the container ID")
	}
	return nil
}

// available fields by default
var defaultFields = mapstr.M{
	"process": mapstr.M{
		"name":       nil,
		"title":      nil,
		"executable": nil,
		"args":       nil,
		"pid":        nil,
		"parent": mapstr.M{
			"pid": nil,
		},
		"entity_id":  nil,
		"start_time": nil,
		"owner": mapstr.M{
			"name": nil,
			"id":   nil,
		},
<<<<<<< HEAD
=======
		"group": mapstr.M{
			"name": nil,
			"id":   nil,
		},
		"thread": mapstr.M{
			"capabilities": mapstr.M{
				"effective": nil,
				"permitted": nil,
			},
		},
>>>>>>> ca4adcecac ([Auditbeat] fim(kprobes): enrich file events by coupling add_process_metadata processor (#38776))
	},
	"container": mapstr.M{
		"id": nil,
	},
}

// fields declared in here will only appear when requested explicitly
// with `restricted_fields: true`.
var restrictedFields = mapstr.M{
	"process": mapstr.M{
		"env": nil,
	},
}

func init() {
	restrictedFields.DeepUpdate(defaultFields)
}

func defaultConfig() config {
	return config{
		IgnoreMissing:         true,
		OverwriteKeys:         false,
		RestrictedFields:      false,
		MatchPIDs:             []string{"process.pid", "process.parent.pid"},
		HostPath:              "/",
		CgroupCacheExpireTime: cacheExpiration,
	}
}

type ConfigOption func(c *config)

func ConfigOverwriteKeys(overwriteKeys bool) ConfigOption {
	return func(c *config) {
		c.OverwriteKeys = overwriteKeys
	}
}

func ConfigMatchPIDs(matchPIDs []string) ConfigOption {
	return func(c *config) {
		c.MatchPIDs = matchPIDs
	}
}

func (c *config) getMappings() (mappings mapstr.M, err error) {
	mappings = mapstr.M{}
	validFields := defaultFields
	if c.RestrictedFields {
		validFields = restrictedFields
	}
	fieldPrefix := c.Target
	if len(fieldPrefix) > 0 {
		fieldPrefix += "."
	}
	wantedFields := c.Fields
	if len(wantedFields) == 0 {
		wantedFields = []string{"process", "container"}
	}
	for _, docSrc := range wantedFields {
		dstField := constructPath(fieldPrefix, docSrc)
		reqField, err := validFields.GetValue(docSrc)
		if err != nil {
			return nil, fmt.Errorf("field '%v' not found", docSrc)
		}
		if reqField != nil {
			for subField := range reqField.(mapstr.M) {
				key := dstField + "." + subField
				val := docSrc + "." + subField
				if _, err = mappings.Put(key, val); err != nil {
					return nil, fmt.Errorf("failed to set mapping '%v' -> '%v': %w", dstField, docSrc, err)
				}
			}
		} else {
			prev, err := mappings.Put(dstField, docSrc)
			if err != nil {
				return nil, fmt.Errorf("failed to set mapping '%v' -> '%v': %w", dstField, docSrc, err)
			}
			if prev != nil {
				return nil, fmt.Errorf("field '%v' repeated", docSrc)
			}
		}
	}
	return mappings.Flatten(), nil
}

// constructPath returns a full JSON path given the prefix and target taking
// care to ensure that parent process attributes are placed directly within the
// parent object.
func constructPath(prefix, target string) string {
	if prefix == "parent." || strings.HasSuffix(prefix, ".parent.") {
		return prefix + strings.TrimPrefix(target, "process.")
	}
	return prefix + target
}
