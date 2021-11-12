// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	header = `function parse(m) {`
	footer = `}`
)

var log = logp.NewLogger("test")

func testMapStr() common.MapStr {
	return common.MapStr{
		"obj": common.MapStr{
			"key": "val",
		},
	}
}

func TestJSMapStr(t *testing.T) {
	logp.TestingSetup()

	type testCase struct {
		name   string
		source string
		assert func(t testing.TB, m common.MapStr, err error)
	}
	var cases = []testCase{
		{
			name:   "Put",
			source: `m.Put("hello", "world");`,
			assert: func(t testing.TB, m common.MapStr, err error) {
				v, _ := m.GetValue("hello")
				assert.Equal(t, "world", v)
			},
		},
		{
			name: "Get",
			source: `
				var v = m.Get("obj.key");

				if ("val" !== v) {
					throw "failed to get value";
				}`,
		},
		{
			name: "Get Object",
			source: `
				var o = m.Get("obj");

				  if ("val" !== o.key) {
					throw "failed to get value";
				  }`,
		},
		{
			name: "Get Undefined Key",
			source: `
				var v = m.Get().obj.key;

				  if ("val" !== v) {
					throw "failed to get value";
				  }`,
		},
		{
			name:   "Delete",
			source: `if (!m.Delete("obj.key")) { throw "delete failed"; }`,
			assert: func(t testing.TB, m common.MapStr, err error) {
				ip, _ := m.GetValue("obj.key")
				assert.Nil(t, ip)
			},
		},
		{
			name:   "Rename",
			source: `if (!m.Rename("obj", "renamed")) { throw "rename failed"; }`,
			assert: func(t testing.TB, m common.MapStr, err error) {
				v, _ := m.GetValue("renamed.key")
				assert.Equal(t, "val", v)
			},
		},
		{
			name:   "AppendTo",
			source: `m.AppendTo("obj.key", "val2");`,
			assert: func(t testing.TB, m common.MapStr, err error) {
				if assert.NoError(t, err) {
					vals, _ := m.GetValue("obj.key")
					assert.Equal(t, []string{"val", "val2"}, vals)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := newScriptFromConfig(log, &scriptConfig{Source: header + tc.source + footer})
			if err != nil {
				t.Fatal(err)
			}

			m := testMapStr()
			if _, err := p.run(m); tc.assert != nil {
				tc.assert(t, m, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}
