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

package cleanup_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/cleanup"
)

func TestIfBool(t *testing.T) {
	testcases := []struct {
		title   string
		fn      func(*bool, func())
		value   bool
		cleanup bool
	}{
		{
			"IfNot runs cleanup",
			cleanup.IfNot, false, true,
		},
		{
			"IfNot does not run cleanup",
			cleanup.IfNot, true, false,
		},
		{
			"If runs cleanup",
			cleanup.If, true, true,
		},
		{
			"If does not run cleanup",
			cleanup.If, false, false,
		},
	}

	for _, test := range testcases {
		test := test
		t.Run(test.title, func(t *testing.T) {
			executed := false
			func() {
				v := test.value
				defer test.fn(&v, func() { executed = true })
			}()

			assert.Equal(t, test.cleanup, executed)
		})
	}
}
