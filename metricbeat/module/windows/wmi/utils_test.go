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

package wmi

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	wmi "github.com/microsoft/wmi/pkg/wmiinstance"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type MockWmiSession struct {
}

const MockTimeout = time.Second * 5

// Mock Implementation of QueryInstances function
// This simulate a long-running query
func (c *MockWmiSession) QueryInstances(queryExpression string) ([]*wmi.WmiInstance, error) {
	time.Sleep(MockTimeout)
	return []*wmi.WmiInstance{}, nil
}

func TestExecuteGuardedQueryInstances(t *testing.T) {
	mockSession := new(MockWmiSession)
	query := "SELECT * FROM Win32_OpeartingSystem"
	timeout := 200 * time.Millisecond

	startTime := time.Now()
	expectedError := fmt.Errorf("the execution of the query '%s' exceeded the warning threshold of %s", query, timeout)
	_, err := ExecuteGuardedQueryInstances(mockSession, query, timeout, logptest.NewTestingLogger(t, "wmi"))
	// Make sure the return time is less than the MockTimeout
	assert.Less(t, time.Since(startTime), MockTimeout, "The return time should be less than the sleep time")
	// Make sure the error returned is the expected one
	assert.Equal(t, err, expectedError, "Expected the returned error to match the expected error")
}

const TEST_DATE_FORMAT string = "2006-01-02T15:04:05.999999-07:00"

// Dummy internal conversion function for tests: converts string to uint64
func dummyUint64Converter(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func TestGenericWmiConversionFunction(t *testing.T) {
	type testCase struct {
		name        string
		input       interface{}
		want        interface{}
		expectError bool
	}

	tests := []testCase{
		{
			name:  "single string",
			input: "123",
			want:  uint64(123),
		},
		{
			name:  "slice of strings as []interface{}",
			input: []interface{}{"1", "2", "3"},
			want:  []uint64{1, 2, 3},
		},
		{
			name:        "slice contains non-string",
			input:       []interface{}{"1", 2, "3"},
			expectError: true,
		},
		{
			name:        "unsupported type (int)",
			input:       42,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GenericWmiConversionFunction[uint64](tc.input, dummyUint64Converter)
			if (err != nil) != tc.expectError {
				t.Fatalf("unexpected error state: got err %v, want error? %v", err, tc.expectError)
			}
			if err == nil {
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("unexpected result: got %#v, want %#v", got, tc.want)
				}
			}
		})
	}
}

func Test_ConversionFunctions(t *testing.T) {
	tests := []struct {
		name        string
		conversion  WmiConversionFunction
		input       interface{}
		expected    interface{}
		expectErr   bool
		description string
	}{
		// Test cases for ConvertUint64
		{
			name:        "ConvertUint64 - valid input",
			conversion:  ConvertUint64,
			input:       "12345",
			expected:    uint64(12345),
			expectErr:   false,
			description: "Should convert string to uint64",
		},
		{
			name:        "ConvertUint64 - valid array input",
			conversion:  ConvertUint64,
			input:       []interface{}{"12345", "345"},
			expected:    []uint64{12345, 345},
			expectErr:   false,
			description: "Should convert string to uint64",
		},
		{
			name:        "ConvertUint64 - invalid input",
			conversion:  ConvertUint64,
			input:       "notANumber",
			expected:    nil,
			expectErr:   true,
			description: "Should return error for invalid uint64 string",
		},
		// Test cases for ConvertSint64
		{
			name:        "ConvertSint64 - valid input",
			conversion:  ConvertSint64,
			input:       "-12345",
			expected:    int64(-12345),
			expectErr:   false,
			description: "Should convert string to sint64",
		},
		{
			name:        "ConvertSint64 - valid array input",
			conversion:  ConvertSint64,
			input:       []interface{}{"-12345", "345"},
			expected:    []int64{-12345, 345},
			expectErr:   false,
			description: "Should convert string to uint64",
		},
		{
			name:        "ConvertSint64 - invalid input",
			conversion:  ConvertSint64,
			input:       "notANumber",
			expected:    nil,
			expectErr:   true,
			description: "Should return error for invalid sint64 string",
		},

		// Test cases for ConvertDatetime
		{
			name:        "ConvertDatetime - valid input",
			conversion:  ConvertDatetime,
			input:       "20231224093045.123456+000",
			expected:    mustParseTime(TEST_DATE_FORMAT, "2023-12-24T09:30:45.123456+00:00"),
			expectErr:   false,
			description: "Should convert string to time.Time",
		},
		{
			name:        "ConvertDatetime - valid input - timezone set",
			conversion:  ConvertDatetime,
			input:       "20231224093045.123456-690",
			expected:    mustParseTime(TEST_DATE_FORMAT, "2023-12-24T09:30:45.123456-11:30"),
			expectErr:   false,
			description: "Should convert string to time.Time",
		},
		{
			name:        "ConvertDatetime - invalid input",
			conversion:  ConvertDatetime,
			input:       "invalidDatetime",
			expected:    nil,
			expectErr:   true,
			description: "Should return error for invalid datetime string",
		},
		// Test cases for ConvertIdentity
		{
			name:        "ConvertIdentity - valid input",
			conversion:  ConvertIdentity,
			input:       "test string",
			expected:    "test string",
			expectErr:   false,
			description: "Should return the same string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.conversion(tt.input)

			if tt.expectErr {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.expected, result, tt.description)
			}
		})
	}
}

// Helper function to parse time
func mustParseTime(layout, value string) time.Time {
	parsed, err := time.Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func Test_WMISchema_Get_BaseAndSubclass(t *testing.T) {
	// Set up base and subclass schema
	base := map[string]WmiConversionFunction{
		"BaseFieldA": ConvertIdentity,
		"BaseFieldB": ConvertSint64,
	}

	schema := NewWMISchema(base)
	schema.Put("SubClass", "SubClassField", ConvertSint64)

	// Base key should resolve
	if fn, ok := schema.Get("SubClass", "BaseFieldA"); !ok || fn == nil {
		t.Errorf("expected to get 'BaseFieldA' key from base class")
	}

	// Subclass key should resolve
	if fn, ok := schema.Get("SubClass", "SubClassField"); !ok || fn == nil {
		t.Errorf("expected to get SubOnly key from subclass")
	}

	// Missing key should fail
	if _, ok := schema.Get("SubClass", "Missing"); ok {
		t.Errorf("did not expect to find missing key")
	}
}

func Test_WMISchema_Put_AddsToSubClass(t *testing.T) {
	schema := NewWMISchema(make(map[string]WmiConversionFunction))

	if _, ok := schema.Get("NewClass", "Key"); ok {
		t.Errorf("expected key to be missing before Put")
	}

	schema.Put("NewClass", "Key", ConvertIdentity)

	if fn, ok := schema.Get("NewClass", "Key"); !ok || fn == nil {
		t.Errorf("expected key to be present after Put")
	}
}
