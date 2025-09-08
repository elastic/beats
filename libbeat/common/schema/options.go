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

package schema

import (
	"errors"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// DefaultApplyOptions are the default options for Apply()
var DefaultApplyOptions = []ApplyOption{AllRequired}

// ApplyOption modifies the result of Apply
type ApplyOption func(mapstr.M, []error) (mapstr.M, []error)

// AllRequired considers any missing field as an error, except if explicitly
// set as optional
func AllRequired(event mapstr.M, errs []error) (mapstr.M, []error) {
	k := 0
	for i, err := range errs {
		var keyErr *KeyNotFoundError
		if errors.As(err, &keyErr) {
			if keyErr.Optional {
				continue
			}
		}
		errs[k] = errs[i]
		k++
	}
	return event, errs[:k]
}

// FailOnRequired considers missing fields as an error only if they are set
// as required
func FailOnRequired(event mapstr.M, errs []error) (mapstr.M, []error) {
	k := 0
	for i, err := range errs {
		var keyErr *KeyNotFoundError
		if errors.As(err, &keyErr) {
			if !keyErr.Required {
				continue
			}
		}
		errs[k] = errs[i]
		k++
	}
	return event, errs[:k]
}

// NotFoundKeys calls a function with the list of missing keys as parameter
func NotFoundKeys(cb func(keys []string)) ApplyOption {
	return func(event mapstr.M, errs []error) (mapstr.M, []error) {
		var keys []string
		for _, err := range errs {
			var keyErr *KeyNotFoundError
			if errors.As(err, &keyErr) {
				keys = append(keys, keyErr.Key())
			}
		}
		cb(keys)
		return event, errs
	}
}
