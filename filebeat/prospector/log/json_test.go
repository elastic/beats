package log

import (
	"testing"
	"time"

	"github.com/elastic/beats/filebeat/harvester/reader"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestAddJSONFields(t *testing.T) {
	type io struct {
		Data          common.MapStr
		Text          *string
		JSONConfig    reader.JSONConfig
		ExpectedItems common.MapStr
	}

	text := "hello"

	now := time.Now()

	tests := []io{
		{
			// by default, don't overwrite keys
			Data:       common.MapStr{"type": "test_type", "json": common.MapStr{"type": "test", "text": "hello"}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true},
			ExpectedItems: common.MapStr{
				"type": "test_type",
				"text": "hello",
			},
		},
		{
			// overwrite keys if asked
			Data:       common.MapStr{"type": "test_type", "json": common.MapStr{"type": "test", "text": "hello"}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: common.MapStr{
				"type": "test",
				"text": "hello",
			},
		},
		{
			// without keys_under_root, put everything in a json key
			Data:       common.MapStr{"type": "test_type", "json": common.MapStr{"type": "test", "text": "hello"}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{},
			ExpectedItems: common.MapStr{
				"json": common.MapStr{"type": "test", "text": "hello"},
			},
		},
		{
			// when MessageKey is defined, the Text overwrites the value of that key
			Data:       common.MapStr{"type": "test_type", "json": common.MapStr{"type": "test", "text": "hi"}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{MessageKey: "text"},
			ExpectedItems: common.MapStr{
				"json": common.MapStr{"type": "test", "text": "hello"},
				"type": "test_type",
			},
		},
		{
			// when @timestamp is in JSON and overwrite_keys is true, parse it
			// in a common.Time
			Data:       common.MapStr{"@timestamp": now, "type": "test_type", "json": common.MapStr{"type": "test", "@timestamp": "2016-04-05T18:47:18.444Z"}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: common.MapStr{
				"@timestamp": common.MustParseTime("2016-04-05T18:47:18.444Z"),
				"type":       "test",
			},
		},
		{
			// when the parsing on @timestamp fails, leave the existing value and add an error key
			// in a common.Time
			Data:       common.MapStr{"@timestamp": common.Time(now), "type": "test_type", "json": common.MapStr{"type": "test", "@timestamp": "2016-04-05T18:47:18.44XX4Z"}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: common.MapStr{
				"@timestamp": common.Time(now),
				"type":       "test",
				"error":      common.MapStr{"type": "json", "message": "@timestamp not overwritten (parse error on 2016-04-05T18:47:18.44XX4Z)"},
			},
		},
		{
			// when the @timestamp has the wrong type, leave the existing value and add an error key
			// in a common.Time
			Data:       common.MapStr{"@timestamp": common.Time(now), "type": "test_type", "json": common.MapStr{"type": "test", "@timestamp": 42}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: common.MapStr{
				"@timestamp": common.Time(now),
				"type":       "test",
				"error":      common.MapStr{"type": "json", "message": "@timestamp not overwritten (not string)"},
			},
		},
		{
			// if overwrite_keys is true, but the `type` key in json is not a string, ignore it
			Data:       common.MapStr{"type": "test_type", "json": common.MapStr{"type": 42}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: common.MapStr{
				"type":  "test_type",
				"error": common.MapStr{"type": "json", "message": "type not overwritten (not string)"},
			},
		},
		{
			// if overwrite_keys is true, but the `type` key in json is empty, ignore it
			Data:       common.MapStr{"type": "test_type", "json": common.MapStr{"type": ""}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: common.MapStr{
				"type":  "test_type",
				"error": common.MapStr{"type": "json", "message": "type not overwritten (invalid value [])"},
			},
		},
		{
			// if overwrite_keys is true, but the `type` key in json starts with _, ignore it
			Data:       common.MapStr{"@timestamp": common.Time(now), "type": "test_type", "json": common.MapStr{"type": "_type"}},
			Text:       &text,
			JSONConfig: reader.JSONConfig{KeysUnderRoot: true, OverwriteKeys: true},
			ExpectedItems: common.MapStr{
				"type":  "test_type",
				"error": common.MapStr{"type": "json", "message": "type not overwritten (invalid value [_type])"},
			},
		},
	}

	for _, test := range tests {
		h := Harvester{}
		h.config.JSON = &test.JSONConfig

		var jsonFields common.MapStr
		if fields, ok := test.Data["json"]; ok {
			jsonFields = fields.(common.MapStr)
		}

		h.mergeJSONFields(test.Data, jsonFields, test.Text)

		t.Log("Executing test:", test)
		for k, v := range test.ExpectedItems {
			assert.Equal(t, v, test.Data[k])
		}
	}
}
