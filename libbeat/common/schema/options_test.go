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

package schema

import (
	"testing"

	"errors"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestApplyOptions(t *testing.T) {
	cases := []struct {
		Description string
		Options     []ApplyOption
		Errors      []error
		ExpectError bool
	}{
		{
			"all fields required, no error",
			[]ApplyOption{AllRequired},
			nil,
			false,
		},
		{
			"all fields required, an error",
			[]ApplyOption{AllRequired},
			[]error{
				NewKeyNotFoundError("foo"),
			},
			true,
		},
		{
			"all fields required, some other error, it should fail",
			[]ApplyOption{AllRequired},
			[]error{
				errors.New("something bad happened"),
			},
			true,
		},
		{
			"all fields required, an error, collecting missing keys doesn't alter result",
			[]ApplyOption{NotFoundKeys(func([]string) {}), AllRequired},
			[]error{
				NewKeyNotFoundError("foo"),
			},
			true,
		},
		{
			"fail on required, an error, not required",
			[]ApplyOption{FailOnRequired},
			[]error{
				&KeyNotFoundError{errorKey: errorKey{"foo"}, Required: false},
			},
			false,
		},
		{
			"fail on required, an error, required",
			[]ApplyOption{FailOnRequired},
			[]error{
				&KeyNotFoundError{errorKey: errorKey{"foo"}, Required: true},
			},
			true,
		},
		{
			"fail on required, some other error, it should fail",
			[]ApplyOption{FailOnRequired},
			[]error{
				errors.New("something bad happened"),
			},
			true,
		},
	}

	for _, c := range cases {
		event := mapstr.M{}
		errors := c.Errors
		for _, opt := range c.Options {
			event, errors = opt(event, errors)
		}
		if c.ExpectError {
			assert.NotEmpty(t, errors, c.Description)
		} else {
			assert.Empty(t, errors, c.Description)
		}
	}
}

func TestNotFoundKeys(t *testing.T) {
	cases := []struct {
		Description string
		Errors      []error
		Expected    []string
	}{
		{
			"empty errors, no key",
			nil,
			[]string{},
		},
		{
			"key not found error",
			[]error{
				NewKeyNotFoundError("foo"),
			},
			[]string{"foo"},
		},
		{
			"only another error, so no key",
			[]error{
				NewWrongFormatError("foo", ""),
			},
			[]string{},
		},
		{
			"two errors, only one is key not found",
			[]error{
				NewKeyNotFoundError("foo"),
				NewWrongFormatError("bar", ""),
			},
			[]string{"foo"},
		},
	}

	for _, c := range cases {
		opt := NotFoundKeys(func(keys []string) {
			assert.ElementsMatch(t, c.Expected, keys, c.Description)
		})
		opt(mapstr.M{}, c.Errors)
	}
}
