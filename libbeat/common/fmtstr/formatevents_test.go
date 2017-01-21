package fmtstr

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestEventFormatString(t *testing.T) {
	tests := []struct {
		title    string
		format   string
		event    common.MapStr
		expected string
		fields   []string
	}{
		{
			"no fields configured",
			"format string",
			nil,
			"format string",
			nil,
		},
		{
			"expand event field",
			"%{[key]}",
			common.MapStr{"key": "value"},
			"value",
			[]string{"key"},
		},
		{
			"expand with default",
			"%{[key]:default}",
			common.MapStr{},
			"default",
			nil,
		},
		{
			"expand nested event field",
			"%{[nested.key]}",
			common.MapStr{"nested": common.MapStr{"key": "value"}},
			"value",
			[]string{"nested.key"},
		},
		{
			"expand nested event field (alt. syntax)",
			"%{[nested][key]}",
			common.MapStr{"nested": common.MapStr{"key": "value"}},
			"value",
			[]string{"nested.key"},
		},
		{
			"multiple event fields",
			"%{[key1]} - %{[key2]}",
			common.MapStr{"key1": "v1", "key2": "v2"},
			"v1 - v2",
			[]string{"key1", "key2"},
		},
		{
			"same fields",
			"%{[key]} - %{[key]}",
			common.MapStr{"key": "value"},
			"value - value",
			[]string{"key"},
		},
		{
			"same fields with default (first)",
			"%{[key]:default} - %{[key]}",
			common.MapStr{"key": "value"},
			"value - value",
			[]string{"key"},
		},
		{
			"same fields with default (second)",
			"%{[key]} - %{[key]:default}",
			common.MapStr{"key": "value"},
			"value - value",
			[]string{"key"},
		},
		{
			"test timestamp formatter",
			"%{[key]}: %{+YYYY.MM.dd}",
			common.MapStr{
				"@timestamp": common.Time(
					time.Date(2015, 5, 1, 20, 12, 34, 0, time.Local),
				),
				"key": "timestamp",
			},
			"timestamp: 2015.05.01",
			[]string{"key"},
		},
	}

	for i, test := range tests {
		t.Logf("test(%v): %v", i, test.title)

		fs, err := CompileEvent(test.format)
		if err != nil {
			t.Error(err)
			continue
		}

		actual, err := fs.Run(test.event)

		assert.NoError(t, err)
		assert.Equal(t, test.expected, actual)
		assert.Equal(t, test.fields, fs.Fields())
	}
}

func TestEventFormatStringErrors(t *testing.T) {
	tests := []struct {
		title          string
		format         string
		expectCompiles bool
		event          common.MapStr
	}{
		{
			"empty field",
			"%{[]}",
			false, nil,
		},
		{
			"field not closed",
			"%{[field}",
			false, nil,
		},
		{
			"no field accessor",
			"%{field}",
			false, nil,
		},
		{
			"unknown operator",
			"%{[field]:?fail}",
			false, nil,
		},
		{
			"too many operators",
			"%{[field]:a:b}",
			false, nil,
		},
		{
			"invalid timestamp formatter",
			"%{+abc}",
			false, nil,
		},
		{
			"missing required field",
			"%{[key]}",
			true,
			common.MapStr{},
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		fs, err := CompileEvent(test.format)
		if !test.expectCompiles {
			assert.Error(t, err)
			continue
		}
		if err != nil {
			t.Error(err)
			continue
		}

		_, err = fs.Run(test.event)
		assert.Error(t, err)
	}
}

func TestEventFormatStringFromConfig(t *testing.T) {
	tests := []struct {
		v        interface{}
		event    common.MapStr
		expected string
	}{
		{
			"plain string",
			common.MapStr{},
			"plain string",
		},
		{
			100,
			common.MapStr{},
			"100",
		},
		{
			true,
			common.MapStr{},
			"true",
		},
		{
			"%{[key]}",
			common.MapStr{"key": "value"},
			"value",
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v -> %v", i, test.v, test.expected)

		config, err := common.NewConfigFrom(common.MapStr{
			"test": test.v,
		})
		if err != nil {
			t.Error(err)
			continue
		}

		testConfig := struct {
			Test *EventFormatString `config:"test"`
		}{}
		err = config.Unpack(&testConfig)
		if err != nil {
			t.Error(err)
			continue
		}

		actual, err := testConfig.Test.Run(test.event)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.expected, actual)
	}
}
