package dissect

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestProcessor(t *testing.T) {
	tests := []struct {
		name   string
		c      map[string]interface{}
		fields common.MapStr
		values map[string]string
	}{
		{
			name:   "default field/default target",
			c:      map[string]interface{}{"tokenizer": "hello %{key}"},
			fields: common.MapStr{"message": "hello world"},
			values: map[string]string{"dissect.key": "world"},
		},
		{
			name:   "default field/target root",
			c:      map[string]interface{}{"tokenizer": "hello %{key}", "target_prefix": ""},
			fields: common.MapStr{"message": "hello world"},
			values: map[string]string{"key": "world"},
		},
		{
			name: "specific field/target root",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key}",
				"target_prefix": "",
				"field":         "new_field",
			},
			fields: common.MapStr{"new_field": "hello world"},
			values: map[string]string{"key": "world"},
		},
		{
			name: "specific field/specific target",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key}",
				"target_prefix": "new_target",
				"field":         "new_field",
			},
			fields: common.MapStr{"new_field": "hello world"},
			values: map[string]string{"new_target.key": "world"},
		},
		{
			name: "extract to already existing namespace not conflicting",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key} %{key2}",
				"target_prefix": "extracted",
				"field":         "message",
			},
			fields: common.MapStr{"message": "hello world super", "extracted": common.MapStr{"not": "hello"}},
			values: map[string]string{"extracted.key": "world", "extracted.key2": "super", "extracted.not": "hello"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := common.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := newProcessor(c)
			if !assert.NoError(t, err) {
				return
			}

			e := beat.Event{Fields: test.fields}
			newEvent, err := processor.Run(&e)
			if !assert.NoError(t, err) {
				return
			}

			for field, value := range test.values {
				v, err := newEvent.GetValue(field)
				if !assert.NoError(t, err) {
					return
				}

				assert.Equal(t, value, v)
			}
		})
	}
}

func TestFieldDoesntExist(t *testing.T) {
	c, err := common.NewConfigFrom(map[string]interface{}{"tokenizer": "hello %{key}"})
	if !assert.NoError(t, err) {
		return
	}

	processor, err := newProcessor(c)
	if !assert.NoError(t, err) {
		return
	}

	e := beat.Event{Fields: common.MapStr{"hello": "world"}}
	_, err = processor.Run(&e)
	if !assert.Error(t, err) {
		return
	}
}

func TestFieldAlreadyExist(t *testing.T) {
	tests := []struct {
		name      string
		tokenizer string
		prefix    string
		fields    common.MapStr
	}{
		{
			name:      "no prefix",
			tokenizer: "hello %{key}",
			prefix:    "",
			fields:    common.MapStr{"message": "hello world", "key": "exists"},
		},
		{
			name:      "with prefix",
			tokenizer: "hello %{key}",
			prefix:    "extracted",
			fields:    common.MapStr{"message": "hello world", "extracted": "exists"},
		},
		{
			name:      "with conflicting key in prefix",
			tokenizer: "hello %{key}",
			prefix:    "extracted",
			fields:    common.MapStr{"message": "hello world", "extracted": common.MapStr{"key": "exists"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := common.NewConfigFrom(map[string]interface{}{
				"tokenizer":     test.tokenizer,
				"target_prefix": test.prefix,
			})

			if !assert.NoError(t, err) {
				return
			}

			processor, err := newProcessor(c)
			if !assert.NoError(t, err) {
				return
			}

			e := beat.Event{Fields: test.fields}
			_, err = processor.Run(&e)
			if !assert.Error(t, err) {
				return
			}
		})
	}
}
