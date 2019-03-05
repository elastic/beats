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

package mapval

import "github.com/elastic/beats/libbeat/common"

type flatValidator struct {
	path  path
	isDef IsDef
}

// CompiledSchema represents a compiled definition for driving a Validator.
type CompiledSchema []flatValidator

// Check executes the the checks within the CompiledSchema
func (cs CompiledSchema) Check(actual common.MapStr) *Results {
	results := NewResults()
	for _, pv := range cs {
		actualV, actualKeyExists := pv.path.getFrom(actual)

		if !pv.isDef.optional || pv.isDef.optional && actualKeyExists {
			var checkRes *Results
			checkRes = pv.isDef.check(pv.path, actualV, actualKeyExists)
			results.merge(checkRes)
		}
	}

	return results
}
