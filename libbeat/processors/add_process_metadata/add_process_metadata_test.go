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

package add_process_metadata

import (
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestAddProcessMetadata(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(processorName))
	startTime := time.Now()
	testProcs := testProvider{
		1: {
			name:  "systemd",
			title: "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
			exe:   "/usr/lib/systemd/systemd",
			args:  []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
			env: map[string]string{
				"HOME":       "/",
				"TERM":       "linux",
				"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
				"LANG":       "en_US.UTF-8",
			},
			pid:       1,
			ppid:      0,
			startTime: startTime,
		},
	}

	for _, test := range []struct {
		description             string
		config, event, expected common.MapStr
		err, initErr            error
	}{
		{
			description: "default fields",
			config: common.MapStr{
				"match_pids": []string{"system.process.ppid"},
			},
			event: common.MapStr{
				"system": common.MapStr{
					"process": common.MapStr{
						"ppid": "1",
					},
				},
			},
			expected: common.MapStr{
				"system": common.MapStr{
					"process": common.MapStr{
						"ppid": "1",
					},
				},
				"process": common.MapStr{
					"name":       "systemd",
					"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
					"executable": "/usr/lib/systemd/systemd",
					"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
					"pid":        1,
					"ppid":       0,
					"start_time": startTime,
				},
			},
		},
		{
			description: "single field",
			config: common.MapStr{
				"match_pids":     []string{"system.process.ppid"},
				"target":         "system.process.parent",
				"include_fields": []string{"process.name"},
			},
			event: common.MapStr{
				"system": common.MapStr{
					"process": common.MapStr{
						"ppid": "1",
					},
				},
			},
			expected: common.MapStr{
				"system": common.MapStr{
					"process": common.MapStr{
						"ppid": "1",
						"parent": common.MapStr{
							"process": common.MapStr{
								"name": "systemd",
							},
						},
					},
				},
			},
		},
		{
			description: "multiple fields",
			config: common.MapStr{
				"match_pids":     []string{"system.other.pid", "system.process.ppid"},
				"target":         "extra",
				"include_fields": []string{"process.title", "process.start_time"},
			},
			event: common.MapStr{
				"system": common.MapStr{
					"process": common.MapStr{
						"ppid": "1",
					},
				},
			},
			expected: common.MapStr{
				"system": common.MapStr{
					"process": common.MapStr{
						"ppid": "1",
					},
				},
				"extra": common.MapStr{
					"process": common.MapStr{
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"start_time": startTime,
					},
				},
			},
		},
		{
			description: "complete process info",
			config: common.MapStr{
				"match_pids": []string{"ppid"},
				"target":     "parent",
			},
			event: common.MapStr{
				"ppid": "1",
			},
			expected: common.MapStr{
				"ppid": "1",
				"parent": common.MapStr{
					"process": common.MapStr{
						"name":       "systemd",
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"executable": "/usr/lib/systemd/systemd",
						"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
						"pid":        1,
						"ppid":       0,
						"start_time": startTime,
					},
				},
			},
		},
		{
			description: "complete process info (restricted fields)",
			config: common.MapStr{
				"match_pids":        []string{"ppid"},
				"restricted_fields": true,
				"target":            "parent",
			},
			event: common.MapStr{
				"ppid": "1",
			},
			expected: common.MapStr{
				"ppid": "1",
				"parent": common.MapStr{
					"process": common.MapStr{
						"name":       "systemd",
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"executable": "/usr/lib/systemd/systemd",
						"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
						"pid":        1,
						"ppid":       0,
						"start_time": startTime,
						"env": map[string]string{
							"HOME":       "/",
							"TERM":       "linux",
							"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
							"LANG":       "en_US.UTF-8",
						},
					},
				},
			},
		},
		{
			description: "complete process info (restricted fields - alt)",
			config: common.MapStr{
				"match_pids":        []string{"ppid"},
				"restricted_fields": true,
				"target":            "parent",
				"include_fields":    []string{"process"},
			},
			event: common.MapStr{
				"ppid": "1",
			},
			expected: common.MapStr{
				"ppid": "1",
				"parent": common.MapStr{
					"process": common.MapStr{
						"name":       "systemd",
						"title":      "/usr/lib/systemd/systemd --switched-root --system --deserialize 22",
						"executable": "/usr/lib/systemd/systemd",
						"args":       []string{"/usr/lib/systemd/systemd", "--switched-root", "--system", "--deserialize", "22"},
						"pid":        1,
						"ppid":       0,
						"start_time": startTime,
						"env": map[string]string{
							"HOME":       "/",
							"TERM":       "linux",
							"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
							"LANG":       "en_US.UTF-8",
						},
					},
				},
			},
		},
		{
			description: "env field (restricted_fields: true)",
			config: common.MapStr{
				"match_pids":        []string{"ppid"},
				"restricted_fields": true,
				"target":            "parent",
				"include_fields":    []string{"process.env"},
			},
			event: common.MapStr{
				"ppid": "1",
			},
			expected: common.MapStr{
				"ppid": "1",
				"parent": common.MapStr{
					"process": common.MapStr{
						"env": map[string]string{
							"HOME":       "/",
							"TERM":       "linux",
							"BOOT_IMAGE": "/boot/vmlinuz-4.11.8-300.fc26.x86_64",
							"LANG":       "en_US.UTF-8",
						},
					},
				},
			},
		},
		{
			description: "env field (restricted_fields: false)",
			config: common.MapStr{
				"match_pids":     []string{"ppid"},
				"target":         "parent",
				"include_fields": []string{"process.env"},
			},
			event: common.MapStr{
				"ppid": "1",
			},
			expected: nil,
			initErr:  errors.New("error unpacking add_process_metadata.target_fields: field 'process.env' not found"),
		},
		{
			description: "fields not found (ignored)",
			config: common.MapStr{
				"match_pids": []string{"ppid"},
			},
			event: common.MapStr{
				"other": "field",
			},
			expected: common.MapStr{
				"other": "field",
			},
		},
		{
			description: "fields not found (reported)",
			config: common.MapStr{
				"match_pids":     []string{"ppid"},
				"ignore_missing": false,
			},
			event: common.MapStr{
				"other": "field",
			},
			expected: common.MapStr{
				"other": "field",
			},
			err: ErrNoMatch,
		},
		{
			description: "overwrite keys",
			config: common.MapStr{
				"overwrite_keys": true,
				"match_pids":     []string{"ppid"},
				"include_fields": []string{"process.name"},
			},
			event: common.MapStr{
				"ppid": 1,
				"process": common.MapStr{
					"name": "other",
				},
			},
			expected: common.MapStr{
				"ppid": 1,
				"process": common.MapStr{
					"name": "systemd",
				},
			},
		},
		{
			description: "overwrite keys error",
			config: common.MapStr{
				"match_pids":     []string{"ppid"},
				"include_fields": []string{"process.name"},
			},
			event: common.MapStr{
				"ppid": 1,
				"process": common.MapStr{
					"name": "other",
				},
			},
			expected: common.MapStr{
				"ppid": 1,
				"process": common.MapStr{
					"name": "other",
				},
			},
			err: errors.New("error applying add_process_metadata processor: target field 'process.name' already exists and overwrite_keys is false"),
		},
		{
			description: "bad PID field cast",
			config: common.MapStr{
				"match_pids": []string{"ppid"},
			},
			event: common.MapStr{
				"ppid": "a",
			},
			expected: common.MapStr{
				"ppid": "a",
			},
			err: errors.New("error applying add_process_metadata processor: cannot convert string field 'ppid' to an integer: strconv.Atoi: parsing \"a\": invalid syntax"),
		},
		{
			description: "bad PID field type",
			config: common.MapStr{
				"match_pids": []string{"ppid"},
			},
			event: common.MapStr{
				"ppid": false,
			},
			expected: common.MapStr{
				"ppid": false,
			},
			err: errors.New("error applying add_process_metadata processor: cannot parse field 'ppid' (not an integer or string)"),
		},
		{
			description: "process not found",
			config: common.MapStr{
				"match_pids": []string{"ppid"},
			},
			event: common.MapStr{
				"ppid": 42,
			},
			expected: common.MapStr{
				"ppid": 42,
			},
			err: ErrNoProcess,
		},
		{
			description: "lookup first PID",
			config: common.MapStr{
				"match_pids": []string{"nil", "ppid"},
			},
			event: common.MapStr{
				"nil":  0,
				"ppid": 1,
			},
			expected: common.MapStr{
				"nil":  0,
				"ppid": 1,
			},
			err: ErrNoProcess,
		},
	} {
		t.Run(test.description, func(t *testing.T) {
			config, err := common.NewConfigFrom(test.config)
			if err != nil {
				t.Fatal(err)
			}
			proc, err := newProcessMetadataProcessorWithProvider(config, testProcs)
			if test.initErr == nil {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assert.EqualError(t, err, test.initErr.Error())
				return
			}
			t.Log(proc.String())
			ev := beat.Event{
				Fields: test.event,
			}
			result, err := proc.Run(&ev)
			if test.err == nil {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assert.EqualError(t, err, test.err.Error())
			}
			if test.expected != nil {
				assert.Equal(t, test.expected, result.Fields)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestSelf(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(processorName))
	config, err := common.NewConfigFrom(common.MapStr{
		"match_pids": []string{"self_pid"},
		"target":     "self",
	})
	if err != nil {
		t.Fatal(err)
	}
	proc, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	selfPID := os.Getpid()
	ev := beat.Event{
		Fields: common.MapStr{
			"self_pid": selfPID,
		},
	}
	result, err := proc.Run(&ev)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result.Fields)
	pidField, err := result.Fields.GetValue("self.process.pid")
	if err != nil {
		t.Fatal(err)
	}
	pid, ok := pidField.(int)
	assert.True(t, ok)
	assert.Equal(t, selfPID, pid)
}

func TestBadProcess(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(processorName))
	config, err := common.NewConfigFrom(common.MapStr{
		"match_pids": []string{"self_pid"},
		"target":     "self",
	})
	if err != nil {
		t.Fatal(err)
	}
	proc, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	ev := beat.Event{
		Fields: common.MapStr{
			"self_pid": 0,
		},
	}
	result, err := proc.Run(&ev)
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Fields)
	assert.Equal(t, ev.Fields, result.Fields)
}
