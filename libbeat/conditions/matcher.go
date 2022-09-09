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

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/elastic-agent-libs/logp"
)

type matcherMap map[string]match.Matcher
type rawMap map[string]interface{}

// Matcher is a Condition that works with beat's internal notion of a string matcher.
type Matcher struct {
	name     string
	matchers matcherMap
	raw      rawMap
}

// NewMatcherCondition builds a new Matcher with the given human name using the provided config fields.
// The compiler function will take those fields and compile them.
func NewMatcherCondition(
	name string,
	fields map[string]interface{},
	compile func(string) (match.Matcher, error),
) (condition Matcher, err error) {
	condition.name = name
	condition.raw = fields
	condition.matchers = matcherMap{}
	condition.raw = rawMap{}

	if len(fields) == 0 {
		return condition, nil
	}

	for field, value := range fields {
		var err error

		switch v := value.(type) {
		case string:
			condition.matchers[field], err = compile(v)
			if err != nil {
				return condition, err
			}

		default:
			return condition, fmt.Errorf("unexpected type %T of %v", value, value)
		}
	}

	return condition, nil
}

// Check determines whether the given event matches this condition.
func (c Matcher) Check(event ValuesMap) bool {
	if c.matchers == nil {
		return true
	}

	for field, matcher := range c.matchers {
		value, err := event.GetValue(field)
		if err != nil {
			return false
		}

		switch v := value.(type) {
		case string:
			if !matcher.MatchString(v) {
				return false
			}

		case []interface{}, []string:
			if !matcher.MatchAnyString(v) {
				return false
			}
		default:
			str, err := ExtractString(value)
			if err != nil {
				logp.L().Named(logName).Warnf("unexpected type %T in %v condition as it accepts only strings.", value, c.name)
				return false
			}

			if !matcher.MatchString(str) {
				return false
			}
		}
	}

	return true
}

func (c Matcher) String() string {
	return fmt.Sprintf("%v: %v", c.name, c.raw)
}
