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

package cfgwarn

import (
	"fmt"
	"strings"

	"github.com/joeshaw/multierror"

	"github.com/menderesk/beats/v7/libbeat/common"
)

func checkRemovedSettings(cfg *common.Config, settings ...string) error {
	var errs multierror.Errors
	for _, setting := range settings {
		if err := checkRemovedSetting(cfg, setting); err != nil {
			errs = append(errs, err)
		}
	}

	return errs.Err()
}

func checkRemovedSetting(cfg *common.Config, setting string) error {
	segments := strings.Split(setting, ".")

	L := len(segments)
	name := segments[L-1]
	path := segments[:L-1]

	current := cfg

	// we are looking for any key that match the name.
	for _, p := range path {
		current, _ = current.Child(p, -1)
		if current == nil {
			break
		}
	}

	// full path to setting not available -> setting not found
	if current == nil {
		return nil
	}

	if !current.HasField(name) {
		return nil
	}

	return fmt.Errorf("setting '%v' has been removed", current.PathOf(name))
}

// CheckRemoved6xSettings prints a warning if the obsolete setting is used.
func CheckRemoved6xSettings(cfg *common.Config, settings ...string) error {
	return checkRemovedSettings(cfg, settings...)
}

// CheckRemoved6xSetting prints a warning if the obsolete setting is used.
func CheckRemoved6xSetting(cfg *common.Config, setting string) error {
	return checkRemovedSetting(cfg, setting)
}
