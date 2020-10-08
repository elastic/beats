// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
)

func TestRenderInputs(t *testing.T) {
	testcases := map[string]struct {
		input     transpiler.Node
		expected  transpiler.Node
		varsArray []*transpiler.Vars
		err       bool
	}{
		"inputs not list": {
			input: transpiler.NewKey("inputs", transpiler.NewStrVal("not list")),
			err:   true,
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{}),
			},
		},
		"bad variable error": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.name|'missing ending quote}")),
				}),
			})),
			err: true,
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
					},
				}),
			},
		},
		"basic single var": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.name}")),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value1")),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
					},
				}),
			},
		},
		"duplicate result is removed": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.name}")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.diff}")),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value1")),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value1",
					},
				}),
			},
		},
		"missing var removes input": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.name}")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.missing|var1.diff}")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.removed}")),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value1")),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value1",
					},
				}),
			},
		},
		"duplicate var result but unique input not removed": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.name}")),
					transpiler.NewKey("unique", transpiler.NewStrVal("0")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.diff}")),
					transpiler.NewKey("unique", transpiler.NewStrVal("1")),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value1")),
					transpiler.NewKey("unique", transpiler.NewStrVal("0")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value1")),
					transpiler.NewKey("unique", transpiler.NewStrVal("1")),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value1",
					},
				}),
			},
		},
		"duplicates across vars array handled": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.name}")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("${var1.diff}")),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value1")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value2")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value3")),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("key", transpiler.NewStrVal("value4")),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value1",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value2",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value3",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value2",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
						"diff": "value4",
					},
				}),
			},
		},
		"nested in streams": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/${var1.name}.log"),
							})),
						}),
					})),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value1.log"),
							})),
						}),
					})),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value2.log"),
							})),
						}),
					})),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value3.log"),
							})),
						}),
					})),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value4.log"),
							})),
						}),
					})),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value2",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value2",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value3",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value4",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"missing": "other",
					},
				}),
			},
		},
		"inputs with processors": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/${var1.name}.log"),
							})),
						}),
					})),
					transpiler.NewKey("processors", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("user", transpiler.NewStrVal("user1")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("user")),
							})),
						}),
					})),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value1.log"),
							})),
						}),
					})),
					transpiler.NewKey("processors", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("user", transpiler.NewStrVal("user1")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("user")),
							})),
						}),
					})),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value2.log"),
							})),
						}),
					})),
					transpiler.NewKey("processors", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("user", transpiler.NewStrVal("user1")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("user")),
							})),
						}),
					})),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
					},
				}),
				mustMakeVars(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value2",
					},
				}),
			},
		},
		"vars with processors": {
			input: transpiler.NewKey("inputs", transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/${var1.name}.log"),
							})),
						}),
					})),
					transpiler.NewKey("processors", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("user", transpiler.NewStrVal("user1")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("user")),
							})),
						}),
					})),
				}),
			})),
			expected: transpiler.NewList([]transpiler.Node{
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value1.log"),
							})),
						}),
					})),
					transpiler.NewKey("processors", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("custom", transpiler.NewStrVal("value1")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("dynamic")),
							})),
						}),
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("user", transpiler.NewStrVal("user1")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("user")),
							})),
						}),
					})),
				}),
				transpiler.NewDict([]transpiler.Node{
					transpiler.NewKey("type", transpiler.NewStrVal("logfile")),
					transpiler.NewKey("streams", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("paths", transpiler.NewList([]transpiler.Node{
								transpiler.NewStrVal("/var/log/value2.log"),
							})),
						}),
					})),
					transpiler.NewKey("processors", transpiler.NewList([]transpiler.Node{
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("custom", transpiler.NewStrVal("value2")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("dynamic")),
							})),
						}),
						transpiler.NewDict([]transpiler.Node{
							transpiler.NewKey("add_fields", transpiler.NewDict([]transpiler.Node{
								transpiler.NewKey("fields", transpiler.NewDict([]transpiler.Node{
									transpiler.NewKey("user", transpiler.NewStrVal("user1")),
								})),
								transpiler.NewKey("to", transpiler.NewStrVal("user")),
							})),
						}),
					})),
				}),
			}),
			varsArray: []*transpiler.Vars{
				mustMakeVarsP(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value1",
					},
				},
					"var1",
					[]map[string]interface{}{
						{
							"add_fields": map[string]interface{}{
								"fields": map[string]interface{}{
									"custom": "value1",
								},
								"to": "dynamic",
							},
						},
					}),
				mustMakeVarsP(map[string]interface{}{
					"var1": map[string]interface{}{
						"name": "value2",
					},
				},
					"var1",
					[]map[string]interface{}{
						{
							"add_fields": map[string]interface{}{
								"fields": map[string]interface{}{
									"custom": "value2",
								},
								"to": "dynamic",
							},
						},
					}),
			},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			v, err := renderInputs(test.input, test.varsArray)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected.String(), v.String())
			}
		})
	}
}

func mustMakeVars(mapping map[string]interface{}) *transpiler.Vars {
	v, err := transpiler.NewVars(mapping)
	if err != nil {
		panic(err)
	}
	return v
}

func mustMakeVarsP(mapping map[string]interface{}, processorKey string, processors transpiler.Processors) *transpiler.Vars {
	v, err := transpiler.NewVarsWithProcessors(mapping, processorKey, processors)
	if err != nil {
		panic(err)
	}
	return v
}
