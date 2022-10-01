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

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestApplyOptions(t *testing.T) {
	cases := []struct {
		Description string
		Options     []ApplyOption
		Errors      multierror.Errors
		ExpectError bool
	}{
		{
			"all fields required, no error",
			[]ApplyOption{AllRequired},
			multierror.Errors{},
			false,
		},
		{
			"all fields required, an error",
			[]ApplyOption{AllRequired},
			multierror.Errors{
				NewKeyNotFoundError("foo"),
			},
			true,
		},
		{
			"all fields required, some other error, it should fail",
			[]ApplyOption{AllRequired},
			multierror.Errors{
				errors.New("something bad happened"),
			},
			true,
		},
		{
			"all fields required, an error, collecting missing keys doesn't alter result",
			[]ApplyOption{NotFoundKeys(func([]string) {}), AllRequired},
			multierror.Errors{
				NewKeyNotFoundError("foo"),
			},
			true,
		},
		{
			"fail on required, an error, not required",
			[]ApplyOption{FailOnRequired},
			multierror.Errors{
				&KeyNotFoundError{errorKey: errorKey{"foo"}, Required: false},
			},
			false,
		},
		{
			"fail on required, an error, required",
			[]ApplyOption{FailOnRequired},
			multierror.Errors{
				&KeyNotFoundError{errorKey: errorKey{"foo"}, Required: true},
			},
			true,
		},
		{
			"fail on required, some other error, it should fail",
			[]ApplyOption{FailOnRequired},
			multierror.Errors{
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
			assert.Error(t, errors.Err(), c.Description)
		} else {
			assert.NoError(t, errors.Err(), c.Description)
		}
	}
}

func TestNotFoundKeys(t *testing.T) {
	cases := []struct {
		Description string
		Errors      multierror.Errors
		Expected    []string
	}{
		{
			"empty errors, no key",
			multierror.Errors{},
			[]string{},
		},
		{
			"key not found error",
			multierror.Errors{
				NewKeyNotFoundError("foo"),
			},
			[]string{"foo"},
		},
		{
			"only another error, so no key",
			multierror.Errors{
				NewWrongFormatError("foo", ""),
			},
			[]string{},
		},
		{
			"two errors, only one is key not found",
			multierror.Errors{
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
