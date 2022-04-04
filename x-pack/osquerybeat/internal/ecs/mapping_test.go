// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ecs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testOsqueryResult = Doc{
	"uid":         275,
	"gid_signed":  275,
	"gid":         275,
	"shell":       "/usr/bin/false",
	"uid_signed":  275,
	"is_hidden":   0,
	"description": "Demo Daemon",
	"directory":   "/var/empty",
	"uuid":        "FFFFEEEE-DDDD-CCCC-BBBB-AAAA00000113",
	"username":    "_demod",
	"foo":         "bar",
	"test":        "testval",
}

func TestMap(t *testing.T) {
	mapping := Mapping{
		"user.id":       {Field: "uid"},
		"user.group.id": {Field: "gid"},
		"user.name":     {Field: "username"},
		"description":   {Field: "description"},
		"a.b.c.d.e.f":   {Field: "uuid"},
		"a.b.c.d.g":     {Field: "uid_signed"},
	}

	doc := Doc(mapping.Map(testOsqueryResult))

	for dst, mi := range mapping {
		val, ok := doc.Get(dst)
		if !ok {
			t.Errorf("key [%v] not found", dst)
			break
		}
		if testOsqueryResult[mi.Field] != val {
			t.Errorf("key [%v]=[%v], expected [%v]", mi.Field, val, testOsqueryResult[mi.Field])
		}
	}
}

func TestMapBadKeys(t *testing.T) {
	mapping := Mapping{
		"":   {Field: ""},
		"..": {Field: "test"},
	}

	doc := Doc(mapping.Map(testOsqueryResult))

	for _, mi := range mapping {
		_, ok := doc.Get(mi.Field)
		if ok {
			t.Errorf("key [%v] is expected to be not found", mi.Field)
		}
	}
}

func TestMapValue(t *testing.T) {
	mapping := Mapping{
		"value.empty":  {Value: ""},
		"value.zero":   {Value: 0},
		"value.number": {Value: 42},
		"value.map":    {Value: map[string]interface{}{"foo": "bar"}},
		"value.array":  {Value: []interface{}{"1234", "test", 42}},
	}

	doc := Doc(mapping.Map(testOsqueryResult))

	for dst, mi := range mapping {
		val, ok := doc.Get(dst)
		if !ok {
			t.Errorf("key [%v] not found", dst)
			break
		}
		diff := cmp.Diff(mi.Value, val)
		if diff != "" {
			t.Errorf("key [%v]=[%v], expected [%v]", dst, val, mi.Value)
		}
	}
}
