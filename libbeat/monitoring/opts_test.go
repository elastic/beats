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

// +build !integration

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	tests := []struct {
		name     string
		parent   *options
		options  []Option
		expected options
	}{
		{
			"empty parent without opts should generate defaults",
			nil,
			nil,
			defaultOptions,
		},
		{
			"non empty parent should return same options",
			&options{},
			nil,
			options{},
		},
		{
			"apply publishexpvar",
			&options{publishExpvar: false},
			[]Option{PublishExpvar},
			options{publishExpvar: true},
		},
		{
			"apply disable publishexpvar",
			&options{publishExpvar: true},
			[]Option{IgnorePublishExpvar},
			options{publishExpvar: false},
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.name)

		origParent := options{}
		if test.parent != nil {
			origParent = *test.parent
		}
		actual := applyOpts(test.parent, test.options)
		assert.NotNil(t, actual)

		// test parent has not been modified by accident
		if test.parent != nil {
			assert.Equal(t, origParent, *test.parent)
		}

		// check parent and actual are same object if options is nil
		if test.parent != nil && test.options == nil {
			assert.Equal(t, test.parent, actual)
		}

		// validate output
		assert.Equal(t, test.expected, *actual)
	}
}
