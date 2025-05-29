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

package mage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_BuildArgs_ParseBuildTags(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect []string
	}{{
		name:   "no flags",
		input:  nil,
		expect: []string{},
	}, {
		name:   "multiple flags with no tags",
		input:  []string{"-a", "-b", "-key=value"},
		expect: []string{"-a", "-b", "-key=value"},
	}, {
		name:   "one build tag",
		input:  []string{"-tags=example"},
		expect: []string{"-tags=example"},
	}, {
		name:   "multiple build tags",
		input:  []string{"-tags=example", "-tags=test"},
		expect: []string{"-tags=example,test"},
	}, {
		name:   "joined build tags",
		input:  []string{"-tags=example,test"},
		expect: []string{"-tags=example,test"},
	}, {
		name:   "multiple build tags with other flags",
		input:  []string{"-tags=example", "-tags=test", "-key=value", "-a"},
		expect: []string{"-key=value", "-a", "-tags=example,test"},
	}, {
		name:   "incorrectly formatted tag",
		input:  []string{"-tags= example"},
		expect: []string{},
	}, {
		name:   "incorrectly formatted tag with valid tag",
		input:  []string{"-tags= example", "-tags=test"},
		expect: []string{"-tags=test"},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := BuildArgs{ExtraFlags: tc.input}
			flags := args.ParseBuildTags()
			assert.EqualValues(t, tc.expect, flags)
		})
	}
}
