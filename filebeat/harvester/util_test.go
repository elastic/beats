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

//go:build !integration
// +build !integration

package harvester

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/common/match"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

// InitMatchers initializes a list of compiled regular expressions.
func InitMatchers(exprs ...string) ([]match.Matcher, error) {
	result := []match.Matcher{}

	for _, exp := range exprs {
		rexp, err := match.Compile(exp)
		if err != nil {
			logp.Err("Fail to compile the regexp %s: %s", exp, err)
			return nil, err
		}
		result = append(result, rexp)
	}
	return result, nil
}

func TestMatchAnyRegexps(t *testing.T) {
	matchers, err := InitMatchers("\\.gz$")
	assert.NoError(t, err)
	assert.Equal(t, MatchAny(matchers, "/var/log/log.gz"), true)
}

func TestExcludeLine(t *testing.T) {
	regexp, err := InitMatchers("^DBG")
	assert.NoError(t, err)
	assert.True(t, MatchAny(regexp, "DBG: a debug message"))
	assert.False(t, MatchAny(regexp, "ERR: an error message"))
}

func TestIncludeLine(t *testing.T) {
	regexp, err := InitMatchers("^ERR", "^WARN")

	assert.NoError(t, err)
	assert.False(t, MatchAny(regexp, "DBG: a debug message"))
	assert.True(t, MatchAny(regexp, "ERR: an error message"))
	assert.True(t, MatchAny(regexp, "WARNING: a simple warning message"))
}

func TestInitRegexp(t *testing.T) {
	_, err := InitMatchers("(((((")
	assert.Error(t, err)
}
