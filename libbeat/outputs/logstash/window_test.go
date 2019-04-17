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

// +build !integration

package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShrinkWindowSizeNeverZero(t *testing.T) {
	enableLogging([]string{"logstash"})

	windowSize := 124
	var w window
	w.init(windowSize, defaultConfig().BulkMaxSize)

	w.windowSize = int32(windowSize)
	for i := 0; i < 100; i++ {
		w.shrinkWindow()
	}

	assert.Equal(t, 1, int(w.windowSize))
}

func TestGrowWindowSizeUpToBatchSizes(t *testing.T) {
	batchSize := 114
	windowSize := 1024
	testGrowWindowSize(t, 10, 0, windowSize, batchSize, batchSize)
}

func TestGrowWindowSizeUpToMax(t *testing.T) {
	batchSize := 114
	windowSize := 64
	testGrowWindowSize(t, 10, 0, windowSize, batchSize, windowSize)
}

func TestGrowWindowSizeOf1(t *testing.T) {
	batchSize := 114
	windowSize := 1024
	testGrowWindowSize(t, 1, 0, windowSize, batchSize, batchSize)
}

func TestGrowWindowSizeToMaxOKOnly(t *testing.T) {
	batchSize := 114
	windowSize := 1024
	maxOK := 71
	testGrowWindowSize(t, 1, maxOK, windowSize, batchSize, maxOK)
}

func testGrowWindowSize(t *testing.T,
	initial, maxOK, windowSize, batchSize, expected int,
) {
	enableLogging([]string{"logstash"})
	var w window
	w.init(initial, windowSize)
	w.maxOkWindowSize = maxOK
	for i := 0; i < 100; i++ {
		w.tryGrowWindow(batchSize)
	}

	assert.Equal(t, expected, int(w.windowSize))
	assert.Equal(t, expected, int(w.maxOkWindowSize))
}
