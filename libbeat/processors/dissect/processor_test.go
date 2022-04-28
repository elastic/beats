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

package dissect

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestProcessor(t *testing.T) {
	tests := []struct {
		name   string
		c      map[string]interface{}
		fields mapstr.M
		values map[string]string
	}{
		{
			name:   "default field/default target",
			c:      map[string]interface{}{"tokenizer": "hello %{key}"},
			fields: mapstr.M{"message": "hello world"},
			values: map[string]string{"dissect.key": "world"},
		},
		{
			name:   "default field/target root",
			c:      map[string]interface{}{"tokenizer": "hello %{key}", "target_prefix": ""},
			fields: mapstr.M{"message": "hello world"},
			values: map[string]string{"key": "world"},
		},
		{
			name: "specific field/target root",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key}",
				"target_prefix": "",
				"field":         "new_field",
			},
			fields: mapstr.M{"new_field": "hello world"},
			values: map[string]string{"key": "world"},
		},
		{
			name: "specific field/specific target",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key}",
				"target_prefix": "new_target",
				"field":         "new_field",
			},
			fields: mapstr.M{"new_field": "hello world"},
			values: map[string]string{"new_target.key": "world"},
		},
		{
			name: "extract to already existing namespace not conflicting",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key} %{key2}",
				"target_prefix": "extracted",
				"field":         "message",
			},
			fields: mapstr.M{"message": "hello world super", "extracted": mapstr.M{"not": "hello"}},
			values: map[string]string{"extracted.key": "world", "extracted.key2": "super", "extracted.not": "hello"},
		},
		{
			name: "trimming trailing spaces",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key} %{key2}",
				"target_prefix": "",
				"field":         "message",
				"trim_values":   "right",
				"trim_chars":    " \t",
			},
			fields: mapstr.M{"message": "hello world\t super "},
			values: map[string]string{"key": "world", "key2": "super"},
		},
		{
			name: "not trimming by default",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key} %{key2}",
				"target_prefix": "",
				"field":         "message",
			},
			fields: mapstr.M{"message": "hello world\t super "},
			values: map[string]string{"key": "world\t", "key2": "super "},
		},
		{
			name: "trim leading space",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key} %{key2}",
				"target_prefix": "",
				"field":         "message",
				"trim_values":   "left",
				"trim_chars":    " \t",
			},
			fields: mapstr.M{"message": "hello \tworld\t \tsuper "},
			values: map[string]string{"key": "world\t", "key2": "super "},
		},
		{
			name: "trim all space",
			c: map[string]interface{}{
				"tokenizer":     "hello %{key} %{key2}",
				"target_prefix": "",
				"field":         "message",
				"trim_values":   "all",
				"trim_chars":    " \t",
			},
			fields: mapstr.M{"message": "hello \tworld\t \tsuper "},
			values: map[string]string{"key": "world", "key2": "super"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := common.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c)
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

	t.Run("supports metadata as a target", func(t *testing.T) {
		e := &beat.Event{
			Meta: mapstr.M{
				"message": "hello world",
			},
		}
		expMeta := mapstr.M{
			"message": "hello world",
			"key":     "world",
		}

		c := map[string]interface{}{
			"tokenizer":     "hello %{key}",
			"field":         "@metadata.message",
			"target_prefix": "@metadata",
		}
		cfg, err := common.NewConfigFrom(c)
		assert.NoError(t, err)

		processor, err := NewProcessor(cfg)
		assert.NoError(t, err)

		newEvent, err := processor.Run(e)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, e.Fields, newEvent.Fields)
	})
}

func TestFieldDoesntExist(t *testing.T) {
	c, err := common.NewConfigFrom(map[string]interface{}{"tokenizer": "hello %{key}"})
	if !assert.NoError(t, err) {
		return
	}

	processor, err := NewProcessor(c)
	if !assert.NoError(t, err) {
		return
	}

	e := beat.Event{Fields: mapstr.M{"hello": "world"}}
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
		fields    mapstr.M
	}{
		{
			name:      "no prefix",
			tokenizer: "hello %{key}",
			prefix:    "",
			fields:    mapstr.M{"message": "hello world", "key": "exists"},
		},
		{
			name:      "with prefix",
			tokenizer: "hello %{key}",
			prefix:    "extracted",
			fields:    mapstr.M{"message": "hello world", "extracted": "exists"},
		},
		{
			name:      "with conflicting key in prefix",
			tokenizer: "hello %{key}",
			prefix:    "extracted",
			fields:    mapstr.M{"message": "hello world", "extracted": mapstr.M{"key": "exists"}},
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

			processor, err := NewProcessor(c)
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

func TestErrorFlagging(t *testing.T) {
	t.Run("when the parsing fails add a flag", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{ok} - %{notvalid}",
		})

		if !assert.NoError(t, err) {
			return
		}

		processor, err := NewProcessor(c)
		if !assert.NoError(t, err) {
			return
		}

		e := beat.Event{Fields: mapstr.M{"message": "hello world"}}
		event, err := processor.Run(&e)

		if !assert.Error(t, err) {
			return
		}

		flags, err := event.GetValue(beat.FlagField)
		if !assert.NoError(t, err) {
			return
		}

		assert.Contains(t, flags, flagParsingError)
	})

	t.Run("when the parsing is succesful do not add a flag", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{ok} %{valid}",
		})

		if !assert.NoError(t, err) {
			return
		}

		processor, err := NewProcessor(c)
		if !assert.NoError(t, err) {
			return
		}

		e := beat.Event{Fields: mapstr.M{"message": "hello world"}}
		event, err := processor.Run(&e)

		if !assert.NoError(t, err) {
			return
		}

		_, err = event.GetValue(beat.FlagField)
		assert.Error(t, err)
	})
}

func TestIgnoreFailure(t *testing.T) {
	tests := []struct {
		name  string
		c     map[string]interface{}
		msg   string
		err   error
		flags bool
	}{
		{
			name:  "default is to fail",
			c:     map[string]interface{}{"tokenizer": "hello %{key}"},
			msg:   "something completely different",
			err:   errors.New("could not find beginning delimiter: `hello ` in remaining: `something completely different`, (offset: 0)"),
			flags: true,
		},
		{
			name: "ignore_failure is a noop on success",
			c:    map[string]interface{}{"tokenizer": "hello %{key}", "ignore_failure": true},
			msg:  "hello world",
		},
		{
			name:  "ignore_failure hides the error but maintains flags",
			c:     map[string]interface{}{"tokenizer": "hello %{key}", "ignore_failure": true},
			msg:   "something completely different",
			flags: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := common.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c)
			if !assert.NoError(t, err) {
				return
			}

			e := beat.Event{Fields: mapstr.M{"message": test.msg}}
			event, err := processor.Run(&e)
			if test.err == nil {
				if !assert.NoError(t, err) {
					return
				}
			} else {
				if !assert.EqualError(t, err, test.err.Error()) {
					return
				}
			}
			flags, err := event.GetValue(beat.FlagField)
			if test.flags {
				if !assert.NoError(t, err) || !assert.Contains(t, flags, flagParsingError) {
					return
				}
			} else {
				if !assert.Error(t, err) {
					return
				}
			}
		})
	}
}

func TestOverwriteKeys(t *testing.T) {
	tests := []struct {
		name   string
		c      map[string]interface{}
		fields mapstr.M
		values mapstr.M
		err    error
	}{
		{
			name:   "fail by default if key exists",
			c:      map[string]interface{}{"tokenizer": "hello %{key}", "target_prefix": ""},
			fields: mapstr.M{"message": "hello world", "key": 42},
			values: mapstr.M{"message": "hello world", "key": 42},
			err:    errors.New("cannot override existing key with `key`"),
		},
		{
			name:   "fail if key exists and overwrite disabled",
			c:      map[string]interface{}{"tokenizer": "hello %{key}", "target_prefix": "", "overwrite_keys": false},
			fields: mapstr.M{"message": "hello world", "key": 42},
			values: mapstr.M{"message": "hello world", "key": 42},
			err:    errors.New("cannot override existing key with `key`"),
		},
		{
			name:   "overwrite existing keys",
			c:      map[string]interface{}{"tokenizer": "hello %{key}", "target_prefix": "", "overwrite_keys": true},
			fields: mapstr.M{"message": "hello world", "key": 42},
			values: mapstr.M{"message": "hello world", "key": "world"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := common.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c)
			if !assert.NoError(t, err) {
				return
			}

			e := beat.Event{Fields: test.fields}
			event, err := processor.Run(&e)
			if test.err == nil {
				if !assert.NoError(t, err) {
					return
				}
			} else {
				if !assert.EqualError(t, err, test.err.Error()) {
					return
				}
			}

			for field, value := range test.values {
				v, err := event.GetValue(field)
				if !assert.NoError(t, err) {
					return
				}

				assert.Equal(t, value, v)
			}
		})
	}
}

func TestProcessorConvert(t *testing.T) {
	tests := []struct {
		name   string
		c      map[string]interface{}
		fields mapstr.M
		values map[string]interface{}
	}{
		{
			name:   "extract integer",
			c:      map[string]interface{}{"tokenizer": "userid=%{user_id|integer}"},
			fields: mapstr.M{"message": "userid=7736"},
			values: map[string]interface{}{"dissect.user_id": int32(7736)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := common.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c)
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
