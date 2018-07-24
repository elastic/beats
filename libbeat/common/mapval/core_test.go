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

package mapval

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

// assertResults validates the schema passed successfully.
func assertResults(t *testing.T, r Results) Results {
	for _, err := range r.Errors() {
		assert.NoError(t, err)
	}
	return r
}

func TestFlat(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"baz": 1,
	}

	results := Schema(Map{
		"foo": "bar",
		"baz": IsIntGt(0),
	})(m)

	assertResults(t, results)
}

func TestBadFlat(t *testing.T) {
	m := common.MapStr{}

	fakeT := new(testing.T)

	results := Schema(Map{
		"notafield": IsDuration,
	})(m)

	assertResults(fakeT, results)
	assert.True(t, fakeT.Failed())

	result := results["notafield"][0]
	assert.False(t, result.Valid)
	assert.Equal(t, result.Message, KeyMissingVR.Message)
}

func TestNested(t *testing.T) {
	m := common.MapStr{
		"foo": common.MapStr{
			"bar": "baz",
			"dur": time.Duration(100),
		},
	}

	results := Schema(Map{
		"foo": Map{
			"bar": "baz",
		},
		"foo.dur": IsDuration,
	})(m)

	assertResults(t, results)

	assert.Len(t, results, 2, "One result per matcher")
}

func TestComposition(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"baz": "bot",
	}

	fooValidator := Schema(Map{"foo": "bar"})
	bazValidator := Schema(Map{"baz": "bot"})
	composed := Compose(fooValidator, bazValidator)

	// Test that the validators work individually
	assertResults(t, fooValidator(m))
	assertResults(t, bazValidator(m))

	// Test that the composition of them works
	assertResults(t, composed(m))

	assert.Len(t, composed(m), 2)

	badValidator := Schema(Map{"notakey": "blah"})
	badComposed := Compose(badValidator, composed)

	fakeT := new(testing.T)
	assertResults(fakeT, badComposed(m))
	assert.Len(t, badComposed(m), 3)
	assert.True(t, fakeT.Failed())
}

func TestStrictFunc(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"baz": "bot",
		"nest": common.MapStr{
			"very": common.MapStr{
				"deep": "true",
			},
		},
	}

	validValidator := Schema(Map{
		"foo": "bar",
		"baz": "bot",
		"nest": Map{
			"very": Map{
				"deep": "true",
			},
		},
	})

	assertResults(t, validValidator(m))

	partialValidator := Schema(Map{
		"foo": "bar",
	})

	// Should pass, since this is not a strict check
	assertResults(t, partialValidator(m))

	res := Strict(partialValidator)(m)
	assert.Equal(t, []ValueResult{StrictFailureVR}, res.DetailedErrors()["baz"])
	assert.Equal(t, []ValueResult{StrictFailureVR}, res.DetailedErrors()["nest.very.deep"])
	assert.Nil(t, res.DetailedErrors()["bar"])
	assert.False(t, res.Valid())
}

func TestOptional(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
	}

	validator := Schema(Map{
		"non": Optional(IsEqual("foo")),
	})

	assertResults(t, validator(m))
}

func TestExistence(t *testing.T) {
	m := common.MapStr{
		"exists": "foo",
	}

	validator := Schema(Map{
		"exists": KeyPresent,
		"non":    KeyMissing,
	})

	assertResults(t, validator(m))
}

func TestComplex(t *testing.T) {
	m := common.MapStr{
		"foo": "bar",
		"hash": common.MapStr{
			"baz": 1,
			"bot": 2,
			"deep_hash": common.MapStr{
				"qux": "quark",
			},
		},
		"slice": []string{"pizza", "pasta", "and more"},
		"empty": nil,
	}

	res := Schema(Map{
		"foo": "bar",
		"hash": Map{
			"baz": 1,
			"bot": 2,
			"deep_hash": Map{
				"qux": "quark",
			},
		},
		"slice":        []string{"pizza", "pasta", "and more"},
		"empty":        KeyPresent,
		"doesNotExist": KeyMissing,
	})(m)

	assertResults(t, res)
}
