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
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_process_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/convert"
	_ "github.com/elastic/beats/v7/libbeat/processors/decode_csv_fields"
	_ "github.com/elastic/beats/v7/libbeat/processors/dissect"
	_ "github.com/elastic/beats/v7/libbeat/processors/extract_array"
	_ "github.com/elastic/beats/v7/libbeat/processors/urldecode"
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

	processedEvent, err := processors.Run(event)
	if err != nil {
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

	assert.Equal(t, expectedEvent, processedEvent.Fields)
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

	processedEvent, _ := processors.Run(event)

	expectedEvent := mapstr.M{
		"type": "process",
	}

	assert.Equal(t, expectedEvent, processedEvent.Fields)
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
				"fields": []string{"proc.cpu.start_time", "mem", "proc.cmdline", "beat", "dd"},
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

	processedEvent, _ := processors.Run(event)

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

	assert.Equal(t, expectedEvent, processedEvent.Fields)
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
			"@timestamp": "2016-01-24T18:35:19.308Z",
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

	actual1, _ := processors.Run(event1)
	actual2, _ := processors.Run(event2)

	assert.Equal(t, expected1, actual1.Fields)
	assert.Equal(t, expected2, actual2.Fields)
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

	processedEvent, _ := processors.Run(event)

	assert.Nil(t, processedEvent)
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

	processedEvent, _ := processors.Run(event)

	assert.Nil(t, processedEvent)
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
				"fields": []string{"foo.bar", "proc.cpu", "proc.sss", "beat", "mem"},
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

	processedEvent, _ := processors.Run(event)

	expectedEvent := mapstr.M{
		"proc": mapstr.M{
			"cmdline": "/sbin/launchd",
		},
		"type": "process",
	}

	assert.Equal(t, expectedEvent, processedEvent.Fields)
}

const (
	fieldCount = 20
	depth      = 3
)

func BenchmarkEventBackups(b *testing.B) {
	// listing all the processors that revert changes in case of an error
	yml := []map[string]interface{}{
		{
			"append": map[string]interface{}{
				"target_field":  "append_target",
				"values":        []interface{}{"third", "fourth"},
				"fail_on_error": true,
			},
		},
		{
			"copy_fields": map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"from": "copy_from",
						"to":   "copy.to",
					},
				},
				"fail_on_error": true,
			},
		},
		{
			"decode_base64_field": map[string]interface{}{
				"field": map[string]interface{}{
					"from": "base64_from",
					"to":   "base64_to",
				},
				"fail_on_error": true,
			},
		},
		{
			"decompress_gzip_field": map[string]interface{}{
				"field": map[string]interface{}{
					"from": "gzip_from",
					"to":   "gzip_to",
				},
				"fail_on_error": true,
			},
		},
		{
			"rename": map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"from": "rename_from",
						"to":   "rename.to",
					},
				},
				"fail_on_error": true,
			},
		},
		{
			"replace": map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"field":       "replace_test",
						"pattern":     "to replace",
						"replacement": "replaced",
					},
				},
				"fail_on_error": true,
			},
		},
		{
			"truncate_fields": map[string]interface{}{
				"fields":         []interface{}{"to_truncate"},
				"max_characters": 4,
				"fail_on_error":  true,
			},
		},
		{
			"convert": map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"from": "convert_from",
						"to":   "convert.to",
						"type": "integer",
					},
				},
				"fail_on_error": true,
			},
		},
		{
			"decode_csv_fields": map[string]interface{}{
				"fields": map[string]interface{}{
					"csv_from": "csv.to",
				},
				"fail_on_error": true,
			},
		},
		// it creates a backup unless `ignore_failure` is true
		{
			"dissect": map[string]interface{}{
				"tokenizer": "%{key1} %{key2}",
				"field":     "to_dissect",
			},
		},
		{
			"extract_array": map[string]interface{}{
				"field": "array_test",
				"mappings": map[string]interface{}{
					"array_first":  0,
					"array_second": 1,
				},
				"fail_on_error": true,
			},
		},
		{
			"urldecode": map[string]interface{}{
				"fields": []map[string]interface{}{
					{
						"from": "url_from",
						"to":   "url.to",
					},
				},

				"fail_on_error": true,
			},
		},
	}

	processors := GetProcessors(b, yml)
	event := &beat.Event{
		Timestamp: time.Now(),
		Meta:      mapstr.M{},
		Fields: mapstr.M{
			"append_target": []interface{}{"first", "second"},
			"copy_from":     "to_copy",
			"base64_from":   "dmFsdWU=",
			// "decompressed data"
			"gzip_from":    string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0}),
			"rename_from":  "renamed_value",
			"replace_test": "something to replace",
			"to_truncate":  "something very long",
			"convert_from": "42",
			"csv_from":     "1,2,3,4",
			"to_dissect":   "some words",
			"array_test":   []string{"first", "second"},
			"url_from":     "https%3A%2F%2Fwww.elastic.co%3Fsome",
		},
	}

	expFields := mapstr.M{
		"append_target": []interface{}{"first", "second", "third", "fourth"},
		"copy_from":     "to_copy",
		"copy": mapstr.M{
			"to": "to_copy",
		},
		"base64_from":  "dmFsdWU=",
		"base64_to":    "value",
		"gzip_from":    string([]byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 74, 73, 77, 206, 207, 45, 40, 74, 45, 46, 78, 77, 81, 72, 73, 44, 73, 4, 4, 0, 0, 255, 255, 108, 158, 105, 19, 17, 0, 0, 0}),
		"gzip_to":      "decompressed data",
		"rename":       mapstr.M{"to": "renamed_value"},
		"replace_test": "something replaced",
		"to_truncate":  "some",
		"convert_from": "42",
		"convert":      mapstr.M{"to": int32(42)},
		"csv_from":     "1,2,3,4",
		"csv":          mapstr.M{"to": []string{"1", "2", "3", "4"}},
		"to_dissect":   "some words",
		"dissect": mapstr.M{
			"key1": "some",
			"key2": "words",
		},
		"array_test":   []string{"first", "second"},
		"array_first":  "first",
		"array_second": "second",
		"url_from":     "https%3A%2F%2Fwww.elastic.co%3Fsome",
		"url":          mapstr.M{"to": "https://www.elastic.co?some"},
	}

	generateFields(b, event.Meta, fieldCount, depth)
	generateFields(b, event.Fields, fieldCount, depth)

	var (
		result *beat.Event
		clone  *beat.Event
		err    error
	)

	b.Run("run processors that use backups", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			clone = event.Clone() // necessary for making and comparing changes
			result, err = processors.Run(clone)
		}
		require.NoError(b, err)
		require.NotNil(b, result)
	})

	require.Equal(b, fmt.Sprintf("%p", clone), fmt.Sprintf("%p", result), "should be the same event")
	for key := range expFields {
		require.Equal(b, expFields[key], clone.Fields[key], fmt.Sprintf("%s does not match", key))
	}
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
