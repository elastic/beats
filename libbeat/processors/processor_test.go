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

package processors_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func GetProcessors(t testing.TB, yml []map[string]interface{}) *processors.Processors {
	list, err := MakeProcessors(t, yml)
	if err != nil {
		t.Fatal(err)
	}

	return list
}

func MakeProcessors(t testing.TB, yml []map[string]interface{}) (*processors.Processors, error) {
	t.Helper()

	var config processors.PluginConfig
	for _, processor := range yml {
		processorCfg, err := conf.NewConfigFrom(processor)
		if err != nil {
			t.Fatal(err)
		}

		config = append(config, processorCfg)
	}

	return processors.New(config)
}

func TestBadConfig(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"contains": map[string]string{
						"proc.name": "test",
					},
				},
				"fields": []string{"proc.cpu.total_p", "proc.mem", "dd"},
			},
			"drop_fields": map[string]interface{}{
				"fields": []string{"proc.cpu"},
			},
		},
	}

	_, err := MakeProcessors(t, yml)
	assert.Error(t, err)
}

func TestIncludeFields(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"contains": map[string]string{
						"proc.name": "test",
					},
				},
				"fields": []string{"proc.cpu.total_p", "proc.mem", "dd"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},
			"proc": mapstr.M{
				"cpu": mapstr.M{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"name":    "test-1",
				"cmdline": "/sbin/launchd",
				"mem": mapstr.M{
					"rss":   11194368,
					"rss_p": 0,
					"share": 0,
					"size":  int64(2555572224),
				},
			},
			"type": "process",
		},
	}
	ed := beat.NewEventEditor(event)
	dropped, err := processors.Run(ed)
	if err != nil || dropped {
		t.Fatal(err)
	}

	expectedEvent := mapstr.M{
		"proc": mapstr.M{
			"cpu": mapstr.M{
				"total_p": 0,
			},
			"mem": mapstr.M{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  int64(2555572224),
			},
		},
		"type": "process",
	}
	ed.Apply()
	assert.Equal(t, expectedEvent, event.Fields)
}

func TestIncludeFields1(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"regexp": map[string]string{
						"proc.cmdline": "launchd",
					},
				},
				"fields": []string{"proc.cpu.total_add"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},

			"proc": mapstr.M{
				"cpu": mapstr.M{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"cmdline": "/sbin/launchd",
				"mem": mapstr.M{
					"rss":   11194368,
					"rss_p": 0,
					"share": 0,
					"size":  int64(2555572224),
				},
			},
			"type": "process",
		},
	}

	ed := beat.NewEventEditor(event)
	_, _ = processors.Run(ed)

	expectedEvent := mapstr.M{
		"type": "process",
	}
	ed.Apply()
	assert.Equal(t, expectedEvent, event.Fields)
}

func TestDropFields(t *testing.T) {
	yml := []map[string]interface{}{
		{
			"drop_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"equals": map[string]string{
						"beat.hostname": "mar",
					},
				},
				"fields": []string{"proc.cpu.start_time", "mem", "proc.cmdline", "beat"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},

			"proc": mapstr.M{
				"cpu": mapstr.M{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"cmdline": "/sbin/launchd",
			},
			"mem": mapstr.M{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  int64(2555572224),
			},
			"type": "process",
		},
	}

	ed := beat.NewEventEditor(event)
	dropped, err := processors.Run(ed)
	assert.NoError(t, err)
	assert.False(t, dropped)

	expectedEvent := mapstr.M{
		"proc": mapstr.M{
			"cpu": mapstr.M{
				"system":  26027,
				"total":   79390,
				"total_p": 0,
				"user":    53363,
			},
		},
		"type": "process",
	}

	assert.Equal(t, expectedEvent, event.Fields)
}

func TestMultipleIncludeFields(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"contains": map[string]string{
						"beat.name": "my-shipper",
					},
				},
				"fields": []string{"proc"},
			},
		},
		{
			"include_fields": map[string]interface{}{
				"fields": []string{"proc.cpu.start_time", "proc.cpu.total_p", "proc.mem.rss_p", "proc.cmdline"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event1 := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},

			"proc": mapstr.M{
				"cpu": mapstr.M{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"cmdline": "/sbin/launchd",
			},
			"mem": mapstr.M{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  int64(2555572224),
			},
			"type": "process",
		},
	}

	event2 := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},
			"fs": mapstr.M{
				"device_name": "devfs",
				"total":       198656,
				"used":        198656,
				"used_p":      1,
				"free":        0,
				"avail":       0,
				"files":       677,
				"free_files":  0,
				"mount_point": "/dev",
			},
			"type": "process",
		},
	}

	expected1 := mapstr.M{
		"proc": mapstr.M{
			"cpu": mapstr.M{
				"start_time": "Jan14",
				"total_p":    0,
			},
			"cmdline": "/sbin/launchd",
		},

		"type": "process",
	}

	expected2 := mapstr.M{
		"type": "process",
	}

	ed1 := beat.NewEventEditor(event1)
	ed2 := beat.NewEventEditor(event2)
	_, _ = processors.Run(ed1)
	_, _ = processors.Run(ed2)

	assert.Equal(t, expected1, event1.Fields)
	assert.Equal(t, expected2, event2.Fields)
}

func TestDropEvent(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"drop_event": map[string]interface{}{
				"when": map[string]interface{}{
					"range": map[string]interface{}{
						"proc.cpu.total_p": map[string]float64{
							"lt": 0.5,
						},
					},
				},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},
			"proc": mapstr.M{
				"cpu": mapstr.M{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"name":    "test-1",
				"cmdline": "/sbin/launchd",
				"mem": mapstr.M{
					"rss":   11194368,
					"rss_p": 0,
					"share": 0,
					"size":  int64(2555572224),
				},
			},
			"type": "process",
		},
	}

	ed := beat.NewEventEditor(event)
	dropped, _ := processors.Run(ed)

	assert.True(t, dropped)
}

func TestEmptyCondition(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"drop_event": map[string]interface{}{},
		},
	}

	processors := GetProcessors(t, yml)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},
			"proc": mapstr.M{
				"cpu": mapstr.M{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"name":    "test-1",
				"cmdline": "/sbin/launchd",
				"mem": mapstr.M{
					"rss":   11194368,
					"rss_p": 0,
					"share": 0,
					"size":  int64(2555572224),
				},
			},
			"type": "process",
		},
	}

	ed := beat.NewEventEditor(event)
	dropped, _ := processors.Run(ed)

	assert.True(t, dropped)
}

func TestBadCondition(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"drop_event": map[string]interface{}{
				"when": map[string]interface{}{
					"equal": map[string]string{
						"type": "process",
					},
				},
			},
		},
	}

	_, err := MakeProcessors(t, yml)
	assert.Error(t, err)
}

func TestMissingFields(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"equals": map[string]string{
						"type": "process",
					},
				},
			},
		},
	}

	_, err := MakeProcessors(t, yml)
	assert.Error(t, err)
}

func TestBadConditionConfig(t *testing.T) {
	logp.TestingSetup()

	yml := []map[string]interface{}{
		{
			"include_fields": map[string]interface{}{
				"when": map[string]interface{}{
					"fake": map[string]string{
						"type": "process",
					},
				},
				"fields": []string{"proc.cpu.start_time", "proc.cpu.total_p", "proc.mem.rss_p", "proc.cmdline"},
			},
		},
	}

	_, err := MakeProcessors(t, yml)
	assert.Error(t, err)
}

func TestDropMissingFields(t *testing.T) {
	yml := []map[string]interface{}{
		{
			"drop_fields": map[string]interface{}{
				"ignore_missing": true,
				"fields":         []string{"foo.bar", "proc.cpu", "proc.sss", "beat", "mem"},
			},
		},
	}

	processors := GetProcessors(t, yml)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"beat": mapstr.M{
				"hostname": "mar",
				"name":     "my-shipper-1",
			},

			"proc": mapstr.M{
				"cpu": mapstr.M{
					"start_time": "Jan14",
					"system":     26027,
					"total":      79390,
					"total_p":    0,
					"user":       53363,
				},
				"cmdline": "/sbin/launchd",
			},
			"mem": mapstr.M{
				"rss":   11194368,
				"rss_p": 0,
				"share": 0,
				"size":  int64(2555572224),
			},
			"type": "process",
		},
	}

	ed := beat.NewEventEditor(event)
	dropped, err := processors.Run(ed)
	assert.False(t, dropped)
	assert.NoError(t, err)

	expectedEvent := mapstr.M{
		"proc": mapstr.M{
			"cmdline": "/sbin/launchd",
		},
		"type": "process",
	}

	assert.Equal(t, expectedEvent, event.Fields)
}

const (
	fieldCount   = 10000
	nestingLevel = 3
)

func BenchmarkProcessorsRun(b *testing.B) {
	processors := processors.NewList(nil)
	key1 := "added.field"
	proc1 := actions.NewAddFields(mapstr.M{key1: "first"}, true)
	processors.AddProcessor(proc1)
	key2 := "field-0.field-0"
	proc2 := actions.NewAddFields(mapstr.M{key2: "second"}, true)
	processors.AddProcessor(proc2)

	event := &beat.Event{
		Timestamp: time.Now(),
		Meta:      mapstr.M{},
		Fields:    mapstr.M{},
	}

	generateFields(b, event.Meta, 100, 2)
	generateFields(b, event.Fields, 100, 2)

	ed := beat.NewEventEditor(event)

	var (
		dropped bool
		err     error
	)

	b.Run("processors.Run", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			dropped, err = processors.Run(ed)
			require.NoError(b, err)
			require.False(b, dropped)
		}
	})

	added, err := ed.GetValue(key1)
	require.NoError(b, err)
	require.Equal(b, "first", added)

	added, err = ed.GetValue(key2)
	require.NoError(b, err)
	require.Equal(b, "second", added)
}

func generateFields(t require.TestingT, m mapstr.M, count, nesting int) {
	for i := 0; i < count; i++ {
		var err error
		if nesting == 0 {
			_, err = m.Put(fmt.Sprintf("field-%d", i), fmt.Sprintf("value-%d", i))
		} else {
			nested := mapstr.M{}
			generateFields(t, nested, count, nesting-1)
			_, err = m.Put(fmt.Sprintf("field-%d", i), nested)
		}
		require.NoError(t, err)
	}
}
