package input

import (
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestEventToMapStr(t *testing.T) {
	// Test 'fields' is not present when it is nil.
	event := Event{}
	mapStr := event.ToMapStr()
	_, found := mapStr["fields"]
	assert.False(t, found)
}

func TestEventToMapStrJSON(t *testing.T) {
	type io struct {
		Event         Event
		ExpectedItems common.MapStr
	}

	text := "hello"

	now := time.Now()

	tests := []io{
		{
			// by default, don't overwrite keys
			Event: Event{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hello"},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true},
			},
			ExpectedItems: common.MapStr{
				"type": "test_type",
				"text": "hello",
			},
		},
		{
			// overwrite keys if asked
			Event: Event{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hello"},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"type": "test",
				"text": "hello",
			},
		},
		{
			// without keys_under_root, put everything in a json key
			Event: Event{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hello"},
				JSONConfig:   &reader.JSONConfig{},
			},
			ExpectedItems: common.MapStr{
				"json": common.MapStr{"type": "test", "text": "hello"},
				"type": "test_type",
			},
		},
		{
			// when MessageKey is defined, the Text overwrites the value of that key
			Event: Event{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "text": "hi"},
				JSONConfig:   &reader.JSONConfig{MessageKey: "text"},
			},
			ExpectedItems: common.MapStr{
				"json": common.MapStr{"type": "test", "text": "hello"},
				"type": "test_type",
			},
		},
		{
			// when @timestamp is in JSON and overwrite_keys is true, parse it
			// in a common.Time
			Event: Event{
				ReadTime:     now,
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "@timestamp": "2016-04-05T18:47:18.444Z"},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"@timestamp": common.MustParseTime("2016-04-05T18:47:18.444Z"),
				"type":       "test",
			},
		},
		{
			// when the parsing on @timestamp fails, leave the existing value and add an error key
			// in a common.Time
			Event: Event{
				ReadTime:     now,
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "@timestamp": "2016-04-05T18:47:18.44XX4Z"},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"@timestamp": common.Time(now),
				"type":       "test",
				"json_error": "@timestamp not overwritten (parse error on 2016-04-05T18:47:18.44XX4Z)",
			},
		},
		{
			// when the @timestamp has the wrong type, leave the existing value and add an error key
			// in a common.Time
			Event: Event{
				ReadTime:     now,
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "test", "@timestamp": 42},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"@timestamp": common.Time(now),
				"type":       "test",
				"json_error": "@timestamp not overwritten (not string)",
			},
		},
		{
			// if overwrite_keys is true, but the `type` key in json is not a string, ignore it
			Event: Event{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": 42},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"type":       "test_type",
				"json_error": "type not overwritten (not string)",
			},
		},
		{
			// if overwrite_keys is true, but the `type` key in json is empty, ignore it
			Event: Event{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": ""},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"type":       "test_type",
				"json_error": "type not overwritten (invalid value [])",
			},
		},
		{
			// if overwrite_keys is true, but the `type` key in json starts with _, ignore it
			Event: Event{
				DocumentType: "test_type",
				Text:         &text,
				JSONFields:   common.MapStr{"type": "_type"},
				JSONConfig:   &reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			},
			ExpectedItems: common.MapStr{
				"type":       "test_type",
				"json_error": "type not overwritten (invalid value [_type])",
			},
		},
	}

	for _, test := range tests {
		result := test.Event.ToMapStr()
		t.Log("Executing test:", test)
		for k, v := range test.ExpectedItems {
			assert.Equal(t, v, result[k])
		}
	}
}
