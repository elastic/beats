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
		hashmap     map[string]interface{}
		expectedMap map[string]interface{}
		ast         *AST
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
		"support null (nil) values": {
			hashmap: map[string]interface{}{
				"nil_v": nil,
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "nil_v"},
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
							value: NewList(
								[]Node{
									&IntVal{value: 20},
									&IntVal{value: 30},
									&IntVal{value: 40},
								},
							),
						},
						&Key{name: "timeout", value: &IntVal{value: 12}},
					},
				},
			},
		},
		"support unsigned integers": {
			hashmap: map[string]interface{}{
				"timeout": 12,
				"range":   []uint64{20, 30, 40},
			},
			ast: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "range",
							value: NewList(
								[]Node{
									&UIntVal{value: uint64(20)},
									&UIntVal{value: uint64(30)},
									&UIntVal{value: uint64(40)},
								},
							),
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
							value: NewList(
								[]Node{
									&FloatVal{value: 20.0},
									&FloatVal{value: 30.0},
									&FloatVal{value: 40.0},
								},
							),
						},
						&Key{
							name: "range64",
							value: NewList(
								[]Node{
									&FloatVal{value: 20.0},
									&FloatVal{value: 30.0},
									&FloatVal{value: 40.0},
								},
							),
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
							value: NewDict(
								[]Node{
									&Key{name: "ignore_older", value: &StrVal{value: "20s"}},
									&Key{name: "paths", value: &List{value: []Node{
										&StrVal{value: "/var/log/log1"},
										&StrVal{value: "/var/log/log2"},
									}}},
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								}),
						},
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
				root: NewDict(
					[]Node{
						&Key{
							name: "inputs",
							value: NewDict(
								[]Node{
									&Key{name: "ignore_older", value: &StrVal{value: "20s"}},
									&Key{name: "paths", value: &List{value: []Node{
										&StrVal{value: "/var/log/log1"},
										&StrVal{value: "/var/log/log2"},
									}}},
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								}),
						},
						&Key{
							name: "outputs",
							value: NewDict(
								[]Node{
									&Key{
										name: "elasticsearch",
										value: NewDict(
											[]Node{
												&Key{
													name: "ssl",
													value: NewDict(
														[]Node{
															&Key{name: "certificates_authorities",
																value: NewList(
																	[]Node{
																		&StrVal{value: "abc1"},
																		&StrVal{value: "abc2"},
																	},
																),
															},
														}),
												},
											}),
									},
								}),
						},
					}),
			},
		},
		"Keys with multiple levels of deeps with compact keys": {
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":         "log/docker",
					"ignore_older": "20s",
					"paths":        []string{"/var/log/log1", "/var/log/log2"},
				},
				"outputs.elasticsearch": map[string]interface{}{
					"ssl": map[string]interface{}{
						"certificates_authorities": []string{"abc1", "abc2"},
					},
				},
			},
			expectedMap: map[string]interface{}{
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
							value: NewDict(
								[]Node{
									&Key{name: "ignore_older", value: &StrVal{value: "20s"}},
									&Key{name: "paths", value: &List{value: []Node{
										&StrVal{value: "/var/log/log1"},
										&StrVal{value: "/var/log/log2"},
									}}},
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								}),
						},
						&Key{
							name: "outputs",
							value: NewDict(
								[]Node{
									&Key{
										name: "elasticsearch",
										value: NewDict(
											[]Node{
												&Key{
													name: "ssl",
													value: NewDict(
														[]Node{
															&Key{name: "certificates_authorities",
																value: NewList(
																	[]Node{
																		&StrVal{value: "abc1"},
																		&StrVal{value: "abc2"},
																	},
																),
															},
														}),
												},
											}),
									},
								}),
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

				expectedMap := test.hashmap
				if test.expectedMap != nil {
					expectedMap = test.expectedMap
				}

				if !assert.True(t, yamlComparer(expectedMap, visitor.Content)) {
					diff := cmp.Diff(test.hashmap, visitor.Content)
					t.Logf("Mismatch (-want, +got)\n%s", diff)
				}
			})
		}
	})
}

func TestInsert(t *testing.T) {
	testcases := map[string]struct {
		hashmap  map[string]interface{}
		selector Selector
		node     Node
		expected *AST
	}{
		"insert root": {
			selector: "inputs",
			node: NewList([]Node{
				NewDict([]Node{
					NewKey("type", NewStrVal("test-key")),
				}),
			}),
			hashmap: map[string]interface{}{
				"outputs": map[string]interface{}{
					"type": "elasticsearch",
					"host": "demo.host.co",
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict([]Node{
									NewKey("type", NewStrVal("test-key")),
								}),
							}),
						},
						&Key{
							name: "outputs",
							value: NewDict(
								[]Node{
									&Key{name: "host", value: &StrVal{value: "demo.host.co"}},
									&Key{name: "type", value: &StrVal{value: "elasticsearch"}},
								}),
						},
					},
				},
			},
		},
		"insert sub key": {
			selector: "outputs.sub",
			node: NewList([]Node{
				NewDict([]Node{
					NewKey("type", NewStrVal("test-key")),
				}),
			}),
			hashmap: map[string]interface{}{
				"outputs": map[string]interface{}{
					"type": "elasticsearch",
					"host": "demo.host.co",
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "outputs",
							value: NewDict(
								[]Node{
									&Key{name: "host", value: &StrVal{value: "demo.host.co"}},
									&Key{name: "sub", value: NewList([]Node{
										NewDict([]Node{
											NewKey("type", NewStrVal("test-key")),
										}),
									})},
									&Key{name: "type", value: &StrVal{value: "elasticsearch"}},
								}),
						},
					},
				},
			},
		},
		"insert at index": {
			selector: "inputs.0.sub",
			node: NewList([]Node{
				NewDict([]Node{
					NewKey("type", NewStrVal("test-key")),
				}),
			}),
			hashmap: map[string]interface{}{
				"inputs": []interface{}{
					map[string]interface{}{
						"type":         "log/docker",
						"ignore_older": "20s",
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList(
								[]Node{
									NewDict([]Node{
										NewKey("ignore_older", NewStrVal("20s")),
										NewKey("sub", NewList([]Node{
											NewDict([]Node{
												NewKey("type", NewStrVal("test-key")),
											}),
										})),
										NewKey("type", NewStrVal("log/docker")),
									}),
								}),
						},
					},
				},
			},
		},

		"insert at index when array empty": {
			selector: "inputs.0.sub",
			node: NewList([]Node{
				NewDict([]Node{
					NewKey("type", NewStrVal("test-key")),
				}),
			}),
			hashmap: map[string]interface{}{
				"inputs": make([]interface{}, 0),
				"outputs": map[string]interface{}{
					"type": "elasticsearch",
					"host": "demo.host.co",
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList(
								[]Node{
									NewDict(
										[]Node{
											NewKey("sub", NewList([]Node{
												NewDict([]Node{
													NewKey("type", NewStrVal("test-key")),
												}),
											})),
										},
									),
								}),
						},
						&Key{
							name: "outputs",
							value: NewDict(
								[]Node{
									NewKey("host", &StrVal{value: "demo.host.co"}),
									NewKey("type", &StrVal{value: "elasticsearch"}),
								}),
						},
					},
				},
			},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			ast, err := NewAST(test.hashmap)
			require.NoError(t, err)

			err = Insert(ast, test.node, test.selector)
			require.NoError(t, err)

			if !assert.True(t, reflect.DeepEqual(test.expected, ast)) {
				t.Logf(
					`received: %+v
					 expected: %+v`, ast, test.expected)
			}

		})
	}
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
							value: NewDict(
								[]Node{
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								}),
						},
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
							value: NewDict(
								[]Node{
									&Key{name: "ssl", value: NewDict(
										[]Node{
											&Key{name: "ca", value: &List{
												value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
											}},
											&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
										}),
									},
								}),
						},
					},
				},
			},
		},
		"indexed key access when it doesn't exist": {
			selector: "inputs.paths.2",
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
							value: NewDict(
								[]Node{
									&Key{name: "1", value: &StrVal{value: "log/docker"}},
								}),
						},
					},
				},
			},
		},
		"de-normalized hashmap": {
			selector: "inputs.x.ssl",
			hashmap: map[string]interface{}{
				"inputs.x": map[string]interface{}{
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
							value: NewDict(
								[]Node{
									&Key{name: "x", value: NewDict(
										[]Node{
											&Key{name: "ssl", value: NewDict(
												[]Node{
													&Key{name: "ca", value: &List{
														value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
													}},
													&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
												})},
										}),
									},
								}),
						},
					},
				},
			},
		},
		"de-normalized hashmap with duplicate prefix": {
			selector: "inputs",
			hashmap: map[string]interface{}{
				"inputs.x": map[string]interface{}{
					"ssl": map[string]interface{}{
						"ca":          []string{"ca1", "ca2"},
						"certificate": "/etc/ssl/my.crt",
					},
				},
				"inputs.y": map[string]interface{}{
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
							value: NewDict(
								[]Node{
									&Key{name: "x", value: NewDict(
										[]Node{
											&Key{name: "ssl", value: NewDict(
												[]Node{
													&Key{name: "ca", value: &List{
														value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
													}},
													&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
												})},
										}),
									},
									&Key{name: "y", value: NewDict(
										[]Node{
											&Key{name: "ssl", value: NewDict(
												[]Node{
													&Key{name: "ca", value: &List{
														value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
													}},
													&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
												})},
										}),
									},
								}),
						},
					},
				},
			},
		},
		"de-normalized hashmap with duplicate prefix, inputs.x.ssl selector": {
			selector: "inputs.x.ssl",
			hashmap: map[string]interface{}{
				"inputs.x": map[string]interface{}{
					"ssl": map[string]interface{}{
						"ca":          []string{"ca1", "ca2"},
						"certificate": "/etc/ssl/my.crt",
					},
				},
				"inputs.y": map[string]interface{}{
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
							value: NewDict(
								[]Node{
									&Key{name: "x", value: NewDict(
										[]Node{
											&Key{name: "ssl", value: NewDict(
												[]Node{
													&Key{name: "ca", value: &List{
														value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
													}},
													&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
												})},
										}),
									},
								}),
						},
					},
				},
			},
		},
		"de-normalized hashmap middle": {
			selector: "inputs.x.ssl",
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"x.ssl": map[string]interface{}{
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
							value: NewDict(
								[]Node{
									&Key{name: "x", value: NewDict(
										[]Node{
											&Key{name: "ssl", value: NewDict(
												[]Node{
													&Key{name: "ca", value: &List{
														value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
													}},
													&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
												})},
										}),
									},
								}),
						},
					},
				},
			},
		},
		"de-normalized hashmap middle with collision": {
			selector: "inputs.x.ssl",
			hashmap: map[string]interface{}{
				"inputs": map[string]interface{}{
					"x": map[string]interface{}{
						"ssl": map[string]interface{}{
							"ca": []string{"ca1", "ca2"},
						},
					},
					"x.ssl": map[string]interface{}{
						"certificate": "/etc/ssl/my.crt",
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewDict(
								[]Node{
									&Key{name: "x", value: NewDict(
										[]Node{
											&Key{name: "ssl", value: NewDict(
												[]Node{
													&Key{name: "ca", value: &List{
														value: []Node{&StrVal{value: "ca1"}, &StrVal{value: "ca2"}},
													}},
													&Key{name: "certificate", value: &StrVal{value: "/etc/ssl/my.crt"}},
												})},
										}),
									},
								}),
						},
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

func TestAST_Apply(t *testing.T) {
	testcases := map[string]struct {
		input    map[string]interface{}
		expected *AST
		vars     *Vars
		matchErr bool
	}{
		"2 vars missing with default": {
			input: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":  "log/docker",
					"paths": []string{"/var/log/${var1.key1}", "/var/log/${var1.missing|'other'}"},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewDict(
								[]Node{
									&Key{
										name: "paths",
										value: &List{
											value: []Node{
												&StrVal{value: "/var/log/value1"},
												&StrVal{value: "/var/log/other"},
											},
										},
									},
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								}),
						},
					},
				},
			},
			vars: mustMakeVars(map[string]interface{}{
				"var1": map[string]interface{}{
					"key1": "value1",
				},
			}),
		},
		"2 vars missing no default": {
			input: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":  "log/docker",
					"paths": []string{"/var/log/${var1.key1}", "/var/log/${var1.missing}"},
				},
			},
			vars: mustMakeVars(map[string]interface{}{
				"var1": map[string]interface{}{
					"key1": "value1",
				},
			}),
			matchErr: true,
		},
		"vars not string": {
			input: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":  "log/docker",
					"paths": []string{"/var/log/${var1.key1}"},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewDict(
								[]Node{
									&Key{
										name: "paths",
										value: &List{
											value: []Node{
												&StrVal{value: "/var/log/1"},
											},
										},
									},
									&Key{name: "type", value: &StrVal{value: "log/docker"}},
								}),
						},
					},
				},
			},
			vars: mustMakeVars(map[string]interface{}{
				"var1": map[string]interface{}{
					"key1": 1,
				},
			}),
		},
		"vars replace with object": {
			input: map[string]interface{}{
				"inputs": map[string]interface{}{
					"type":  "logfile",
					"paths": []string{"/var/log/syslog"},
					"processors": []map[string]interface{}{
						{
							"add_fields": map[string]interface{}{
								"labels": "${host.labels}",
							},
						},
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewDict(
								[]Node{
									&Key{
										name: "paths",
										value: &List{
											value: []Node{
												&StrVal{value: "/var/log/syslog"},
											},
										},
									},
									&Key{
										name: "processors",
										value: &List{
											value: []Node{
												NewDict(
													[]Node{
														&Key{
															name: "add_fields",
															value: NewDict(
																[]Node{
																	&Key{
																		name: "labels",
																		value: &List{
																			value: []Node{
																				&StrVal{value: "label1"},
																				&StrVal{value: "label2"},
																			},
																		},
																	},
																},
															),
														},
													},
												),
											},
										},
									},
									&Key{name: "type", value: &StrVal{value: "logfile"}},
								}),
						},
					},
				},
			},
			vars: mustMakeVars(map[string]interface{}{
				"host": map[string]interface{}{
					"labels": []string{
						"label1",
						"label2",
					},
				},
			}),
		},
		"condition false str removes dict from list": {
			input: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":  "logfile",
						"paths": []string{"/var/log/syslog"},
					},
					{
						"type":      "logfile",
						"paths":     []string{"/var/log/other"},
						"condition": "false",
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/syslog"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
							}),
						},
					},
				},
			},
		},
		"condition false removes dict from list": {
			input: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":  "logfile",
						"paths": []string{"/var/log/syslog"},
					},
					{
						"type":      "logfile",
						"paths":     []string{"/var/log/other"},
						"condition": false,
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/syslog"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
							}),
						},
					},
				},
			},
		},
		"condition true string keeps dict in list w/o condition key": {
			input: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":  "logfile",
						"paths": []string{"/var/log/syslog"},
					},
					{
						"type":      "logfile",
						"paths":     []string{"/var/log/other"},
						"condition": "true",
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/syslog"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/other"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
							}),
						},
					},
				},
			},
		},
		"condition true keeps dict in list w/o condition key": {
			input: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":  "logfile",
						"paths": []string{"/var/log/syslog"},
					},
					{
						"type":      "logfile",
						"paths":     []string{"/var/log/other"},
						"condition": true,
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/syslog"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/other"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
							}),
						},
					},
				},
			},
		},
		"condition eval keeps dict in list w/o condition key": {
			input: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":  "logfile",
						"paths": []string{"/var/log/syslog"},
					},
					{
						"type":      "logfile",
						"paths":     []string{"/var/log/other"},
						"condition": "arrayContains(${host.labels}, 'label2')",
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/syslog"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/other"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
							}),
						},
					},
				},
			},
			vars: mustMakeVars(map[string]interface{}{
				"host": map[string]interface{}{
					"labels": []string{
						"label1",
						"label2",
					},
				},
			}),
		},
		"condition eval removes dict from list": {
			input: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":  "logfile",
						"paths": []string{"/var/log/syslog"},
					},
					{
						"type":      "logfile",
						"paths":     []string{"/var/log/other"},
						"condition": "arrayContains(${host.labels}, 'missing')",
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/syslog"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
							}),
						},
					},
				},
			},
			vars: mustMakeVars(map[string]interface{}{
				"host": map[string]interface{}{
					"labels": []string{
						"label1",
						"label2",
					},
				},
			}),
		},
		"condition eval removes dict from dict": {
			input: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"type":  "logfile",
						"paths": []string{"/var/log/syslog"},
					},
					{
						"type":  "logfile",
						"paths": []string{"/var/log/other"},
						"nested": map[string]interface{}{
							"removed":   "whole dict removed",
							"condition": "arrayContains(${host.labels}, 'missing')",
						},
					},
				},
			},
			expected: &AST{
				root: &Dict{
					value: []Node{
						&Key{
							name: "inputs",
							value: NewList([]Node{
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/syslog"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
								NewDict(
									[]Node{
										&Key{
											name: "paths",
											value: &List{
												value: []Node{
													&StrVal{value: "/var/log/other"},
												},
											},
										},
										&Key{name: "type", value: &StrVal{value: "logfile"}},
									}),
							}),
						},
					},
				},
			},
			vars: mustMakeVars(map[string]interface{}{
				"host": map[string]interface{}{
					"labels": []string{
						"label1",
						"label2",
					},
				},
			}),
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			v, err := NewAST(test.input)
			require.NoError(t, err)
			err = v.Apply(test.vars)
			if test.matchErr {
				require.Equal(t, ErrNoMatch, err)
			} else {
				require.NoError(t, err)
				if !assert.True(t, reflect.DeepEqual(test.expected, v)) {
					t.Logf(
						`received: %+v
					 expected: %+v`, v, test.expected)
				}
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

func TestHash(t *testing.T) {
	tests := map[string]struct {
		c1    *AST
		c2    *AST
		match bool
	}{
		"same ast must match": {
			c1: &AST{
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
			c2: &AST{
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
			match: true,
		},
		"slice reordering doesn't match": {
			c1: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "inputs", value: &List{
							value: []Node{
								&Dict{
									value: []Node{
										&Key{name: "paths", value: &List{value: []Node{
											&StrVal{value: "/var/log/log2"},
											&StrVal{value: "/var/log/log1"},
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
			c2: &AST{
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
			match: false,
		},
		"match with int / float / bool": {
			c1: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "integer", value: &IntVal{value: 1}},
						&Key{name: "float", value: &FloatVal{value: 1.1234}},
						&Key{name: "bool1", value: &BoolVal{value: true}},
						&Key{name: "bool2", value: &BoolVal{value: false}},
					},
				},
			},
			c2: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "integer", value: &IntVal{value: 1}},
						&Key{name: "float", value: &FloatVal{value: 1.1234}},
						&Key{name: "bool1", value: &BoolVal{value: true}},
						&Key{name: "bool2", value: &BoolVal{value: false}},
					},
				},
			},
			match: true,
		},
		"different bool don't match": {
			c1: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &BoolVal{value: true}},
					},
				},
			},
			c2: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &BoolVal{value: false}},
					},
				},
			},
			match: false,
		},
		"different integer don't match": {
			c1: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &IntVal{value: 1}},
					},
				},
			},
			c2: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &IntVal{value: 2}},
					},
				},
			},
			match: false,
		},
		"different float don't match": {
			c1: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &FloatVal{value: 1.0}},
					},
				},
			},
			c2: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &FloatVal{value: 2.0}},
					},
				},
			},
			match: false,
		},
		"different floats representing the same value match": {
			c1: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &IntVal{value: 1}},
					},
				},
			},
			c2: &AST{
				root: &Dict{
					value: []Node{
						&Key{name: "v", value: &FloatVal{value: 1.0}},
					},
				},
			},
			match: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.match, test.c1.Equal(test.c2))
		})

		t.Run("test base64 string", func(t *testing.T) {
			assert.Equal(t, test.match, test.c1.HashStr() == test.c2.HashStr())
		})
	}
}

func mustMakeVars(mapping map[string]interface{}) *Vars {
	v, err := NewVars(mapping, nil)
	if err != nil {
		panic(err)
	}
	return v
}
