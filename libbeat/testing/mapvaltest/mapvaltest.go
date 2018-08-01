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

package mapvaltest

// skimatest is a separate package from skima since we don't want to import "testing"
// into skima, since there is a good chance we'll use skima for running user-defined
// tests in heartbeat at runtime.

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/davecgh/go-spew/spew"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
)

// Test takes the output from a Validator invocation and runs test assertions on the result.
// If you are using this library for testing you will probably want to run Test(t, Compile(Map{...}), actual) as a pattern.
func Test(t *testing.T, v mapval.Validator, m common.MapStr) *mapval.Results {
	r := v(m)

	if !r.Valid {
		assert.Fail(
			t,
			"mapval could not validate map",
			"%d errors validating source: \n%s", len(r.Errors()), spew.Sdump(m),
		)
	}

	for _, err := range r.Errors() {
		assert.NoError(t, err)
	}
	return r
}
