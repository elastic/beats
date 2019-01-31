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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/go-ucfg/yaml"
)

func TestFieldsHasKey(t *testing.T) {
	tests := map[string]struct {
		key    string
		fields Fields
		result bool
	}{
		"empty fields": {
			key:    "test.find",
			fields: Fields{},
			result: false,
		},
		"unknown nested key": {
			key: "test.find",
			fields: Fields{
				Field{Name: "test"},
				Field{Name: "find"},
			},
			result: false,
		},
		"has": {
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
		"no leave node": {
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

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.result, test.fields.HasKey(test.key))
		})
	}
}

func TestDynamicYaml(t *testing.T) {
	tests := map[string]struct {
		input  []byte
		output Field
		error  bool
	}{
		"dynamic enabled": {
			input: []byte(`{name: test, dynamic: true}`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{true},
			},
		},
		"dynamic enabled2": {
			input: []byte(`{name: test, dynamic: "true"}`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{true},
			},
		},
		"invalid setting": {
			input: []byte(`{name: test, dynamic: "blue"}`),
			error: true,
		},
		"strict mode": {
			input: []byte(`{name: test, dynamic: "strict"}`),
			output: Field{
				Name:    "test",
				Dynamic: DynamicType{"strict"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			keys := Field{}

			cfg, err := yaml.NewConfig(test.input)
			assert.NoError(t, err)
			err = cfg.Unpack(&keys)

			if err != nil {
				assert.True(t, test.error)
			} else {
				assert.Equal(t, test.output.Dynamic, keys.Dynamic)
			}
		})
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

func TestFieldValidate(t *testing.T) {
	tests := map[string]struct {
		cfg   MapStr
		field Field
		err   bool
	}{
		"top level object type config": {
			cfg:   MapStr{"object_type": "scaled_float", "object_type_mapping_type": "float", "scaling_factor": 10},
			field: Field{ObjectType: "scaled_float", ObjectTypeMappingType: "float", ScalingFactor: 10},
			err:   false,
		},
		"multiple object type configs": {
			cfg: MapStr{"object_type_params": []MapStr{
				{"object_type": "scaled_float", "object_type_mapping_type": "float", "scaling_factor": 100}}},
			field: Field{ObjectTypeParams: []ObjectTypeCfg{{ObjectType: "scaled_float", ObjectTypeMappingType: "float", ScalingFactor: 100}}},
			err:   false,
		},
		"invalid config mixing object_type and object_type_params": {
			cfg: MapStr{
				"object_type":        "scaled_float",
				"object_type_params": []MapStr{{"object_type": "scaled_float", "object_type_mapping_type": "float"}}},
			err: true,
		},
		"invalid config mixing object_type_mapping_type and object_type_params": {
			cfg: MapStr{
				"object_type_mapping_type": "float",
				"object_type_params":       []MapStr{{"object_type": "scaled_float", "object_type_mapping_type": "float"}}},
			err: true,
		},
		"invalid config mixing scaling_factor and object_type_params": {
			cfg: MapStr{
				"scaling_factor":     100,
				"object_type_params": []MapStr{{"object_type": "scaled_float", "object_type_mapping_type": "float"}}},
			err: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := NewConfigFrom(test.cfg)
			require.NoError(t, err)
			var f Field
			err = cfg.Unpack(&f)
			if test.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.field, f)
			}
		})
	}
}

func TestFieldConcat(t *testing.T) {
	tests := map[string]struct {
		a, b Fields
		want Fields
		fail bool
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
			a:    Fields{{Name: "a"}},
			b:    Fields{{Name: "a"}},
			fail: true,
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
		"nested duplicates fail": {
			fail: true,
			a: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}, {Name: "c"}},
			}},
			b: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "c"}},
			}},
		},
		"a is prefix of b": {
			fail: true,
			a:    Fields{{Name: "a"}},
			b: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}},
			}},
		},
		"b is prefix of a": {
			fail: true,
			a: Fields{{
				Name:   "a",
				Fields: Fields{{Name: "b"}},
			}},
			b: Fields{{Name: "a"}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fs, err := ConcatFields(test.a, test.b)
			if test.fail {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, test.want, fs)
		})
	}
}
