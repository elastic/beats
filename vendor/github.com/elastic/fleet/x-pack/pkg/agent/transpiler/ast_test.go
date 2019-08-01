// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package transpiler

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestAST(t *testing.T) {
	testcases := map[string]struct {
		hashmap map[string]interface{}
		ast     *AST
	}{
		"simple slice/string": {
			hashmap: map[string]interface{}{
				"inputs": []map[string]interface{}{
					map[string]interface{}{
						"paths": []string{"/var/log/log1", "/var/log/log2"},
					},
					map[string]interface{}{
						"paths": []string{"/var/log/log1", "/var/log/log2"},
					},
				},
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "inputs", value: &List{
							value: []Node{
								&Dict{
									value: []Node{
										&Key{name: "paths", value: &List{value: []Node{
											&StrVal{value: "/var/log/log1"},
											&StrVal{value: "/var/log/log2"},
										}}},
									},
								},
								&Dict{
									value: []Node{
										&Key{name: "paths", value: &List{value: []Node{
											&StrVal{value: "/var/log/log1"},
											&StrVal{value: "/var/log/log2"},
										}}},
									},
								},
							},
						},
						},
					},
				},
			},
		},
		"integer as key": {
			hashmap: map[string]interface{}{
				"1": []string{"/var/log/log1", "/var/log/log2"},
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "1", value: &List{value: []Node{
							&StrVal{value: "/var/log/log1"},
							&StrVal{value: "/var/log/log2"},
						}}},
					},
				},
			},
		},
		"support bool": {
			hashmap: map[string]interface{}{
				"true_v":  true,
				"false_v": false,
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "false_v", value: &BoolVal{value: false}},
						&Key{name: "true_v", value: &BoolVal{value: true}},
					},
				},
			},
		},
		"support integers": {
			hashmap: map[string]interface{}{
				"timeout": 12,
				"range":   []int{20, 30, 40},
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "range",
							value: &List{
								[]Node{
									&IntVal{value: 20},
									&IntVal{value: 30},
									&IntVal{value: 40},
								},
							},
						},
						&Key{name: "timeout", value: &IntVal{value: 12}},
					},
				},
			},
		},
		"support floats": {
			hashmap: map[string]interface{}{
				"ratio":   0.5,
				"range64": []float64{20.0, 30.0, 40.0},
				"range32": []float32{20.0, 30.0, 40.0},
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "range32",
							value: &List{
								[]Node{
									&FloatVal{value: 20.0},
									&FloatVal{value: 30.0},
									&FloatVal{value: 40.0},
								},
							},
						},
						&Key{
							name: "range64",
							value: &List{
								[]Node{
									&FloatVal{value: 20.0},
									&FloatVal{value: 30.0},
									&FloatVal{value: 40.0},
								},
							},
						},
						&Key{name: "ratio", value: &FloatVal{value: 0.5}},
					},
				},
			},
		},
		"Keys inside Keys with slices": {
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":         "log/docker",
					"ignore_older": "20s",
					"paths":        []string{"/var/log/log1", "/var/log/log2"},
				},
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: &Dict{
								[]Node{
									&Key{name: "ignore_older", value: &StrVal{value: "20s"}},
									&Key{name: "paths", value: &List{value: []Node{
										&StrVal{value: "/var/log/log1"},
										&StrVal{value: "/var/log/log2"},
									}}},
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								},
							}},
					},
				},
			},
		},
		"Keys with multiple levels of deeps": {
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":         "log/docker",
					"ignore_older": "20s",
					"paths":        []string{"/var/log/log1", "/var/log/log2"},
				},
				"outputs": map[string]interface{}{
					"elasticsearch": map[string]interface{}{
						"ssl": map[string]interface{}{
							"certificates_authorities": []string{"abc1", "abc2"},
						},
					},
				},
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: &Dict{
								[]Node{
									&Key{name: "ignore_older", value: &StrVal{value: "20s"}},
									&Key{name: "paths", value: &List{value: []Node{
										&StrVal{value: "/var/log/log1"},
										&StrVal{value: "/var/log/log2"},
									}}},
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								},
							}},
						&Key{
							name: "outputs",
							value: &Dict{
								[]Node{
									&Key{
										name: "elasticsearch",
										value: &Dict{
											[]Node{
												&Key{
													name: "ssl",
													value: &Dict{
														[]Node{
															&Key{name: "certificates_authorities",
																value: &List{
																	[]Node{
																		&StrVal{value: "abc1"},
																		&StrVal{value: "abc2"},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	t.Run("MAP to AST", func(t *testing.T) {
		for name, test := range testcases {
			t.Run(name, func(t *testing.T) {
				v, err := NewAST(test.hashmap)
				require.NoError(t, err)
				if !assert.True(t, yamlComparer(test.ast, v)) {
					diff := cmp.Diff(test.ast, v)
					t.Logf("Mismatch (-want, +got)\n%s", diff)
				}
			})
		}
	})

	t.Run("AST to MAP", func(t *testing.T) {
		for name, test := range testcases {
			t.Run(name, func(t *testing.T) {
				visitor := &MapVisitor{}
				test.ast.Accept(visitor)

				if !assert.True(t, yamlComparer(test.hashmap, visitor.Content)) {
					diff := cmp.Diff(test.hashmap, visitor.Content)
					t.Logf("Mismatch (-want, +got)\n%s", diff)
				}
			})
		}
	})
}

func TestSelector(t *testing.T) {
	testcases := map[string]struct {
		hashmap  map[string]interface{}
		selector Selector
		expected *AST
		notFound bool
	}{
		"two levels of keys": {
			selector: "inputs.type",
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":         "log/docker",
					"ignore_older": "20s",
					"paths":        []string{"/var/log/log1", "/var/log/log2"},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: &Dict{
								[]Node{
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								},
							}},
					},
				},
			},
		},
		"three level of keys": {
			selector: "inputs.ssl",
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":         "log/docker",
					"ignore_older": "20s",
					"paths":        []string{"/var/log/log1", "/var/log/log2"},
					"ssl": map[string]interface{}{
						"ca":          []string{"ca1", "ca2"},
						"certificate": "/etc/ssl/my.crt",
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: &Dict{
								[]Node{
									&Key{name: "ssl", value: &Dict{
										[]Node{
											&Key{name: "ca", value: &List{
												value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
											}},
											&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
										}}},
								},
							}},
					},
				},
			},
		},
		"indexed key access when it doesn't exist": {
			selector: "inputs.paths.1",
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":         "log/docker",
					"ignore_older": "20s",
					"paths":        []string{"/var/log/log1", "/var/log/log2"},
				},
			},
			notFound: true,
		},
		"integer in string for a key": {
			selector: "inputs.1",
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"1":            "log/docker",
					"ignore_older": "20s",
					"paths":        []string{"/var/log/log1", "/var/log/log2"},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: &Dict{
								[]Node{
									&Key{name: "1", value: &StrVal{value: "log/docker"}},
								},
							}},
					},
				},
			},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			v, err := NewAST(test.hashmap)
			require.NoError(t, err)
			a, ok := Select(v, test.selector)
			if test.notFound {
				require.False(t, ok)
				return
			}

			require.True(t, ok)
			if !assert.True(t, reflect.DeepEqual(test.expected, a)) {
				t.Logf(
					`received: %+v
					 expected: %+v`, v, test.expected)
			}
		})
	}
}

func TestCount(t *testing.T) {
	ast := &AST{
		root: &Dict{
			value: []Node{
				&Key{name: "inputs", value: &List{
					value: []Node{
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
					},
				},
				},
			},
		},
	}

	result := CountComp(ast, "inputs", func(a int) bool { return a == 2 })
	assert.True(t, result)

	result = CountComp(ast, "inputs2", func(a int) bool { return a == 0 })
	assert.True(t, result)
}

func TestMarshalerToYAML(t *testing.T) {
	ast := &AST{
		root: &Dict{
			value: []Node{
				&Key{name: "inputs", value: &List{
					value: []Node{
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
					},
				},
				},
			},
		},
	}

	b, err := yaml.Marshal(ast)
	require.NoError(t, err)

	expected := []byte(`inputs:
- paths:
  - /var/log/log1
  - /var/log/log2
- paths:
  - /var/log/log1
  - /var/log/log2
`)

	require.True(t, bytes.Equal(expected, b))
}

func yamlComparer(expected, candidate interface{}) bool {
	expectedYAML, err := yaml.Marshal(&expected)
	if err != nil {
		return false
	}

	candidateYAML, err := yaml.Marshal(&candidate)
	if err != nil {
		return false
	}

	return bytes.Equal(expectedYAML, candidateYAML)
}

func TestMarshalerToJSON(t *testing.T) {
	ast := &AST{
		root: &Dict{
			value: []Node{
				&Key{name: "inputs", value: &List{
					value: []Node{
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
					},
				},
				},
			},
		},
	}

	b, err := json.Marshal(ast)
	require.NoError(t, err)

	expected := []byte(`{"inputs":[{"paths":["/var/log/log1","/var/log/log2"]},{"paths":["/var/log/log1","/var/log/log2"]}]}`)
	require.True(t, bytes.Equal(expected, b))
}

func TestASTToMapStr(t *testing.T) {
	ast := &AST{
		root: &Dict{
			value: []Node{
				&Key{name: "inputs", value: &List{
					value: []Node{
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
						&Dict{
							value: []Node{
								&Key{name: "paths", value: &List{value: []Node{
									&StrVal{value: "/var/log/log1"},
									&StrVal{value: "/var/log/log2"},
								}}},
							},
						},
					},
				},
				},
			},
		},
	}

	m, err := ast.Map()
	require.NoError(t, err)

	expected := map[string]interface{}{
		"inputs": []interface{}{
			map[string]interface{}{
				"paths": []interface{}{"/var/log/log1", "/var/log/log2"},
			},
			map[string]interface{}{
				"paths": []interface{}{"/var/log/log1", "/var/log/log2"},
			},
		},
	}

	assert.True(t, reflect.DeepEqual(m, expected))
}
