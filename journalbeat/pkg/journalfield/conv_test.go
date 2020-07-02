//+build linux,cgo

package journalfield

import (
	"testing"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
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
				"host": common.MapStr{
					"boot_id": "123456",
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
