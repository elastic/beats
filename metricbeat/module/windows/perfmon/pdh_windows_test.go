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

package perfmon

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

var validQuery = `\Processor Information(_Total)\% Processor Time`

// TestPdhErrno checks that PdhError provides the correct message for known
// PDH errors and also falls back to Windows error messages for non-PDH errors.
func TestPdhErrno_Error(t *testing.T) {
	assert.Contains(t, PdhErrno(PDH_CSTATUS_BAD_COUNTERNAME).Error(), "Unable to parse the counter path.")
	assert.Contains(t, PdhErrno(15).Error(), "The system cannot find the drive specified.")
}

// TestPdhOpenQueryInvalidQuery will check for file source and throw exception.
func TestPdhOpenQueryInvalidQuery(t *testing.T) {
	handle, err := PdhOpenQuery("invalid string", 0)
	assert.EqualValues(t, handle, InvalidQueryHandle)
	assert.EqualValues(t, err, PDH_FILE_NOT_FOUND)
}

// TestPdhAddCounterInvalidCounter checks for invalid query.
func TestPdhAddCounterInvalidCounter(t *testing.T) {
	handle, err := PdhAddCounter(InvalidQueryHandle, validQuery, 0)
	assert.EqualValues(t, handle, InvalidCounterHandle)
	assert.EqualValues(t, err, PDH_INVALID_ARGUMENT)
}

// TestPdhGetFormattedCounterValueInvalidCounter will test for invalid counters.
func TestPdhGetFormattedCounterValueInvalidCounter(t *testing.T) {
	counterType, counterValue, err := PdhGetFormattedCounterValueDouble(InvalidCounterHandle)
	assert.EqualValues(t, counterType, 0)
	assert.EqualValues(t, counterValue, (*PdhCounterValueDouble)(nil))
	assert.EqualValues(t, err, PDH_INVALID_HANDLE)
}

// TestPdhExpandWildCardPathInvalidPath will test for invalid query path.
func TestPdhExpandWildCardPathInvalidPath(t *testing.T) {
	utfPath, err := syscall.UTF16PtrFromString("sdfhsdhfd")
	assert.Nil(t, err)
	queryList, err := PdhExpandWildCardPath(utfPath)
	assert.Nil(t, queryList)
	assert.EqualValues(t, err, PDH_INVALID_PATH)
}

// TestPdhCollectQueryDataInvalidQuery will check for invalid query.
func TestPdhCollectQueryDataInvalidQuery(t *testing.T) {
	err := PdhCollectQueryData(InvalidQueryHandle)
	assert.EqualValues(t, err, PDH_INVALID_HANDLE)
}

// TestPdhCloseQueryInvalidQuery will check for invalid query.
func TestPdhCloseQueryInvalidQuery(t *testing.T) {
	err := PdhCloseQuery(InvalidQueryHandle)
	assert.EqualValues(t, err, PDH_INVALID_HANDLE)
}

// TestPdhSuccessfulCounterRetrieval will execute the PDH  functions successfully.
func TestPdhSuccessfulCounterRetrieval(t *testing.T) {
	queryHandle, err := PdhOpenQuery("", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer PdhCloseQuery(queryHandle)
	utfPath, err := syscall.UTF16PtrFromString(validQuery)
	if err != nil {
		t.Fatal(err)
	}
	queryList, err := PdhExpandWildCardPath(utfPath)
	if err != nil {
		t.Fatal(err)
	}
	queries := UTF16ToStringArray(queryList)
	var counters []PdhCounterHandle
	for _, query := range queries {
		counterHandle, err := PdhAddCounter(queryHandle, query, 0)
		if err != nil && err != PDH_NO_MORE_DATA {
			t.Fatal(err)
		}
		counters = append(counters, counterHandle)
	}
	//Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	err = PdhCollectQueryData(queryHandle)
	err = PdhCollectQueryData(queryHandle)
	if err != nil {
		t.Fatal(err)
	}
	for _, counter := range counters {
		counterType, counterValue, err := PdhGetFormattedCounterValueDouble(counter)
		assert.Nil(t, err)
		assert.NotZero(t, counterType)
		assert.NotNil(t, counterValue)
	}

}
