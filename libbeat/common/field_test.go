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

package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-ucfg/yaml"
)

func TestFieldsHasNode(t *testing.T) {
	tests := map[string]struct {
		node    string
		fields  Fields
		hasNode bool
	}{
		"empty fields": {
			node:    "a.b",
			fields:  Fields{},
			hasNode: false,
		},
		"no node": {
			node:    "",
			fields:  Fields{Field{Name: "a"}},
			hasNode: false,
		},
		"key not in fields, but node in fields": {
			node: "a.b.c",
			fields: Fields{
				Field{Name: "a", Fields: Fields{Field{Name: "b"}}},
			},
			hasNode: true,
		},
		"last node in fields": {
			node: "a.b.c",
			fields: Fields{
				Field{Name: "a", Fields: Fields{
					Field{Name: "b", Fields: Fields{
						Field{Name: "c"},
					}}}},
			},
			hasNode: true,
		},
		"node in fields": {
			node: "a.b",
			fields: Fields{
				Field{Name: "a", Fields: Fields{
					Field{Name: "b", Fields: Fields{
						Field{Name: "c"},
					}}}},
			},
			hasNode: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.hasNode, test.fields.HasNode(test.node))
		})
	}
}

func TestFieldsHasKey(t *testing.T) {
	tests := []struct {
		key    string
		fields Fields
		result bool
	}{
		{
			key:    "test.find",
			fields: Fields{},
			result: false,
		},
		{
			key: "test.find",
			fields: Fields{
				Field{Name: "test"},
				Field{Name: "find"},
			},
			result: false,
		},
		{
			key: "test.find",
			fields: Fields{
				Field{
					Name: "test", Fields: Fields{
						Field{
							Name: "find",
						},
					},
				},
			},
			result: true,
		},
		{
			key: "test",
			fields: Fields{
				Field{
					Name: "test", Fields: Fields{
						Field{
							Name: "find",
						},
					},
				},
			},
			result: false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.result, test.fields.HasKey(test.key))
	}
}

func TestDynamicYaml(t *testing.T) {
	tests := []struct {
		input  []byte
		output Field
		error  bool
	}{
		{
			input: []byte(`
name: test
dynamic: true`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{true},
			},
		},
		{
			input: []byte(`
name: test
dynamic: "true"`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{true},
			},
		},
		{
			input: []byte(`
name: test
dynamic: "blue"`),
			error: true,
		},
		{
			input: []byte(`
name: test
dynamic: "strict"`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{"strict"},
			},
		},
	}

	for _, test := range tests {
		keys := Field{}

		cfg, err := yaml.NewConfig(test.input)
		assert.NoError(t, err)
		err = cfg.Unpack(&keys)

		if err != nil {
			assert.True(t, test.error)
		} else {
			assert.Equal(t, test.output.Dynamic, keys.Dynamic)
		}
	}
}

func TestGetKeys(t *testing.T) {
	tests := []struct {
		fields Fields
		keys   []string
	}{
		{
			fields: Fields{
				Field{
					Name: "test", Fields: Fields{
						Field{
							Name: "find",
						},
					},
				},
			},
			keys: []string{"test.find"},
		},
		{
			fields: Fields{
				Field{
					Name: "a", Fields: Fields{
						Field{
							Name: "b",
						},
					},
				},
				Field{
					Name: "a", Fields: Fields{
						Field{
							Name: "c",
						},
					},
				},
			},
			keys: []string{"a.b", "a.c"},
		},
		{
			fields: Fields{
				Field{
					Name: "a",
				},
				Field{
					Name: "b",
				},
				Field{
					Name: "c",
				},
			},
			keys: []string{"a", "b", "c"},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.keys, test.fields.GetKeys())
	}
}

func TestFieldConcat(t *testing.T) {
	tests := map[string]struct {
		a, b Fields
		want Fields
		err  string
	}{
		"empty lists": {},
		"first list only": {
			a:    Fields{{Name: "a"}},
			want: Fields{{Name: "a"}},
		},
		"second list only": {
			b:    Fields{{Name: "a"}},
			want: Fields{{Name: "a"}},
		},
		"concat": {
			a:    Fields{{Name: "a"}},
			b:    Fields{{Name: "b"}},
			want: Fields{{Name: "a"}, {Name: "b"}},
		},
		"duplicates fail": {
			a:   Fields{{Name: "a"}},
			b:   Fields{{Name: "a"}},
			err: "1 error: fields contain key <a>",
		},
		"nested with common prefix": {
			a: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}},
			}},
			b: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "c"}},
			}},
			want: Fields{
				{Name: "a", Fields: Fields{{Name: "b"}}},
				{Name: "a", Fields: Fields{{Name: "c"}}},
			},
		},
		"deep nested with common prefix": {
			a: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}},
			}},
			b: Fields{{
				Name: "a",
				Fields: Fields{{Name: "c", Fields: Fields{
					{Name: "d"},
				}}},
			}},
			want: Fields{
				{Name: "a", Fields: Fields{{Name: "b"}}},
				{Name: "a", Fields: Fields{{Name: "c", Fields: Fields{{Name: "d"}}}}},
			},
		},
		"nested duplicates fail": {
			a: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}, {Name: "c"}},
			}},
			b: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "c"}},
			}},
			err: "1 error: fields contain key <a.c>",
		},
		"a is prefix of b": {
			a: Fields{{Name: "a"}},
			b: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}},
			}},
			err: "1 error: fields contain non object node conflicting with key <a.b>",
		},
		"a is object and prefix of b": {
			a: Fields{{Name: "a", Type: "object"}},
			b: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}},
			}},
			want: Fields{
				{Name: "a", Type: "object"},
				{Name: "a", Fields: Fields{{Name: "b"}}},
			},
		},
		"b is prefix of a": {
			a: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}},
			}},
			b:   Fields{{Name: "a"}},
			err: "1 error: fields contain key <a>",
		},
		"multiple errors": {
			a: Fields{
				{Name: "a", Fields: Fields{{Name: "b"}}},
				{Name: "foo", Fields: Fields{{Name: "b"}}},
				{Name: "bar", Type: "object"},
			},
			b: Fields{
				{Name: "bar", Fields: Fields{{Name: "foo"}}},
				{Name: "a"},
				{Name: "foo", Fields: Fields{{Name: "b", Fields: Fields{{Name: "c"}}}}},
			},

			err: "2 errors: fields contain key <a>; fields contain non object node conflicting with key <foo.b.c>",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fs, err := ConcatFields(test.a, test.b)
			if test.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, test.want, fs)
				return
			}
			if assert.Error(t, err) {
				assert.Equal(t, test.err, err.Error())
			}
		})
	}
}

func TestFieldsCanConcat(t *testing.T) {
	tests := map[string]struct {
		key    string
		fields Fields
		err    string
	}{
		"empty fields": {
			key:    "a.b",
			fields: Fields{},
		},
		"no key": {
			key:    "",
			fields: Fields{Field{Name: "a"}},
		},
		"key not in fields, but parent node in fields": {
			key: "a.b.c",
			fields: Fields{
				Field{Name: "a", Fields: Fields{Field{Name: "b"}}},
			},
			err: "fields contain non object node conflicting with key <a.b.c>",
		},
		"key not in fields, but parent node in fields and of type object": {
			key: "a.b.c",
			fields: Fields{
				Field{Name: "a", Fields: Fields{Field{Name: "b", Type: "object"}}},
			},
		},
		"last node in fields": {
			key: "a.b.c",
			fields: Fields{
				Field{Name: "a", Fields: Fields{
					Field{Name: "b", Fields: Fields{
						Field{Name: "c"},
					}}}},
			},
			err: "fields contain key <a.b.c>",
		},
		"node in fields": {
			key: "a.b",
			fields: Fields{
				Field{Name: "a", Fields: Fields{
					Field{Name: "b", Fields: Fields{
						Field{Name: "c"},
					}}}},
			},
			err: "fields contain key <a.b>",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.fields.canConcat(test.key, strings.Split(test.key, "."))
			if test.err == "" {
				assert.Nil(t, err)
				return
			}
			if assert.Error(t, err) {
				assert.Equal(t, test.err, err.Error())
			}
		})
	}
}
