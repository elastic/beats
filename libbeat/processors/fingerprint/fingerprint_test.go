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

package fingerprint

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestWithConfig(t *testing.T) {
	cases := map[string]struct {
		config common.MapStr
		input  common.MapStr
		want   string
	}{
		"hello world": {
			config: common.MapStr{
				"fields": []string{"message"},
			},
			input: common.MapStr{
				"message": "hello world",
			},
			want: "50110bbfc1757f21caacc966b33f5ea2235c4176739447e0b3285dec4e1dd2a4",
		},
		"with string escaping": {
			config: common.MapStr{
				"fields": []string{"message"},
			},
			input: common.MapStr{
				"message": `test message "hello world"`,
			},
			want: "14a0364b79acbe4c78dd5e77db2c93ae8c750518b32581927d50b3eef407184e",
		},
		"with @timestamp": {
			config: common.MapStr{
				"fields": []string{"@timestamp", "message"},
			},
			input: common.MapStr{
				"message": `test message "hello world"`,
			},
			want: "081da76e049554943843b83948ac83ab7aa79fd2849331813e02042586021c26",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			config := config.MustNewConfigFrom(test.config)
			p, err := New(config)
			require.NoError(t, err)

			testEvent := &beat.Event{
				Timestamp: time.Unix(1635443183, 0),
				Fields:    test.input.Clone(),
			}
			newEvent, err := p.Run(testEvent)
			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, test.want, v)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		config := config.MustNewConfigFrom(common.MapStr{
			"fields":       []string{"@metadata.message"},
			"target_field": "@metadata.fingerprint",
		})
		p, err := New(config)
		require.NoError(t, err)

		testEvent := &beat.Event{
			Timestamp: time.Unix(1635443183, 0),
			Meta: common.MapStr{
				"message": "hello world",
			},
		}

		expMeta := common.MapStr{
			"message":     "hello world",
			"fingerprint": "1a3fe8251076ed8de5fd99ce529d2b9971c54851d4d45f5a576bed91d0cc4202",
		}
		newEvent, err := p.Run(testEvent)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, testEvent.Fields, newEvent.Fields)
	})
}

func TestHashMethods(t *testing.T) {
	testFields := common.MapStr{
		"field1":       "foo",
		"field2":       "bar",
		"unused_field": "baz",
	}

	tests := map[string]struct {
		expected string
	}{
		"md5":    {"4c45df4792f3ef850c928ec5f5232538"},
		"sha1":   {"22f76427d626516d3f7a05785165b99617683b22"},
		"sha256": {"1208288932231e313b369bae587ff574cd3016a408e52e7128d7bee752674003"},
		"sha384": {"295adfe0bc03908948e4b0b6a54f441767867e426dda590430459c8a147fbba242a38cba282adee78335b9e08877b86c"},
		"sha512": {"f50ad51b63c92a0ed0c910527119b81806f3110f0afaa1dcb93506a78371ea761e50c0fc09b08c441d832dd2da1b45e5d8361adfb240e1fffc2695122a23e183"},
		"xxhash": {"37bc50682fba6686"},
	}

	for method, test := range tests {
		t.Run(method, func(t *testing.T) {
			testConfig, err := config.NewConfigFrom(common.MapStr{
				"fields": []string{"field1", "field2"},
				"method": method,
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			testEvent := &beat.Event{
				Fields:    testFields.Clone(),
				Timestamp: time.Now(),
			}

			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, test.expected, v)
		})
	}
}

func TestSourceFields(t *testing.T) {
	testFields := common.MapStr{
		"field1": "foo",
		"field2": "bar",
		"nested": common.MapStr{
			"field": "qux",
		},
		"unused_field": "baz",
	}
	expectedFingerprint := "3d51237d384215a6e731f2cc67ead6d7d9a5138377897c8f542a915be3c25bcf"

	tests := map[string]struct {
		fields []string
	}{
		"order_1":            {[]string{"field1", "nested.field"}},
		"order_2":            {[]string{"nested.field", "field1"}},
		"duplicates_ignored": {[]string{"nested.field", "field1", "nested.field"}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testConfig, err := config.NewConfigFrom(common.MapStr{
				"fields": test.fields,
				"method": "sha256",
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			testEvent := &beat.Event{
				Fields:    testFields.Clone(),
				Timestamp: time.Now(),
			}
			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, expectedFingerprint, v)
		})
	}
}

func TestEncoding(t *testing.T) {
	testFields := common.MapStr{
		"field1": "foo",
		"field2": "bar",
		"nested": common.MapStr{
			"field": "qux",
		},
		"unused_field": "baz",
	}

	tests := map[string]struct {
		expectedFingerprint string
	}{
		"hex":    {"8934ca639027aab1ee9f3944d4d6bd1e"},
		"base32": {"RE2MUY4QE6VLD3U7HFCNJVV5DY======"},
		"base64": {"iTTKY5AnqrHunzlE1Na9Hg=="},
	}

	for encoding, test := range tests {
		t.Run(encoding, func(t *testing.T) {
			testConfig, err := config.NewConfigFrom(common.MapStr{
				"fields":   []string{"field2", "nested.field"},
				"method":   "md5",
				"encoding": encoding,
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			testEvent := &beat.Event{
				Fields:    testFields.Clone(),
				Timestamp: time.Now(),
			}
			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, test.expectedFingerprint, v)
		})
	}
}

func TestConsistentHashingTimeFields(t *testing.T) {
	tzUTC := time.UTC
	tzPST := time.FixedZone("Pacific Standard Time", int((-8 * time.Hour).Seconds()))
	tzIST := time.FixedZone("Indian Standard Time", int((5*time.Hour + 30*time.Minute).Seconds()))

	expectedFingerprint := "4534d56a673c2da41df32db5da87cf47e639e84fe82907f2c015c8dfcac5d4f5"

	tests := map[string]struct {
		event common.MapStr
	}{
		"UTC": {
			common.MapStr{
				"timestamp": time.Date(2019, 10, 29, 0, 0, 0, 0, tzUTC),
			},
		},
		"PST": {
			common.MapStr{
				"timestamp": time.Date(2019, 10, 28, 16, 0, 0, 0, tzPST),
			},
		},
		"IST": {
			common.MapStr{
				"timestamp": time.Date(2019, 10, 29, 5, 30, 0, 0, tzIST),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testConfig, err := config.NewConfigFrom(common.MapStr{
				"fields": []string{"timestamp"},
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			testEvent := &beat.Event{
				Fields: test.event,
			}
			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, expectedFingerprint, v)
		})
	}
}

func TestTargetField(t *testing.T) {
	testFields := common.MapStr{
		"field1": "foo",
		"nested": common.MapStr{
			"field": "bar",
		},
		"unused_field": "baz",
	}
	expectedFingerprint := "4cf8b768ad20266c348d63a6d1ff5d6f6f9ed0f59f5c68ae031b78e3e04c5144"

	tests := map[string]struct {
		targetField string
	}{
		"root":   {"target_field"},
		"nested": {"nested.target_field"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testConfig, err := config.NewConfigFrom(common.MapStr{
				"fields":       []string{"field1"},
				"target_field": test.targetField,
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			testEvent := &beat.Event{
				Fields:    testFields.Clone(),
				Timestamp: time.Now(),
			}
			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue(test.targetField)
			assert.NoError(t, err)
			assert.Equal(t, expectedFingerprint, v)

			_, err = newEvent.GetValue("fingerprint")
			assert.EqualError(t, err, common.ErrKeyNotFound.Error())
		})
	}
}

func TestSourceFieldErrors(t *testing.T) {
	testFields := common.MapStr{
		"field1": "foo",
		"field2": "bar",
		"complex_field": map[string]interface{}{
			"child": "qux",
		},
		"unused_field": "baz",
	}

	tests := map[string]struct {
		fields []string
	}{
		"missing": {
			[]string{"field1", "missing_field"},
		},
		"non-scalar": {
			[]string{"field1", "complex_field"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testConfig, err := config.NewConfigFrom(common.MapStr{
				"fields": test.fields,
				"method": "sha256",
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			testEvent := &beat.Event{
				Fields:    testFields.Clone(),
				Timestamp: time.Now(),
			}
			_, err = p.Run(testEvent)
			assert.IsType(t, errComputeFingerprint{}, err)
		})
	}
}

func TestInvalidConfig(t *testing.T) {
	tests := map[string]struct {
		config common.MapStr
	}{
		"no fields": {
			common.MapStr{
				"fields": []string{},
				"method": "sha256",
			},
		},
		"invalid fingerprinting method": {
			common.MapStr{
				"fields": []string{"doesnt", "matter"},
				"method": "non_existent",
			},
		},
		"invalid encoding": {
			common.MapStr{
				"fields":   []string{"doesnt", "matter"},
				"encoding": "non_existent",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testConfig, err := config.NewConfigFrom(test.config)
			assert.NoError(t, err)

			_, err = New(testConfig)
			assert.IsType(t, errConfigUnpack{}, err)
		})
	}
}

func TestIgnoreMissing(t *testing.T) {
	testFields := common.MapStr{
		"field1": "foo",
	}

	tests := map[string]struct {
		assertErr           assert.ErrorAssertionFunc
		expectedFingerprint string
	}{
		"true": {
			assert.NoError,
			"4cf8b768ad20266c348d63a6d1ff5d6f6f9ed0f59f5c68ae031b78e3e04c5144",
		},
		"false": {
			assertErr: assert.Error,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ignoreMissing, _ := strconv.ParseBool(name)
			testConfig, err := config.NewConfigFrom(common.MapStr{
				"fields":         []string{"field1", "missing_field"},
				"ignore_missing": ignoreMissing,
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			testEvent := &beat.Event{
				Fields:    testFields.Clone(),
				Timestamp: time.Now(),
			}
			newEvent, err := p.Run(testEvent)
			test.assertErr(t, err)

			if err == nil {
				v, err := newEvent.GetValue("fingerprint")
				assert.NoError(t, err)
				assert.Equal(t, test.expectedFingerprint, v)
			}
		})
	}
}

func BenchmarkHashMethods(b *testing.B) {
	events := nRandomEvents(100000)

	for method := range hashes {
		testConfig, _ := config.NewConfigFrom(common.MapStr{
			"fields": []string{"message"},
			"method": method,
		})

		p, _ := New(testConfig)

		b.Run(method, func(b *testing.B) {
			b.ResetTimer()
			for _, e := range events {
				_, err := p.Run(&e)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func nRandomEvents(num int) []beat.Event {
	prng := rand.New(rand.NewSource(12345))

	const charset = "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"0123456789"
	charsetLen := len(charset)
	b := make([]byte, 200)

	var events []beat.Event
	for i := 0; i < num; i++ {
		for j := range b {
			b[j] = charset[prng.Intn(charsetLen)]
		}
		events = append(events, beat.Event{
			Fields: common.MapStr{
				"message": string(b),
			},
		})
	}

	return events
}
