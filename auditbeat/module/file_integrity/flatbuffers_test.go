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

package file_integrity

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFBEncodeDecode(t *testing.T) {
	e := testEvent()

	builder, release := fbGetBuilder()
	defer release()
	data := fbEncodeEvent(builder, e)
	t.Log("encoded length:", len(data))

	out := fbDecodeEvent(e.Path, data)
	if out == nil {
		t.Fatal("decode returned nil")
	}

	assert.Equal(t, *e.Info, *out.Info)
	e.Info, out.Info = nil, nil
	assert.Equal(t, e, out)
}

func BenchmarkFBEncodeEvent(b *testing.B) {
	builder, release := fbGetBuilder()
	defer release()
	e := testEvent()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		builder.Reset()
		fbEncodeEvent(builder, e)
	}
}

func BenchmarkFBEventDecode(b *testing.B) {
	builder, release := fbGetBuilder()
	defer release()
	e := testEvent()
	data := fbEncodeEvent(builder, e)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if event := fbDecodeEvent(e.Path, data); event == nil {
			b.Fatal("failed to decode")
		}
	}
}

// JSON benchmarks for comparisons.

func BenchmarkJSONEventEncoding(b *testing.B) {
	e := testEvent()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(e)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONEventDecode(b *testing.B) {
	e := testEvent()
	data, err := json.Marshal(e)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var e *Event
		err := json.Unmarshal(data, &e)
		if err != nil {
			b.Fatal(err)
		}
	}
}
