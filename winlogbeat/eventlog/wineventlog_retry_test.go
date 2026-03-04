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

//go:build windows

package eventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderNoEventRetryCounter(t *testing.T) {
	l := &winEventLog{}

	assert.Equal(t, 1, l.incrementRenderNoEventRetry("bookmark-1"))
	assert.Equal(t, 2, l.incrementRenderNoEventRetry("bookmark-1"))
	assert.Equal(t, 1, l.incrementRenderNoEventRetry("bookmark-2"))
	assert.Equal(t, 2, l.incrementRenderNoEventRetry("bookmark-2"))

	l.resetRenderNoEventRetry()
	assert.Equal(t, 1, l.incrementRenderNoEventRetry("bookmark-2"))
}

func TestRenderNoEventRetryCounterEmptyBookmark(t *testing.T) {
	l := &winEventLog{}

	assert.Equal(t, 1, l.incrementRenderNoEventRetry(""))
	assert.Equal(t, 2, l.incrementRenderNoEventRetry(""))
}

func TestGapRetryCounter(t *testing.T) {
	l := &winEventLog{}

	assert.Equal(t, 1, l.incrementGapRetry("Application:100:102"))
	assert.Equal(t, 2, l.incrementGapRetry("Application:100:102"))
	assert.Equal(t, 1, l.incrementGapRetry("Application:102:110"))
	assert.Equal(t, 2, l.incrementGapRetry("Application:102:110"))

	l.resetGapRetry()
	assert.Equal(t, 1, l.incrementGapRetry("Application:102:110"))
}

func TestRenderNoEventRetryKey(t *testing.T) {
	t.Run("uses bookmark when present", func(t *testing.T) {
		err := &renderNoEventError{bookmark: "bookmark-1"}
		assert.Equal(t, "bookmark-1", err.RetryKey())
	})

	t.Run("falls back when bookmark missing", func(t *testing.T) {
		err := &renderNoEventError{cause: assert.AnError}
		assert.Contains(t, err.RetryKey(), "no-bookmark:")
	})
}
