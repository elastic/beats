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

package monitoring

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafeVars(t *testing.T) {
	t.Run("no concurrency", func(t *testing.T) {
		uintValName := "testUint"
		testReg := NewRegistry().NewRegistry("safe_registry")
		testUint := NewUint(testReg, uintValName)
		testUint.Set(5)
		// Add the first time
		require.NotNil(t, testUint)

		// Add the metric a second time
		testSecondUint := NewUint(testReg, uintValName)
		require.NotNil(t, testSecondUint)
		// make sure we fetch the same unit
		require.Equal(t, uint64(5), testSecondUint.Get())
	})

	t.Run("with concurrency", func(t *testing.T) {
		t.Run("NewInt", func(t *testing.T) {
			reg := NewRegistry()
			name := "foo"

			wg := sync.WaitGroup{}
			assert.NotPanics(t, func() {
				for i := 0; i < 1000; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						NewInt(reg, name)
					}()
				}
			})
			wg.Wait()
		})

		t.Run("NewUint", func(t *testing.T) {
			reg := NewRegistry()
			name := "foo"

			wg := sync.WaitGroup{}
			assert.NotPanics(t, func() {
				for i := 0; i < 1000; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						NewUint(reg, name)
					}()
				}
			})
			wg.Wait()
		})

		t.Run("NewFloat", func(t *testing.T) {
			reg := NewRegistry()
			name := "foo"

			wg := sync.WaitGroup{}
			assert.NotPanics(t, func() {
				for i := 0; i < 1000; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						NewFloat(reg, name)
					}()
				}
			})
			wg.Wait()
		})

		t.Run("NewBool", func(t *testing.T) {
			reg := NewRegistry()
			name := "foo"

			wg := sync.WaitGroup{}
			assert.NotPanics(t, func() {
				for i := 0; i < 1000; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						NewBool(reg, name)
					}()
				}
			})
			wg.Wait()
		})

		t.Run("NewString", func(t *testing.T) {
			reg := NewRegistry()
			name := "foo"

			wg := sync.WaitGroup{}
			assert.NotPanics(t, func() {
				for i := 0; i < 1000; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						NewString(reg, name)
					}()
				}
			})
			wg.Wait()
		})

		t.Run("NewFunc", func(t *testing.T) {
			reg := NewRegistry()
			name := "foo"
			dummyFunc := func(m Mode, v Visitor) {}

			wg := sync.WaitGroup{}
			assert.NotPanics(t, func() {
				for i := 0; i < 1000; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						NewFunc(reg, name, dummyFunc)
					}()
				}
			})
			wg.Wait()
		})
		
		t.Run("NewTimestamp", func(t *testing.T) {
			reg := NewRegistry()
			name := "foo"

			wg := sync.WaitGroup{}
			assert.NotPanics(t, func() {
				for i := 0; i < 1000; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						NewTimestamp(reg, name)
					}()
				}
			})
			wg.Wait()
		})
	})
}

func TestVarsTypes(t *testing.T) {
	testReg := Default.NewRegistry("test_type_reg")

	expected := map[string]interface{}{
		"string_key": "string_val",
		"bool_key":   false,
		"int_key":    int64(42),
		"float_key":  42.1,
		"slice_key":  []string{"test", "string"},
	}

	NewFunc(testReg, "test", func(m Mode, v Visitor) {
		ReportString(v, "string_key", "string_val")
		ReportBool(v, "bool_key", false)
		ReportInt(v, "int_key", 42)
		ReportFloat(v, "float_key", 42.1)
		ReportStringSlice(v, "slice_key", []string{"test", "string"})
	})

	gotData := CollectStructSnapshot(testReg, Full, false)

	require.Equal(t, expected, gotData)
}

func TestNilReg(t *testing.T) {
	uintValName := "testUint"
	// This can also just panic if there's a bug
	testUint := NewUint(nil, uintValName)
	require.NotNil(t, testUint)

}
