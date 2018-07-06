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

// Not is a condition that negates its inner condition.
type Not struct {
	inner Condition
}

// NewNotCondition builds a new Not condition that negates the provided Condition.
func NewNotCondition(c Condition) (Not, error) {
	if c == nil {
		return Not{}, fmt.Errorf("Empty not conditions are not allowed")
	}
	return Not{c}, nil
}

// Check determines whether the given event matches this condition.
func (c Not) Check(event ValuesMap) bool {
	return !c.inner.Check(event)
}

func (c Not) String() string {
	return "!" + c.inner.String()
}
