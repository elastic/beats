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

	"github.com/elastic/beats/libbeat/common"
)

type config struct {
	// IgnoreMissing: Ignore errors if event has no PID field.
	IgnoreMissing bool `config:"ignore_missing"`

	// OverwriteFields allow target_fields to overwrite existing fields.
	OverwriteFields bool `config:"overwrite_fields"`

	// RestrictedFields make restricted fields available (i.e. env).
	RestrictedFields bool `config:"restricted_fields"`

	// IgnoreErrors causes the event not to be discarded when errors
	// are encountered during enrichment.
	// For example when a PID field cannot be converted to an integer or
	// a field cannot be overwritten. It is still an error if no PID field
	// is found. Use `ignore_missing` for that.
	// Instead of the event being dropped, it will return the original event.
	IgnoreErrors bool `config:"ignore_errors"`

	// MatchPIDs fields containing the PID to lookup.
	MatchPIDs []string `config:"match_pids"`

	// TargetFields maps from destination field to value.
	TargetFields common.MapStr `config:"target_fields"`
}

// available fields by default
var validFields = common.MapStr{
	"process": common.MapStr{
		"name":       nil,
		"title":      nil,
		"exe":        nil,
		"args":       nil,
		"pid":        nil,
		"ppid":       nil,
		"start_time": nil,
	},
}

// fields declared in here (and not in validFields) will only appear in an
// event if requested explicitly (restricted_fields: true).
var restrictedFields = common.MapStr{
	"process": common.MapStr{
		"name":       nil,
		"title":      nil,
		"exe":        nil,
		"args":       nil,
		"env":        nil,
		"pid":        nil,
		"ppid":       nil,
		"start_time": nil,
	},
}

func defaultConfig() config {
	return config{
		IgnoreMissing:    true,
		OverwriteFields:  false,
		RestrictedFields: false,
	}
}

func (pf *config) getMappings() (mappings common.MapStr, err error) {
	mappings = common.MapStr{}
	validFields := validFields
	if pf.RestrictedFields {
		validFields = restrictedFields
	}
	for key, srcIf := range pf.TargetFields.Flatten() {
		targetField, ok := srcIf.(string)
		if !ok {
			if srcIf != nil {
				return nil, fmt.Errorf("field '%v' not found (not a string)", srcIf)
			}
			// allow nil to be a synonym for process
			targetField = "process"
		}

		var val interface{}
		var err error
		if val, err = validFields.GetValue(targetField); err != nil {
			return nil, fmt.Errorf("field '%v' not found", targetField)
		}
		if val == nil {
			mappings.Put(key, targetField)
		} else {
			for subkey := range val.(common.MapStr) {
				mappings.Put(key+"."+subkey, targetField+"."+subkey)
			}
		}
	}

	return mappings.Flatten(), nil
}
