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

// +build gotestsample

package testdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssertOutput(t *testing.T) {
	t.Run("assert fails", func(t *testing.T) {
		assert.True(t, false)
	})

	t.Run("assert with message", func(t *testing.T) {
		assert.True(t, false, "My message")
	})

	t.Run("assert with messagef", func(t *testing.T) {
		assert.True(t, false, "My message with arguments: %v", 42)
	})

	t.Run("require fails", func(t *testing.T) {
		require.True(t, false)
	})

	t.Run("require with message", func(t *testing.T) {
		require.True(t, false, "My message")
	})

	t.Run("require with messagef", func(t *testing.T) {
		require.True(t, false, "My message with arguments: %v", 42)
	})

	t.Run("equals map", func(t *testing.T) {
		want := map[string]interface{}{
			"a": 1,
			"b": true,
			"c": "test",
			"e": map[string]interface{}{
				"x": "y",
			},
		}

		got := map[string]interface{}{
			"a": 42,
			"b": false,
			"c": "test",
		}

		assert.Equal(t, want, got)
	})
}

func TestLogOutput(t *testing.T) {
	t.Run("on error", func(t *testing.T) {
		t.Log("Log message should be printed")
		t.Logf("printf style log message: %v", 42)
		t.Error("Log should fail")
		t.Errorf("Log should fail with printf style log: %v", 23)
	})

	t.Run("on fatal", func(t *testing.T) {
		t.Log("Log message should be printed")
		t.Logf("printf style log message: %v", 42)
		t.Fatal("Log should fail")
	})

	t.Run("on fatalf", func(t *testing.T) {
		t.Log("Log message should be printed")
		t.Logf("printf style log message: %v", 42)
		t.Fatalf("Log should fail with printf style log: %v", 42)
	})

	t.Run("with newlines", func(t *testing.T) {
		t.Log("Log\nmessage\nshould\nbe\nprinted")
		t.Logf("printf\nstyle\nlog\nmessage:\n%v", 42)
		t.Fatalf("Log\nshould\nfail\nwith\nprintf\nstyle\nlog:\n%v", 42)
	})
}

func TestWithPanic(t *testing.T) {
	panic("Kaputt.")
}

func TestWithWrongPanic(t *testing.T) {
	t.Run("setup failing go-routine", func(t *testing.T) {
		go func() {
			time.Sleep(1 * time.Second)
			t.Fatal("oops")
		}()
	})

	t.Run("false positive failure", func(t *testing.T) {
		time.Sleep(10 * time.Second)
	})
}
