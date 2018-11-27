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

package fields

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type FieldsGeneratorTestCase struct {
	patterns []string
	fields   []*fieldYml
}

type RemoveProcessorTestCase struct {
	processor map[string]interface{}
	fields    []string
}

func TestFieldsGenerator(t *testing.T) {
	tests := []FieldsGeneratorTestCase{
		FieldsGeneratorTestCase{
			patterns: []string{
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] %{USERNAME:postgresql.log.user}@%{HOSTNAME:postgresql.log.database} %{WORD:postgresql.log.level}:  duration: %{NUMBER:postgresql.log.duration} ms  statement: %{MULTILINEQUERY:postgresql.log.query}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] \\[%{USERNAME:postgresql.log.user}\\]@\\[%{HOSTNAME:postgresql.log.database}\\] %{WORD:postgresql.log.level}:  duration: %{NUMBER:postgresql.log.duration} ms  statement: %{MULTILINEQUERY:postgresql.log.query}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] %{USERNAME:postgresql.log.user}@%{HOSTNAME:postgresql.log.database} %{WORD:postgresql.log.level}:  ?%{GREEDYDATA:postgresql.log.message}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] \\[%{USERNAME:postgresql.log.user}\\]@\\[%{HOSTNAME:postgresql.log.database}\\] %{WORD:postgresql.log.level}:  ?%{GREEDYDATA:postgresql.log.message}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] %{WORD:postgresql.log.level}:  ?%{GREEDYDATA:postgresql.log.message}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "log", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
					&fieldYml{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
					&fieldYml{Name: "timezone", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "thread_id", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "user", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "database", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "level", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "duration", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "query", Description: "Please add description", Example: "Please add example", Type: "text"},
					&fieldYml{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
				},
				},
			},
		},
		FieldsGeneratorTestCase{
			patterns: []string{
				"%{DATA:nginx.error.time} \\[%{DATA:nginx.error.level}\\] %{NUMBER:nginx.error.pid}#%{NUMBER:nginx.error.tid}: (\\*%{NUMBER:nginx.error.connection_id} )?%{GREEDYDATA:nginx.error.message}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "error", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
					&fieldYml{Name: "time", Description: "Please add description", Example: "Please add example", Type: "text"},
					&fieldYml{Name: "level", Description: "Please add description", Example: "Please add example", Type: "text"},
					&fieldYml{Name: "pid", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "tid", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "connection_id", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
				},
				},
			},
		},
		FieldsGeneratorTestCase{
			patterns: []string{
				"\\[%{TIMESTAMP:icinga.main.timestamp}\\] %{WORD:icinga.main.severity}/%{WORD:icinga.main.facility}: %{GREEDYMULTILINE:icinga.main.message}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "main", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
					&fieldYml{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
					&fieldYml{Name: "severity", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "facility", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
				},
				},
			},
		},
		FieldsGeneratorTestCase{
			patterns: []string{
				"(%{POSINT:redis.log.pid}:%{CHAR:redis.log.role} )?%{REDISTIMESTAMP:redis.log.timestamp} %{REDISLEVEL:redis.log.level} %{GREEDYDATA:redis.log.message}",
				"%{POSINT:redis.log.pid}:signal-handler \\(%{POSINT:redis.log.timestamp}\\) %{GREEDYDATA:redis.log.message}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "log", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
					&fieldYml{Name: "pid", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "role", Description: "Please add description", Example: "Please add example"},
					&fieldYml{Name: "timestamp", Description: "Please add description", Example: "Please add example"},
					&fieldYml{Name: "level", Description: "Please add description", Example: "Please add example"},
					&fieldYml{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
				},
				},
			},
		},
		FieldsGeneratorTestCase{
			patterns: []string{
				"\\[%{TIMESTAMP:timestamp}\\] %{WORD:severity}/%{WORD:facility}: %{GREEDYMULTILINE:message}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
				&fieldYml{Name: "severity", Description: "Please add description", Example: "Please add example", Type: "keyword"},
				&fieldYml{Name: "facility", Description: "Please add description", Example: "Please add example", Type: "keyword"},
				&fieldYml{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
			},
		},
		FieldsGeneratorTestCase{
			patterns: []string{
				"\\[%{TIMESTAMP:timestamp}\\] %{WORD:severity}/%{WORD}: %{GREEDYMULTILINE:message}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
				&fieldYml{Name: "severity", Description: "Please add description", Example: "Please add example", Type: "keyword"},
				&fieldYml{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
			},
		},
		FieldsGeneratorTestCase{
			patterns: []string{
				"\\[%{TIMESTAMP:timestamp}\\] %{NUMBER:idx:int}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
				&fieldYml{Name: "idx", Description: "Please add description", Example: "Please add example", Type: "int"},
			},
		},
	}

	for _, tc := range tests {
		var proc processors
		proc.patterns = tc.patterns
		fs, err := proc.processFields()
		if err != nil {
			t.Error(err)
			return
		}

		f := generateFields(fs, false)
		assert.True(t, reflect.DeepEqual(f, tc.fields))
	}
}

// Known limitations
func TestFieldsGeneratorKnownLimitations(t *testing.T) {
	tests := []FieldsGeneratorTestCase{
		// FIXME Field names including dots are not parsed properly
		FieldsGeneratorTestCase{
			patterns: []string{
				"^# User@Host: %{USER:mysql.slowlog.user}(\\[[^\\]]+\\])? @ %{HOSTNAME:mysql.slowlog.host} \\[(%{IP:mysql.slowlog.ip})?\\](\\s*Id:\\s* %{NUMBER:mysql.slowlog.id})?\n# Query_time: %{NUMBER:mysql.slowlog.query_time.sec}\\s* Lock_time: %{NUMBER:mysql.slowlog.lock_time.sec}\\s* Rows_sent: %{NUMBER:mysql.slowlog.rows_sent}\\s* Rows_examined: %{NUMBER:mysql.slowlog.rows_examined}\n(SET timestamp=%{NUMBER:mysql.slowlog.timestamp};\n)?%{GREEDYMULTILINE:mysql.slowlog.query}",
			},
			fields: []*fieldYml{
				&fieldYml{Name: "slowlog", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
					&fieldYml{Name: "user", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "host", Description: "Please add description", Example: "Please add example", Type: "keyword"},
					&fieldYml{Name: "ip", Description: "Please add description", Example: "Please add example"},
					&fieldYml{Name: "id", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "query_time.ms", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "lock_time.ms", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "rows_sent", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "rows_examined", Description: "Please add description", Example: "Please add example", Type: "long"},
					&fieldYml{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
					&fieldYml{Name: "query", Description: "Please add description", Example: "Please add example", Type: "text"},
				},
				},
			},
		},
	}

	for _, tc := range tests {
		var proc processors
		proc.patterns = tc.patterns
		fs, err := proc.processFields()
		if err != nil {
			t.Error(err)
			return
		}

		f := generateFields(fs, false)
		assert.False(t, reflect.DeepEqual(f, tc.fields))
	}
}

func TestRemoveProcessor(t *testing.T) {
	tests := []RemoveProcessorTestCase{
		RemoveProcessorTestCase{
			processor: map[string]interface{}{
				"field": []string{},
			},
			fields: []string{},
		},
		RemoveProcessorTestCase{
			processor: map[string]interface{}{
				"field": []interface{}{},
			},
			fields: []string{},
		},
		RemoveProcessorTestCase{
			processor: map[string]interface{}{
				"field": "prospector.type",
			},
			fields: []string{"prospector.type"},
		},
		RemoveProcessorTestCase{
			processor: map[string]interface{}{
				"field": []string{"prospector.type", "input.type"},
			},
			fields: []string{"prospector.type", "input.type"},
		},
	}

	for _, tc := range tests {
		out := []string{}
		res := accumulateRemoveFields(tc.processor, out)
		assert.True(t, reflect.DeepEqual(res, tc.fields))
	}
}
