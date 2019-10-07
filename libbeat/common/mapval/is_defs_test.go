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
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

func assertIsDefValid(t *testing.T, id IsDef, value interface{}) *Results {
	res := id.check(mustParsePath("p"), value, true)

	if !res.Valid {
		assert.Fail(
			t,
			"Expected Valid IsDef",
			"Isdef %#v was not valid for value %#v with error: ", id, value, res.Errors(),
		)
	}
	return res
}

func assertIsDefInvalid(t *testing.T, id IsDef, value interface{}) *Results {
	res := id.check(mustParsePath("p"), value, true)

	if res.Valid {
		assert.Fail(
			t,
			"Expected invalid IsDef",
			"Isdef %#v was should not have been valid for value %#v",
			id,
			value,
		)
	}
	return res
}

func TestIsArrayOf(t *testing.T) {
	validator := MustCompile(Map{"foo": "bar"})

	id := IsArrayOf(validator)

	goodMap := common.MapStr{"foo": "bar"}
	goodMapArr := []common.MapStr{goodMap, goodMap}

	goodRes := assertIsDefValid(t, id, goodMapArr)
	goodFields := goodRes.Fields
	assert.Len(t, goodFields, 2)
	assert.Contains(t, goodFields, "p.[0].foo")
	assert.Contains(t, goodFields, "p.[1].foo")

	badMap := common.MapStr{"foo": "bot"}
	badMapArr := []common.MapStr{badMap}

	badRes := assertIsDefInvalid(t, id, badMapArr)
	badFields := badRes.Fields
	assert.Len(t, badFields, 1)
	assert.Contains(t, badFields, "p.[0].foo")
}

func TestIsAny(t *testing.T) {
	id := IsAny(IsEqual("foo"), IsEqual("bar"))

	assertIsDefValid(t, id, "foo")
	assertIsDefValid(t, id, "bar")
	assertIsDefInvalid(t, id, "basta")
}

func TestIsEqual(t *testing.T) {
	id := IsEqual("foo")

	assertIsDefValid(t, id, "foo")
	assertIsDefInvalid(t, id, "bar")
}

func TestRegisteredIsEqual(t *testing.T) {
	// Time equality comes from a registered function
	// so this is a quick way to test registered functions
	now := time.Now()
	id := IsEqual(now)

	assertIsDefValid(t, id, now)
	assertIsDefInvalid(t, id, now.Add(100))
}

func TestIsString(t *testing.T) {
	assertIsDefValid(t, IsString, "abc")
	assertIsDefValid(t, IsString, "a")
	assertIsDefInvalid(t, IsString, 123)
}

func TestIsNonEmptyString(t *testing.T) {
	assertIsDefValid(t, IsNonEmptyString, "abc")
	assertIsDefValid(t, IsNonEmptyString, "a")
	assertIsDefInvalid(t, IsNonEmptyString, "")
	assertIsDefInvalid(t, IsString, 123)
}

func TestIsStringMatching(t *testing.T) {
	id := IsStringMatching(regexp.MustCompile(`^f`))

	assertIsDefValid(t, id, "fall")
	assertIsDefInvalid(t, id, "potato")
	assertIsDefInvalid(t, IsString, 123)
}

func TestIsStringContaining(t *testing.T) {
	id := IsStringContaining("foo")

	assertIsDefValid(t, id, "foo")
	assertIsDefValid(t, id, "a foo b")
	assertIsDefInvalid(t, id, "a bar b")
	assertIsDefInvalid(t, IsString, 123)
}

func TestIsDuration(t *testing.T) {
	id := IsDuration

	assertIsDefValid(t, id, time.Duration(1))
	assertIsDefInvalid(t, id, "foo")
}

func TestIsIntGt(t *testing.T) {
	id := IsIntGt(100)

	assertIsDefValid(t, id, 101)
	assertIsDefInvalid(t, id, 100)
	assertIsDefInvalid(t, id, 99)
}

func TestIsNil(t *testing.T) {
	assertIsDefValid(t, IsNil, nil)
	assertIsDefInvalid(t, IsNil, "foo")
}

func TestIsUnique(t *testing.T) {
	tests := []struct {
		name      string
		validator func() Validator
		data      common.MapStr
		isValid   bool
	}{
		{
			"IsUnique find dupes",
			func() Validator {
				v := IsUnique()
				return MustCompile(Map{"a": v, "b": v})
			},
			common.MapStr{"a": 1, "b": 1},
			false,
		},
		{
			"IsUnique separate instances don't care about dupes",
			func() Validator { return MustCompile(Map{"a": IsUnique(), "b": IsUnique()}) },
			common.MapStr{"a": 1, "b": 1},
			true,
		},
		{
			"IsUniqueTo duplicates across namespaces fail",
			func() Validator {
				s := ScopedIsUnique()
				return MustCompile(Map{"a": s.IsUniqueTo("test"), "b": s.IsUniqueTo("test2")})
			},
			common.MapStr{"a": 1, "b": 1},
			false,
		},

		{
			"IsUniqueTo duplicates within a namespace succeeds",
			func() Validator {
				s := ScopedIsUnique()
				return MustCompile(Map{"a": s.IsUniqueTo("test"), "b": s.IsUniqueTo("test")})
			},
			common.MapStr{"a": 1, "b": 1},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.isValid {
				Test(t, tt.validator(), tt.data)
			} else {
				result := tt.validator()(tt.data)
				require.False(t, result.Valid)
			}
		})
	}
}
