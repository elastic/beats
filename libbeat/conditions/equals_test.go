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

	"github.com/stretchr/testify/assert"
)

func TestEqualsCreate(t *testing.T) {
	config := Config{
		Equals: &Fields{fields: map[string]interface{}{
			"proc.pid": 0.08,
		}},
	}

	_, err := NewCondition(&config)
	assert.NotNil(t, err)
}

func TestEqualsSingleFieldPositiveMatch(t *testing.T) {
	testConfig(t, true, secdTestEvent, &Config{
		Equals: &Fields{fields: map[string]interface{}{
			"type": "process",
		}},
	})
}

func TestEqualsBooleanFieldNegativeMatch(t *testing.T) {
	testConfig(t, false, secdTestEvent, &Config{
		Equals: &Fields{fields: map[string]interface{}{
			"final": true,
		}},
	})
}

func TestEqualsMultiFieldAndTypePositiveMatch(t *testing.T) {
	testConfig(t, true, secdTestEvent, &Config{
		Equals: &Fields{fields: map[string]interface{}{
			"type":     "process",
			"proc.pid": 305,
		}},
	})
}
