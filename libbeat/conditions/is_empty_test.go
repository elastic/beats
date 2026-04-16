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
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var isEmptyTestEvent = &beat.Event{
	Timestamp: time.Now(),
	Fields: mapstr.M{
		"empty_string":  "",
		"non_empty":     "hello",
		"nil_value":     nil,
		"empty_map":     mapstr.M{},
		"non_empty_map": mapstr.M{"key": "value"},
		"empty_raw_map": map[string]interface{}{},
		"non_empty_raw_map": map[string]interface{}{
			"key": "value",
		},
	},
}

func TestIsEmptyEmptyStringPositiveMatch(t *testing.T) {
	testConfig(t, true, isEmptyTestEvent, &Config{
		IsEmpty: "empty_string",
	})
}

func TestIsEmptyNonEmptyStringNegativeMatch(t *testing.T) {
	testConfig(t, false, isEmptyTestEvent, &Config{
		IsEmpty: "non_empty",
	})
}

func TestIsEmptyNilValuePositiveMatch(t *testing.T) {
	testConfig(t, true, isEmptyTestEvent, &Config{
		IsEmpty: "nil_value",
	})
}

func TestIsEmptyMissingFieldNegativeMatch(t *testing.T) {
	testConfig(t, false, isEmptyTestEvent, &Config{
		IsEmpty: "does_not_exist",
	})
}

func TestIsEmptyEmptyMapstrPositiveMatch(t *testing.T) {
	testConfig(t, true, isEmptyTestEvent, &Config{
		IsEmpty: "empty_map",
	})
}

func TestIsEmptyNonEmptyMapstrNegativeMatch(t *testing.T) {
	testConfig(t, false, isEmptyTestEvent, &Config{
		IsEmpty: "non_empty_map",
	})
}

func TestIsEmptyEmptyRawMapPositiveMatch(t *testing.T) {
	testConfig(t, true, isEmptyTestEvent, &Config{
		IsEmpty: "empty_raw_map",
	})
}

func TestIsEmptyNonEmptyRawMapNegativeMatch(t *testing.T) {
	testConfig(t, false, isEmptyTestEvent, &Config{
		IsEmpty: "non_empty_raw_map",
	})
}
