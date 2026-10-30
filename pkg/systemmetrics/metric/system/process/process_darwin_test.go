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

//go:build darwin && cgo

package process

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOne_footprint(t *testing.T) {
	const allocSize = 64 << 20

	stats, err := initTestResolver(t)
	require.NoError(t, err, "failed to init process stats")

	before := eventFootprint(t, &stats)
	assert.Positive(t, before, "footprint should be non-zero for the test process")

	// mmap keeps the Go allocator and GC from reusing or releasing memory
	// during the measurement.
	buf, err := syscall.Mmap(-1, 0, allocSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	require.NoError(t, err, "failed to mmap test buffer")
	t.Cleanup(func() {
		assert.NoError(t, syscall.Munmap(buf), "failed to munmap test buffer")
	})
	for i := 0; i < len(buf); i += os.Getpagesize() {
		buf[i] = 1
	}

	after := eventFootprint(t, &stats)
	assert.InDeltaf(t, float64(allocSize), float64(after)-float64(before), allocSize/4,
		"footprint should grow by about %d bytes after dirtying %d bytes of anonymous memory; before=%d after=%d",
		allocSize, allocSize, before, after)
}

func eventFootprint(t *testing.T, stats *Stats) uint64 {
	t.Helper()

	event, err := stats.GetOne(os.Getpid())
	require.NoError(t, err, "GetOne failed for the test process")

	val, err := event.GetValue("memory.footprint")
	require.NoErrorf(t, err, "event has no memory.footprint; memory: %v", event["memory"])

	footprint, ok := val.(uint64)
	require.Truef(t, ok, "memory.footprint has type %T, want uint64", val)
	return footprint
}
