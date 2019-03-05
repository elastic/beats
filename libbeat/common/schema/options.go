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
	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common"
)

// DefaultApplyOptions are the default options for Apply()
var DefaultApplyOptions = []ApplyOption{AllRequired}

// ApplyOption modifies the result of Apply
type ApplyOption func(common.MapStr, multierror.Errors) (common.MapStr, multierror.Errors)

// AllRequired considers any missing field as an error, except if explicitly
// set as optional
func AllRequired(event common.MapStr, errors multierror.Errors) (common.MapStr, multierror.Errors) {
	k := 0
	for i, err := range errors {
		if err, ok := err.(*KeyNotFoundError); ok {
			if err.Optional {
				continue
			}
		}
		errors[k] = errors[i]
		k++
	}
	return event, errors[:k]
}

// FailOnRequired considers missing fields as an error only if they are set
// as required
func FailOnRequired(event common.MapStr, errors multierror.Errors) (common.MapStr, multierror.Errors) {
	k := 0
	for i, err := range errors {
		if err, ok := err.(*KeyNotFoundError); ok {
			if !err.Required {
				continue
			}
		}
		errors[k] = errors[i]
		k++
	}
	return event, errors[:k]
}

// NotFoundKeys calls a function with the list of missing keys as parameter
func NotFoundKeys(cb func(keys []string)) ApplyOption {
	return func(event common.MapStr, errors multierror.Errors) (common.MapStr, multierror.Errors) {
		var keys []string
		for _, err := range errors {
			if err, ok := err.(*KeyNotFoundError); ok {
				keys = append(keys, err.Key())
			}
		}
		cb(keys)
		return event, errors
	}
}
