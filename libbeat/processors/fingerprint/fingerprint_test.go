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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestHashMethods(t *testing.T) {
	testEvent := &beat.Event{
		Fields: common.MapStr{
			"field1":       "foo",
			"field2":       "bar",
			"unused_field": "baz",
		},
		Timestamp: time.Now(),
	}

	tests := []struct {
		method   string
		expected string
	}{
		{
			"md5",
			"4c45df4792f3ef850c928ec5f5232538",
		},
		{
			"sha1",
			"22f76427d626516d3f7a05785165b99617683b22",
		},
		{
			"sha256",
			"1208288932231e313b369bae587ff574cd3016a408e52e7128d7bee752674003",
		},
		{
			"sha384",
			"295adfe0bc03908948e4b0b6a54f441767867e426dda590430459c8a147fbba242a38cba282adee78335b9e08877b86c",
		},
		{
			"sha512",
			"f50ad51b63c92a0ed0c910527119b81806f3110f0afaa1dcb93506a78371ea761e50c0fc09b08c441d832dd2da1b45e5d8361adfb240e1fffc2695122a23e183",
		},
	}

	for _, test := range tests {
		name := fmt.Sprintf("testing %v fingerprinting method", test.method)
		t.Run(name, func(t *testing.T) {
			testConfig, err := common.NewConfigFrom(common.MapStr{
				"fields": []string{"field1", "field2"},
				"method": test.method,
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, test.expected, v)
		})
	}
}

func TestSourceFields(t *testing.T) {
	testEvent := &beat.Event{
		Fields: common.MapStr{
			"field1": "foo",
			"field2": "bar",
			"nested": common.MapStr{
				"field": "qux",
			},
			"unused_field": "baz",
		},
		Timestamp: time.Now(),
	}
	expectedFingerprint := "3d51237d384215a6e731f2cc67ead6d7d9a5138377897c8f542a915be3c25bcf"

	tests := []struct {
		name   string
		fields []string
	}{
		{
			"order is insignificant",
			[]string{"field1", "nested.field"},
		},
		{
			"order is insignificant",
			[]string{"nested.field", "field1"},
		},
		{
			"duplicates are ignored",
			[]string{"nested.field", "field1", "nested.field"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testConfig, err := common.NewConfigFrom(common.MapStr{
				"fields": test.fields,
				"method": "sha256",
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			newEvent, err := p.Run(testEvent)
			assert.NoError(t, err)

			v, err := newEvent.GetValue("fingerprint")
			assert.NoError(t, err)
			assert.Equal(t, expectedFingerprint, v)
		})
	}
}

func TestEncoding(t *testing.T) {
	testEvent := &beat.Event{
		Fields: common.MapStr{
			"field1": "foo",
			"field2": "bar",
			"nested": common.MapStr{
				"field": "qux",
			},
			"unused_field": "baz",
		},
		Timestamp: time.Now(),
	}

	tests := []struct {
		encoding            string
		expectedFingerprint string
	}{
		{
			"hex",
			"8934ca639027aab1ee9f3944d4d6bd1e",
		},
		{
			"base32",
			"RE2MUY4QE6VLD3U7HFCNJVV5DY======",
		},
		{
			"base64",
			"iTTKY5AnqrHunzlE1Na9Hg==",
		},
	}

	for _, test := range tests {
		name := fmt.Sprintf("testing %v encoding", test.encoding)
		t.Run(name, func(t *testing.T) {
			testConfig, err := common.NewConfigFrom(common.MapStr{
				"fields":   []string{"field2", "nested.field"},
				"method":   "md5",
				"encoding": test.encoding,
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

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
	tzMTV := time.FixedZone("Mountain View, California, USA", int((-8 * time.Hour).Seconds()))
	tzBOM := time.FixedZone("Bombay, Maharashtra, India", int((5*time.Hour + 30*time.Minute).Seconds()))

	expectedFingerprint := "4534d56a673c2da41df32db5da87cf47e639e84fe82907f2c015c8dfcac5d4f5"

	tests := []struct {
		name  string
		event common.MapStr
	}{
		{
			"time field in UTC",
			common.MapStr{
				"timestamp": time.Date(2019, 10, 29, 0, 0, 0, 0, tzUTC),
			},
		},
		{
			"time field in Mountain View time",
			common.MapStr{
				"timestamp": time.Date(2019, 10, 28, 16, 0, 0, 0, tzMTV),
			},
		},
		{
			"time field in Bombay time",
			common.MapStr{
				"timestamp": time.Date(2019, 10, 29, 5, 30, 0, 0, tzBOM),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testConfig, err := common.NewConfigFrom(common.MapStr{
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

func TestSourceFieldErrors(t *testing.T) {
	testEvent := &beat.Event{
		Fields: common.MapStr{
			"field1": "foo",
			"field2": "bar",
			"complex_field": map[string]interface{}{
				"child": "qux",
			},
			"unused_field": "baz",
		},
		Timestamp: time.Now(),
	}

	tests := []struct {
		name           string
		fields         []string
		expectedErrMsg string
	}{
		{
			"missing field",
			[]string{"field1", "missing_field"},
			"failed to compute fingerprint: failed to find field [missing_field] in event: key not found",
		},
		{
			"non-scalar field",
			[]string{"field1", "complex_field"},
			"failed to compute fingerprint: cannot compute fingerprint using non-scalar field [complex_field]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testConfig, err := common.NewConfigFrom(common.MapStr{
				"fields": test.fields,
				"method": "sha256",
			})
			assert.NoError(t, err)

			p, err := New(testConfig)
			assert.NoError(t, err)

			_, err = p.Run(testEvent)
			assert.EqualError(t, err, test.expectedErrMsg)
		})
	}
}

func TestInvalidConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         common.MapStr
		expectedErrMsg string
	}{
		{
			"no fields",
			common.MapStr{
				"fields": []string{},
				"method": "sha256",
			},
			"failed to unpack fingerprint processor configuration: empty field accessing 'fields'",
		},
		{
			"invalid fingerprinting method",
			common.MapStr{
				"fields": []string{"doesnt", "matter"},
				"method": "non_existent",
			},
			"failed to unpack fingerprint processor configuration: invalid fingerprinting method [non_existent] accessing 'method'",
		},
		{
			"invalid encoding",
			common.MapStr{
				"fields":   []string{"doesnt", "matter"},
				"encoding": "non_existent",
			},
			"failed to unpack fingerprint processor configuration: invalid encoding method [non_existent]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testConfig, err := common.NewConfigFrom(test.config)
			assert.NoError(t, err)

			_, err = New(testConfig)
			assert.EqualError(t, err, test.expectedErrMsg)
		})
	}
}
