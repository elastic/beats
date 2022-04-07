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

package json

import (
	"testing"
	"time"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
)

var result []byte

func BenchmarkUTCTime(b *testing.B) {
	var r []byte
	codec := New("1.2.3", Config{})
	fields := common.MapStr{"msg": "message"}
	var t time.Time
	var d time.Duration = 1000000000

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t = t.Add(d)
		r, _ = codec.Encode("test", &beat.Event{Fields: fields, Timestamp: t})
	}
	result = r
}

func BenchmarkLocalTime(b *testing.B) {
	var r []byte
	codec := New("1.2.3", Config{LocalTime: true})
	fields := common.MapStr{"msg": "message"}
	var t time.Time
	var d time.Duration = 1000000000

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		t = t.Add(d)
		r, _ = codec.Encode("test", &beat.Event{Fields: fields, Timestamp: t})
	}
	result = r
}
