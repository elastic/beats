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
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
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
			c, err := conf.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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
		cfg, err := conf.NewConfigFrom(c)
		assert.NoError(t, err)

		processor, err := NewProcessor(cfg, logptest.NewTestingLogger(t, ""))
		assert.NoError(t, err)

		newEvent, err := processor.Run(e)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, e.Fields, newEvent.Fields)
	})
}

func TestFieldDoesntExist(t *testing.T) {
	c, err := conf.NewConfigFrom(map[string]interface{}{"tokenizer": "hello %{key}"})
	if !assert.NoError(t, err) {
		return
	}

	processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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
			c, err := conf.NewConfigFrom(map[string]interface{}{
				"tokenizer":     test.tokenizer,
				"target_prefix": test.prefix,
			})

			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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
		c, err := conf.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{ok} - %{notvalid}",
		})

		if !assert.NoError(t, err) {
			return
		}

		processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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
		c, err := conf.NewConfigFrom(map[string]interface{}{
			"tokenizer": "%{ok} %{valid}",
		})

		if !assert.NoError(t, err) {
			return
		}

		processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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
			c, err := conf.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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
			c, err := conf.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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
			c, err := conf.NewConfigFrom(test.c)
			if !assert.NoError(t, err) {
				return
			}

			processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
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

// TestPrefixWithIndirectField verifies that dynamically-created keys
// from indirect fields (%{?name}=%{&name}) are still prefixed correctly.
func TestPrefixWithIndirectField(t *testing.T) {
	settings := map[string]interface{}{
		"tokenizer":     `%{?k1}=%{&k1} msg="%{message}"`,
		"field":         "message",
		"target_prefix": "dissect",
	}
	c, _ := conf.NewConfigFrom(settings)
	p, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	event := &beat.Event{
		Fields: mapstr.M{
			"message": `id=7736 msg="hello"`,
		},
	}

	result, err := p.Run(event)
	if err != nil {
		t.Fatal(err)
	}

	// The indirect field creates a dynamic key "id" with value "7736".
	// With target_prefix="dissect", it should become "dissect.id".
	val, err := result.GetValue("dissect.id")
	if err != nil {
		t.Fatalf("expected dissect.id to exist: %v", err)
	}
	if val != "7736" {
		t.Fatalf("expected dissect.id=7736, got %v", val)
	}

	// Also verify the static field
	val, err = result.GetValue("dissect.message")
	if err != nil {
		t.Fatalf("expected dissect.message to exist: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected dissect.message=hello, got %v", val)
	}
}

// BenchmarkDissectProcessor benchmarks the full processor Run path
// with the dissector already constructed (the real hot path).
func BenchmarkDissectProcessor(b *testing.B) {
	tests := []struct {
		name   string
		tok    string
		msg    string
		prefix string
	}{
		{
			name:   "6_fields_default_prefix",
			tok:    `id=%{id} status=%{status} duration=%{duration} uptime=%{uptime} success=%{success} msg="%{message}"`,
			msg:    `id=7736 status=202 duration=0.975 uptime=1588975628 success=true msg="Request accepted"`,
			prefix: "dissect",
		},
		{
			name:   "6_fields_with_prefix",
			tok:    `id=%{id} status=%{status} duration=%{duration} uptime=%{uptime} success=%{success} msg="%{message}"`,
			msg:    `id=7736 status=202 duration=0.975 uptime=1588975628 success=true msg="Request accepted"`,
			prefix: "dissect",
		},
		{
			name:   "6_fields_nested_prefix",
			tok:    `id=%{id} status=%{status} duration=%{duration} uptime=%{uptime} success=%{success} msg="%{message}"`,
			msg:    `id=7736 status=202 duration=0.975 uptime=1588975628 success=true msg="Request accepted"`,
			prefix: "dissect.parsed",
		},
		{
			// Envoyproxy-style access log with default prefix "dissect"
			// 10 extracted fields — realistic complex pattern
			name:   "envoy_access_log_default_prefix",
			tok:    `%{log_type} [%{timestamp}] "%{method} %{path} %{proto}" %{response_code} %{response_flags} %{bytes_received} %{bytes_sent} %{duration} %{upstream_service_time}`,
			msg:    `ACCESS [2026-04-08T12:00:00.000Z] "GET /api/v1/users HTTP/1.1" 200 - 0 1234 42 38`,
			prefix: "dissect",
		},
		{
			// Cisco ASA 106001 pattern — real ECS dotted field names, no prefix
			name:   "cisco_asa_ecs_no_prefix",
			tok:    `%{network.direction} %{network.transport} connection %{event.outcome} from %{source.address}/%{source.port} to %{destination.address}/%{destination.port} flags %{} on interface %{observer.ingress.interface.name}`,
			msg:    `Inbound TCP connection permitted from 192.168.1.100/44523 to 10.0.0.1/443 flags SYN on interface outside`,
			prefix: "",
		},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			settings := map[string]interface{}{
				"tokenizer":     tc.tok,
				"field":         "message",
				"target_prefix": tc.prefix,
			}
			c, _ := conf.NewConfigFrom(settings)
			p, err := NewProcessor(c, logptest.NewTestingLogger(b, ""))
			if err != nil {
				b.Fatal(err)
			}

			event := &beat.Event{
				Fields: mapstr.M{
					"message": tc.msg,
				},
			}

			// Warm up
			if _, err := p.Run(event); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				// Reset the event for each iteration
				event.Fields = mapstr.M{
					"message": tc.msg,
				}
				_, _ = p.Run(event)
			}
		})
	}
}

// TestDissectOverwriteKeysSafety verifies that the pre-check for existing keys
// prevents partial writes when OverwriteKeys=false (the default). This proves
// the Clone() skip is safe: the processor checks all keys before writing any.
func TestDissectOverwriteKeysSafety(t *testing.T) {
	c, err := conf.NewConfigFrom(map[string]interface{}{
		"tokenizer":     "hello %{key}",
		"target_prefix": "",
	})
	require.NoError(t, err)

	processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	input := mapstr.M{
		"message": "hello world",
		"key":     "existing-value",
	}
	event := &beat.Event{Fields: input.Clone()}
	original := input.Clone()

	result, err := processor.Run(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot override existing key")

	// Remove error.message and dissect flags added by the processor.
	result.Fields.Delete("error")
	result.Fields.Delete(beat.FlagField)
	assert.Equal(t, original, result.Fields,
		"event fields must be unchanged when key conflict is detected (clone skip safety)")
}
