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

//go:build linux && cgo
// +build linux,cgo

package journalfield

import (
	"testing"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestConversion(t *testing.T) {
	tests := map[string]struct {
		fields map[string]string
		want   common.MapStr
	}{
		"field name from fields.go": {
			fields: map[string]string{
				sdjournal.SD_JOURNAL_FIELD_BOOT_ID: "123456",
			},
			want: common.MapStr{
				"journald": common.MapStr{
					"host": common.MapStr{
						"boot_id": "123456",
					},
				},
			},
		},
		"'syslog.pid' field without user append": {
			fields: map[string]string{
				sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID: "123456",
			},
			want: common.MapStr{
				"syslog": common.MapStr{
					"pid": int64(123456),
				},
			},
		},
		"'syslog.priority' field with junk": {
			fields: map[string]string{
				sdjournal.SD_JOURNAL_FIELD_PRIORITY: "123456, ",
			},
			want: common.MapStr{
				"syslog": common.MapStr{
					"priority": int64(123456),
				},
				"log": common.MapStr{
					"syslog": common.MapStr{
						"priority": int64(123456),
					},
				},
			},
		},
		"'syslog.pid' field with user append": {
			fields: map[string]string{
				sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID: "123456,root",
			},
			want: common.MapStr{
				"syslog": common.MapStr{
					"pid": int64(123456),
				},
			},
		},
		"'syslog.pid' field empty": {
			fields: map[string]string{
				sdjournal.SD_JOURNAL_FIELD_SYSLOG_PID: "",
			},
			want: common.MapStr{
				"syslog": common.MapStr{
					"pid": "",
				},
			},
		},
		"custom field": {
			fields: map[string]string{
				"my_custom_field": "value",
			},
			want: common.MapStr{
				"journald": common.MapStr{
					"custom": common.MapStr{
						"my_custom_field": "value",
					},
				},
			},
		},
		"dropped field": {
			fields: map[string]string{
				"_SOURCE_MONOTONIC_TIMESTAMP": "value",
			},
			want: common.MapStr{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			log := logp.NewLogger("test")
			converted := NewConverter(log, nil).Convert(test.fields)
			assert.Equal(t, test.want, converted)
		})
	}
}
