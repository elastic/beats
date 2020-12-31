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

package pdh

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOpenSuccessful will open query successfully.
func TestOpenSuccessful(t *testing.T) {
	var q Query
	err := q.Open()
	assert.NoError(t, err)
	defer q.Close()
}

// TestAddCounterInvalidArgWhenQueryClosed will check if addcounter func fails when query is closed.
func TestAddCounterInvalidArgWhenQueryClosed(t *testing.T) {
	var q Query
	queryPath, err := q.GetCounterPaths(validQuery)
	// if windows os language is ENG then err will be nil, else the GetCounterPaths will execute the AddCounter
	if assert.NoError(t, err) {
		err = q.AddCounter(queryPath[0], "TestInstanceName", "float", false)
		assert.Error(t, err, PDH_INVALID_HANDLE)
	} else {
		assert.Error(t, err, PDH_INVALID_ARGUMENT)
	}
}

// func TestGetFormattedCounterValuesEmptyCounterList will check if getting the counter values will fail when no counter handles are added.
func TestGetFormattedCounterValuesEmptyCounterList(t *testing.T) {
	var q Query
	list, err := q.GetFormattedCounterValues()
	assert.Nil(t, list)
	assert.EqualValues(t, err.Error(), "no counter list found")
}

// TestExpandWildCardPathWithEmptyString will check for a valid path string.
func TestExpandWildCardPathWithEmptyString(t *testing.T) {
	var q Query
	list, err := q.ExpandWildCardPath("")
	assert.Nil(t, list)
	assert.EqualValues(t, err.Error(), "no query path given")
}

// TestSuccessfulQuery retrieves a per counter successfully.
func TestSuccessfulQuery(t *testing.T) {
	var q Query
	err := q.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer q.Close()
	queryPath, err := q.GetCounterPaths(validQuery)
	if err != nil {
		t.Fatal(err)
	}
	err = q.AddCounter(queryPath[0], "TestInstanceName", "floar", false)
	if err != nil {
		t.Fatal(err)
	}
	//Some counters, such as rate counters, require two counter values in order to compute a displayable value. In this case we must call PdhCollectQueryData twice before calling PdhGetFormattedCounterValue.
	// For more information, see Collecting Performance Data (https://docs.microsoft.com/en-us/windows/desktop/PerfCtrs/collecting-performance-data).
	err = q.CollectData()
	err = q.CollectData()
	if err != nil {
		t.Fatal(err)
	}
	list, err := q.GetFormattedCounterValues()
	assert.NoError(t, err)
	assert.NotNil(t, list)
}

func TestMatchInstanceName(t *testing.T) {
	query := "\\SQLServer:Databases(*)\\Log File(s) Used Size (KB)"
	match, err := matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "*")

	query = " \\\\desktop-rfooe09\\per processor network interface card activity(3, microsoft wi-fi directvirtual (gyfyg) adapter #2)\\dpcs queued/sec"
	match, err = matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "3, microsoft wi-fi directvirtual (gyfyg) adapter #2")

	query = " \\\\desktop-rfooe09\\ (test this scenario) per processor network interface card activity(3, microsoft wi-fi directvirtual (gyfyg) adapter #2)\\dpcs queued/sec"
	match, err = matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "3, microsoft wi-fi directvirtual (gyfyg) adapter #2")

	query = "\\RAS\\Bytes Received By Disconnected Clients"
	match, err = matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "RAS")

	query = `\\Process (chrome.exe#4)\\Bytes Received By Disconnected Clients`
	match, err = matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "chrome.exe#4")

	query = "\\BranchCache\\Local Cache: Cache complete file segments"
	match, err = matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "BranchCache")

	query = `\Synchronization(*)\Exec. Resource no-Waits AcqShrdStarveExcl/sec`
	match, err = matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "*")

	query = `\.NET CLR Exceptions(test hellp (dsdsd) #rfsfs #3)\# of Finallys / sec`
	match, err = matchInstanceName(query)
	assert.NoError(t, err)
	assert.Equal(t, match, "test hellp (dsdsd) #rfsfs #3")
}

// TestInstanceNameRegexp tests regular expression for instance.
func TestInstanceNameRegexp(t *testing.T) {
	queryPaths := []string{`\SQLServer:Databases(*)\Log File(s) Used Size (KB)`, `\Search Indexer(*)\L0 Indexes (Wordlists)`,
		`\Search Indexer(*)\L0 Merges (flushes) Now.`, `\NUMA Node Memory(*)\Free & Zero Page List MBytes`}
	for _, path := range queryPaths {
		matches := instanceNameRegexp.FindStringSubmatch(path)
		if assert.Len(t, matches, 2, "regular expression did not return any matches") {
			assert.Equal(t, matches[1], "(*)")
		}
	}
}

// TestObjectNameRegexp tests regular expression for object.
func TestObjectNameRegexp(t *testing.T) {
	queryPaths := []string{`\Web Service Cache\Output Cache Current Flushed Items`,
		`\Web Service Cache\Output Cache Total Flushed Items`, `\Web Service Cache\Total Flushed Metadata`,
		`\Web Service Cache\Kernel: Current URIs Cached`}
	for _, path := range queryPaths {
		matches := objectNameRegexp.FindStringSubmatch(path)
		if assert.Len(t, matches, 2, "regular expression did not return any matches") {
			assert.Equal(t, matches[1], "Web Service Cache")
		}
	}
}

func TestReturnLastInstance(t *testing.T) {
	query := "(*)"
	match := returnLastInstance(query)
	assert.Equal(t, match, "*")

	query = "(3, microsoft wi-fi directvirtual (gyfyg) adapter #2)"
	match = returnLastInstance(query)
	assert.Equal(t, match, "3, microsoft wi-fi directvirtual (gyfyg) adapter #2")

	query = "(test this scenario) per processor network interface card activity(3, microsoft wi-fi directvirtual (gyfyg) adapter #2)"
	match = returnLastInstance(query)
	assert.Equal(t, match, "3, microsoft wi-fi directvirtual (gyfyg) adapter #2")

	query = `(chrome.exe#4)`
	match = returnLastInstance(query)
	assert.Equal(t, match, "chrome.exe#4")

	query = `(test hellp (dsdsd) #rfsfs #3)`
	match = returnLastInstance(query)
	assert.Equal(t, match, "test hellp (dsdsd) #rfsfs #3")
}

func TestUTF16ToStringArray(t *testing.T) {
	var array = []string{"\\\\DESKTOP-RFOOE09\\Physikalischer Datenträger(0 C:)\\Schreibvorgänge/s", "\\\\DESKTOP-RFOOE09\\Physikalischer Datenträger(_Total)\\Schreibvorgänge/s", ""}
	var unicode []uint16
	for _, i := range array {
		uni, err := syscall.UTF16FromString(i)
		assert.NoError(t, err)
		unicode = append(unicode, uni...)
	}
	response := UTF16ToStringArray(unicode)
	assert.NotNil(t, response)
	assert.Equal(t, len(response), 2)
	for _, res := range response {
		assert.Contains(t, array, res)
	}
}
