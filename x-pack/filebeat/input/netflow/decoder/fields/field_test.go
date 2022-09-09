// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fields

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldDict_Merge(t *testing.T) {
	a := FieldDict{
		Key{1, 2}: &Field{"field1", String},
		Key{2, 3}: &Field{"field2", Unsigned32},
	}
	b := FieldDict{
		Key{3, 4}: &Field{"field3", MacAddress},
		Key{4, 5}: &Field{"field4", Ipv4Address},
		Key{5, 6}: &Field{"field5", Ipv6Address},
	}
	c := FieldDict{
		Key{3, 4}: &Field{"field3v2", OctetArray},
		Key{0, 0}: &Field{"field0", DateTimeMicroseconds},
	}

	f := FieldDict{}

	f.Merge(a)

	assert.Len(t, f, len(a))
	if !checkContains(t, f, a) {
		t.FailNow()
	}

	f.Merge(b)
	assert.Len(t, f, len(a)+len(b))
	if !checkContains(t, f, b) {
		t.FailNow()
	}

	f.Merge(c)
	assert.Len(t, f, len(a)+len(b)+len(c)-1)
	if !checkContains(t, f, c) {
		t.FailNow()
	}
}

func checkContains(t testing.TB, dest FieldDict, contains FieldDict) bool {
	for k, v := range contains {
		if !assert.Contains(t, dest, k) || !assert.Equal(t, *v, *dest[k]) {
			return false
		}
	}
	return true
}
