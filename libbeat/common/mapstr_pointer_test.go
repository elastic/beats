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

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMapStrPointer(t *testing.T) {
	data := MapStr{
		"foo": "bar",
	}

	p := NewMapStrPointer(data)
	assert.Equal(t, p.Get(), data)

	newData := MapStr{
		"new": "data",
	}
	p.Set(newData)
	assert.Equal(t, p.Get(), newData)
}

func BenchmarkMapStrPointer(b *testing.B) {
	p := NewMapStrPointer(MapStr{"counter": 0})
	go func() {
		counter := 0
		for {
			counter++
			p.Set(MapStr{"counter": counter})
			time.Sleep(10 * time.Millisecond)
		}
	}()

	for n := 0; n < b.N; n++ {
		_ = p.Get()
	}
}
