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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

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

	// CgroupRegex is the regular expression that captures the container id from cgroup path
	CgroupRegex string `config:"cgroup_regex"`

	// CgroupCacheExpireTime is the length of time before cgroup cache elements expire in seconds,
	// set to 0 to disable the cgroup cache
	CgroupCacheExpireTime time.Duration `config:"cgroup_cache_expire_time"`
}

// available fields by default
var defaultFields = common.MapStr{
	"process": common.MapStr{
		"name":       nil,
		"title":      nil,
		"executable": nil,
		"args":       nil,
		"pid":        nil,
		"parent": common.MapStr{
			"pid": nil,
		},
		"start_time": nil,
		"owner": common.MapStr{
			"name": nil,
			"id":   nil,
		},
	},
	"container": common.MapStr{
		"id": nil,
	},
}

// fields declared in here will only appear when requested explicitly
// with `restricted_fields: true`.
var restrictedFields = common.MapStr{
	"process": common.MapStr{
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
		CgroupPrefixes:        []string{"/kubepods", "/docker"},
		CgroupCacheExpireTime: cacheExpiration,
	}
}

func (pf *config) getMappings() (mappings common.MapStr, err error) {
	mappings = common.MapStr{}
	validFields := defaultFields
	if pf.RestrictedFields {
		validFields = restrictedFields
	}
	fieldPrefix := pf.Target
	if len(fieldPrefix) > 0 {
		fieldPrefix += "."
	}
	wantedFields := pf.Fields
	if len(wantedFields) == 0 {
		wantedFields = []string{"process", "container"}
	}
	for _, docSrc := range wantedFields {
		dstField := fieldPrefix + docSrc
		reqField, err := validFields.GetValue(docSrc)
		if err != nil {
			return nil, fmt.Errorf("field '%v' not found", docSrc)
		}
		if reqField != nil {
			for subField := range reqField.(common.MapStr) {
				key := dstField + "." + subField
				val := docSrc + "." + subField
				if _, err = mappings.Put(key, val); err != nil {
					return nil, errors.Wrapf(err, "failed to set mapping '%v' -> '%v'", dstField, docSrc)
				}
			}
		} else {
			prev, err := mappings.Put(dstField, docSrc)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to set mapping '%v' -> '%v'", dstField, docSrc)
			}
			if prev != nil {
				return nil, fmt.Errorf("field '%v' repeated", docSrc)
			}
		}
	}
	return mappings.Flatten(), nil
}
