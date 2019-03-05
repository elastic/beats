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

package conditions

import "fmt"

// HasFields is a Condition for checking field existence.
type HasFields []string

// NewHasFieldsCondition builds a new HasFields checking the given list of fields.
func NewHasFieldsCondition(fields []string) (hasFieldsCondition HasFields) {
	return HasFields(fields)
}

// Check determines whether the given event matches this condition
func (c HasFields) Check(event ValuesMap) bool {
	for _, field := range c {
		_, err := event.GetValue(field)
		if err != nil {
			return false
		}
	}
	return true
}

func (c HasFields) String() string {
	return fmt.Sprintf("has_fields: %v", []string(c))
}
