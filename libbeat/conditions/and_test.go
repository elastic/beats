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

	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestANDCondition(t *testing.T) {
	logp.TestingSetup()
	config := Config{
		AND: []Config{
			{
				Equals: &Fields{fields: map[string]interface{}{
					"client_server": "mar.local",
				}},
			},
			{
				Range: &Fields{fields: map[string]interface{}{
					"http.code.gte": 200,
					"http.code.lt":  300,
				}},
			},
		},
	}

	cond := GetCondition(t, config)

	assert.True(t, cond.Check(httpResponseTestEvent))
}
