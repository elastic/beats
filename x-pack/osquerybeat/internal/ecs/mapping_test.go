// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ecs

import "testing"

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
		"uid":         "user.id",
		"gid":         "user.group.id",
		"username":    "user.name",
		"description": "description",
		"uuid":        "a.b.c.d.e.f",
		"uid_signed":  "a.b.c.d.g",
	}

	doc := mapping.Map(testOsqueryResult)

	for src, dst := range mapping {
		val, ok := doc.Get(dst)
		if !ok {
			t.Errorf("key [%v] not found", dst)
			break
		}
		if testOsqueryResult[src] != val {
			t.Errorf("key [%v]=[%v], expected [%v]", src, val, testOsqueryResult[src])
		}
	}
}

func TestMapBadKeys(t *testing.T) {
	mapping := Mapping{
		"":           "",
		"foo":        "",
		"test":       "..",
		"uid_signed": "",
	}

	doc := mapping.Map(testOsqueryResult)

	for _, dst := range mapping {
		_, ok := doc.Get(dst)
		if ok {
			t.Errorf("key [%v] is expected to be not found", dst)
		}
	}
}
