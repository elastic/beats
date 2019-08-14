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

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common/match"
)

var matchTestsWindows = []struct {
	file         string
	paths        []string
	excludeFiles []match.Matcher
	result       bool
}{
	{
		`C:\\hello\test\test.log`,
		[]string{`C:\\hello/test/*.log`},
		nil,
		true,
	},
	{
		`C:\\hello\test\test.log`,
		[]string{`C:\\hello\test/*.log`},
		nil,
		true,
	},
	{
		`C:\\hello\test\test.log`,
		[]string{`C://hello/test/*.log`},
		nil,
		true,
	},
	{
		`C:\\hello\test\test.log`,
		[]string{`C://hello//test//*.log`},
		nil,
		true,
	},
	{
		`C://hello/test/test.log`,
		[]string{`C:\\hello\test\*.log`},
		nil,
		true,
	},
	{
		`C://hello/test/test.log`,
		[]string{`C:/hello/test/*.log`},
		nil,
		true,
	},
}

// TestMatchFileWindows test if match works correctly on windows
// Separate test are needed on windows because of automated path conversion
func TestMatchFileWindows(t *testing.T) {
	for _, test := range matchTestsWindows {

		p := Input{
			config: config{
				Paths:        test.paths,
				ExcludeFiles: test.excludeFiles,
			},
		}

		assert.Equal(t, test.result, p.matchesFile(test.file))
	}
}
