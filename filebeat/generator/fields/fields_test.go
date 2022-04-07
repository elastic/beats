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
		{
			patterns: []string{
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] %{USERNAME:postgresql.log.user}@%{HOSTNAME:postgresql.log.database} %{WORD:postgresql.log.level}:  duration: %{NUMBER:postgresql.log.duration} ms  statement: %{MULTILINEQUERY:postgresql.log.query}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] \\[%{USERNAME:postgresql.log.user}\\]@\\[%{HOSTNAME:postgresql.log.database}\\] %{WORD:postgresql.log.level}:  duration: %{NUMBER:postgresql.log.duration} ms  statement: %{MULTILINEQUERY:postgresql.log.query}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] %{USERNAME:postgresql.log.user}@%{HOSTNAME:postgresql.log.database} %{WORD:postgresql.log.level}:  ?%{GREEDYDATA:postgresql.log.message}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] \\[%{USERNAME:postgresql.log.user}\\]@\\[%{HOSTNAME:postgresql.log.database}\\] %{WORD:postgresql.log.level}:  ?%{GREEDYDATA:postgresql.log.message}",
				"%{LOCALDATETIME:postgresql.log.timestamp} %{WORD:postgresql.log.timezone} \\[%{NUMBER:postgresql.log.thread_id}\\] %{WORD:postgresql.log.level}:  ?%{GREEDYDATA:postgresql.log.message}",
			},
			fields: []*fieldYml{
				{
					Name: "log", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
						{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
						{Name: "timezone", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "thread_id", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "user", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "database", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "level", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "duration", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "query", Description: "Please add description", Example: "Please add example", Type: "text"},
						{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
					},
				},
			},
		},
		{
			patterns: []string{
				"%{DATA:nginx.error.time} \\[%{DATA:nginx.error.level}\\] %{NUMBER:nginx.error.pid}#%{NUMBER:nginx.error.tid}: (\\*%{NUMBER:nginx.error.connection_id} )?%{GREEDYDATA:nginx.error.message}",
			},
			fields: []*fieldYml{
				{
					Name: "error", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
						{Name: "time", Description: "Please add description", Example: "Please add example", Type: "text"},
						{Name: "level", Description: "Please add description", Example: "Please add example", Type: "text"},
						{Name: "pid", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "tid", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "connection_id", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
					},
				},
			},
		},
		{
			patterns: []string{
				"\\[%{TIMESTAMP:icinga.main.timestamp}\\] %{WORD:icinga.main.severity}/%{WORD:icinga.main.facility}: %{GREEDYMULTILINE:icinga.main.message}",
			},
			fields: []*fieldYml{
				{
					Name: "main", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
						{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
						{Name: "severity", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "facility", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
					},
				},
			},
		},
		{
			patterns: []string{
				"(%{POSINT:redis.log.pid}:%{CHAR:redis.log.role} )?%{REDISTIMESTAMP:redis.log.timestamp} %{REDISLEVEL:redis.log.level} %{GREEDYDATA:redis.log.message}",
				"%{POSINT:redis.log.pid}:signal-handler \\(%{POSINT:redis.log.timestamp}\\) %{GREEDYDATA:redis.log.message}",
			},
			fields: []*fieldYml{
				{
					Name: "log", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
						{Name: "pid", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "role", Description: "Please add description", Example: "Please add example"},
						{Name: "timestamp", Description: "Please add description", Example: "Please add example"},
						{Name: "level", Description: "Please add description", Example: "Please add example"},
						{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
					},
				},
			},
		},
		{
			patterns: []string{
				"\\[%{TIMESTAMP:timestamp}\\] %{WORD:severity}/%{WORD:facility}: %{GREEDYMULTILINE:message}",
			},
			fields: []*fieldYml{
				{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
				{Name: "severity", Description: "Please add description", Example: "Please add example", Type: "keyword"},
				{Name: "facility", Description: "Please add description", Example: "Please add example", Type: "keyword"},
				{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
			},
		},
		{
			patterns: []string{
				"\\[%{TIMESTAMP:timestamp}\\] %{WORD:severity}/%{WORD}: %{GREEDYMULTILINE:message}",
			},
			fields: []*fieldYml{
				{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
				{Name: "severity", Description: "Please add description", Example: "Please add example", Type: "keyword"},
				{Name: "message", Description: "Please add description", Example: "Please add example", Type: "text"},
			},
		},
		{
			patterns: []string{
				"\\[%{TIMESTAMP:timestamp}\\] %{NUMBER:idx:int}",
			},
			fields: []*fieldYml{
				{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
				{Name: "idx", Description: "Please add description", Example: "Please add example", Type: "int"},
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
		{
			patterns: []string{
				"^# User@Host: %{USER:mysql.slowlog.user}(\\[[^\\]]+\\])? @ %{HOSTNAME:mysql.slowlog.host} \\[(%{IP:mysql.slowlog.ip})?\\](\\s*Id:\\s* %{NUMBER:mysql.slowlog.id})?\n# Query_time: %{NUMBER:mysql.slowlog.query_time.sec}\\s* Lock_time: %{NUMBER:mysql.slowlog.lock_time.sec}\\s* Rows_sent: %{NUMBER:mysql.slowlog.rows_sent}\\s* Rows_examined: %{NUMBER:mysql.slowlog.rows_examined}\n(SET timestamp=%{NUMBER:mysql.slowlog.timestamp};\n)?%{GREEDYMULTILINE:mysql.slowlog.query}",
			},
			fields: []*fieldYml{
				{
					Name: "slowlog", Description: "Please add description", Example: "Please add example", Type: "group", Fields: []*fieldYml{
						{Name: "user", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "host", Description: "Please add description", Example: "Please add example", Type: "keyword"},
						{Name: "ip", Description: "Please add description", Example: "Please add example"},
						{Name: "id", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "query_time.ms", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "lock_time.ms", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "rows_sent", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "rows_examined", Description: "Please add description", Example: "Please add example", Type: "long"},
						{Name: "timestamp", Description: "Please add description", Example: "Please add example", Type: "text"},
						{Name: "query", Description: "Please add description", Example: "Please add example", Type: "text"},
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
		{
			processor: map[string]interface{}{
				"field": []string{},
			},
			fields: []string{},
		},
		{
			processor: map[string]interface{}{
				"field": []interface{}{},
			},
			fields: []string{},
		},
		{
			processor: map[string]interface{}{
				"field": "prospector.type",
			},
			fields: []string{"prospector.type"},
		},
		{
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
