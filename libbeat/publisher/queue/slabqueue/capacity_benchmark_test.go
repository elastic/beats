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

package slabqueue

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Measure empty queue allocation separately from retained event payloads.
// The slot metrics expose the aggregate configured capacity and chunk rounding;
// B/op also includes free-list, queue, and directory allocation overhead.
func BenchmarkCappedPoolCapacity(b *testing.B) {
	const perReceiver = 3200
	for _, receivers := range []int{1, 4, 8, 16} {
		b.Run(fmt.Sprintf("receivers=%d", receivers), func(b *testing.B) {
			var target, capacity int
			for b.Loop() {
				pool := NewPool[publisher.Event](Settings{Events: perReceiver}, nil)
				for range receivers {
					q := pool.Connect()
					q.SetTarget(perReceiver)
				}
				target, capacity = pool.Target(), pool.Capacity()
				pool.Shutdown()
			}
			b.ReportMetric(float64(target), "slots")
			b.ReportMetric(float64(numChunks(capacity)*int(unsafe.Sizeof(chunk[publisher.Event]{}))), "slab-storage-B")
		})
	}
}
