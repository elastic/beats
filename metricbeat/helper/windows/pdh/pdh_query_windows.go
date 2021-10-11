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
// +build windows

package pdh

import (
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/pkg/errors"
)

var (
	instanceNameRegexp = regexp.MustCompile(`(\(.+\))\\`)
	objectNameRegexp   = regexp.MustCompile(`(?:^\\\\[^\\]+\\|^\\)([^\\]+)`)
)

// Counter object will contain the handle and format of each performance counter.
type Counter struct {
	handle       PdhCounterHandle
	format       PdhCounterFormat
	instanceName string
}

// Query contains the pdh.
type Query struct {
	Handle   PdhQueryHandle
	Counters map[string]*Counter
}

// CounterValue contains the performance counter values.
type CounterValue struct {
	Instance    string
	Measurement interface{}
	Err         CounterValueError
}

// CounterValueError contains the performance counter error.
type CounterValueError struct {
	Error   error
	CStatus uint32
}

// Open creates a new query.
func (q *Query) Open() error {
	h, err := PdhOpenQuery("", 0)
	if err != nil {
		return err
	}
	q.Handle = h
	q.Counters = make(map[string]*Counter)
	return nil
}

// AddEnglishCounter adds the specified counter to the query.
func (q *Query) AddEnglishCounter(counterPath string) (PdhCounterHandle, error) {
	h, err := PdhAddEnglishCounter(q.Handle, counterPath, 0)
	return h, err
}

// AddCounter adds the specified counter to the query.
func (q *Query) AddCounter(counterPath string, instance string, format string, wildcard bool) error {
	if _, found := q.Counters[counterPath]; found {
		return nil
	}
	var err error
	var instanceName string
	// Extract the instance name from the counterPath.
	if instance == "" || wildcard {
		instanceName, err = matchInstanceName(counterPath)
		if err != nil {
			return err
		}
	} else {
		instanceName = instance
	}
	h, err := PdhAddCounter(q.Handle, counterPath, 0)
	if err != nil {
		return err
	}

	q.Counters[counterPath] = &Counter{
		handle:       h,
		instanceName: instanceName,
		format:       getPDHFormat(format),
	}
	return nil
}

// GetCounterPaths func will check the computer or log file and return the counter paths that match the given counter path which contains wildcard characters.
func (q *Query) GetCounterPaths(counterPath string) ([]string, error) {
	paths, err := q.ExpandWildCardPath(counterPath)
	if err == nil {
		return paths, err
	}
	//check if Windows installed language is not ENG, the ExpandWildCardPath will return either one of the errors below.
	if err == PDH_CSTATUS_NO_OBJECT || err == PDH_CSTATUS_NO_COUNTER {
		handle, err := q.AddEnglishCounter(counterPath)
		if err != nil {
			return nil, err
		}
		defer PdhRemoveCounter(handle)
		info, err := PdhGetCounterInfo(handle)
		if err != nil {
			return nil, err
		}
		path := UTF16PtrToString(info.SzFullPath)
		if path != counterPath {
			return q.ExpandWildCardPath(path)
		}
	}
	return nil, err
}

// RemoveUnusedCounters will remove all counter handles for the paths that are not found anymore
func (q *Query) RemoveUnusedCounters(counters []string) error {
	// check if the expandwildcard func did expand th wildcard queries, if not, no counters will be removed
	for _, counter := range counters {
		if strings.Contains(counter, "*") {
			return nil
		}
	}
	unused := make(map[string]*Counter)
	for counterPath, counter := range q.Counters {
		if !matchCounter(counterPath, counters) {
			unused[counterPath] = counter
		}
	}
	if len(unused) == 0 {
		return nil
	}
	for counterPath, cnt := range unused {
		err := PdhRemoveCounter(cnt.handle)
		if err != nil {
			return err
		}
		delete(q.Counters, counterPath)
	}
	return nil
}

func matchCounter(counterPath string, counterList []string) bool {
	for _, cn := range counterList {
		if cn == counterPath {
			return true
		}
	}
	return false
}

// CollectData collects the value for all counters in the query.
func (q *Query) CollectData() error {
	return PdhCollectQueryData(q.Handle)
}

// CollectData collects the value for all counters in the query.
func (q *Query) CollectDataEx(interval uint32, event windows.Handle) error {
	return PdhCollectQueryDataEx(q.Handle, interval, event)
}

// GetFormattedCounterValues returns an array of formatted values for a query.
func (q *Query) GetFormattedCounterValues() (map[string][]CounterValue, error) {
	if q.Counters == nil || len(q.Counters) == 0 {
		return nil, errors.New("no counter list found")
	}
	rtn := make(map[string][]CounterValue, len(q.Counters))
	for path, counter := range q.Counters {
		rtn[path] = append(rtn[path], getCounterValue(counter))
	}
	return rtn, nil
}

// GetCountersAndInstances returns a list of counters and instances for a given object
func (q *Query) GetCountersAndInstances(objectName string) ([]string, []string, error) {
	counters, instances, err := PdhEnumObjectItems(objectName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Unable to retrieve counter and instance list for %s", objectName)
	}
	if len(counters) == 0 && len(instances) == 0 {
		return nil, nil, errors.Errorf("Unable to retrieve counter and instance list for %s", objectName)
	}
	return UTF16ToStringArray(counters), UTF16ToStringArray(instances), nil
}

// ExpandWildCardPath  examines local computer and returns those counter paths that match the given counter path which contains wildcard characters.
func (q *Query) ExpandWildCardPath(wildCardPath string) ([]string, error) {
	if wildCardPath == "" {
		return nil, errors.New("no query path given")
	}
	utfPath, err := syscall.UTF16PtrFromString(wildCardPath)
	if err != nil {
		return nil, err
	}
	var expdPaths []uint16

	// PdhExpandWildCardPath will not return the counter paths for windows 32 bit systems but PdhExpandCounterPath will.
	if runtime.GOARCH == "386" {
		if expdPaths, err = PdhExpandCounterPath(utfPath); err != nil {
			return nil, err
		}
		if expdPaths == nil {
			return nil, errors.New("no counter paths found")
		}
		return UTF16ToStringArray(expdPaths), nil
	} else {
		if expdPaths, err = PdhExpandWildCardPath(utfPath); err != nil {
			if err == PDH_MORE_DATA {
				if expdPaths, err = PdhExpandWildCardPath(utfPath); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		paths := UTF16ToStringArray(expdPaths)
		// in several cases ExpandWildCardPath win32 api seems to return initial wildcard without any errors, adding some waiting time between the 2 ExpandWildCardPath api calls seems to be succesfull but that will delay data retrieval
		// A call is triggered again
		if len(paths) == 1 && strings.Contains(paths[0], "*") && paths[0] == wildCardPath {
			expdPaths, err = PdhExpandWildCardPath(utfPath)
			if err == nil {
				return paths, err
			}
		} else {
			return paths, err
		}
	}

	return nil, PdhErrno(syscall.ERROR_NOT_FOUND)
}

// Close closes the query and all of its counters.
func (q *Query) Close() error {
	return PdhCloseQuery(q.Handle)
}

// matchInstanceName will check first for instance and then for any objects names.
func matchInstanceName(counterPath string) (string, error) {
	matches := instanceNameRegexp.FindStringSubmatch(counterPath)
	if len(matches) == 2 {
		return returnLastInstance(matches[1]), nil
	}
	matches = objectNameRegexp.FindStringSubmatch(counterPath)
	if len(matches) == 2 {
		return matches[1], nil
	}
	return "", errors.New("query doesn't contain an instance name. In this case you have to define 'instance_name'")
}

// returnLastInstance will return the content from the last parentheses, this covers cases as `\WF (System.Workflow) 4.0.0.0(*)\Workflows Created`.
func returnLastInstance(match string) string {
	var openedParanth int
	var innerMatch string
	var matches []string
	runeMatch := []rune(match)
	for i := 0; i < len(runeMatch); i++ {
		char := string(runeMatch[i])

		// check if string ends between parentheses
		if char == ")" {
			openedParanth -= 1
		}
		if openedParanth > 0 {
			innerMatch += char
		}
		if openedParanth == 0 && innerMatch != "" {
			matches = append(matches, innerMatch)
			innerMatch = ""
		}
		// check if string starts between parentheses
		if char == "(" {
			openedParanth += 1
		}
	}
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	return match
}

// getCounterValue will retrieve the counter value based on the format applied in the config options
func getCounterValue(counter *Counter) CounterValue {
	counterValue := CounterValue{Instance: counter.instanceName, Err: CounterValueError{CStatus: 0}}
	switch counter.format {
	case PdhFmtLong:
		_, value, err := PdhGetFormattedCounterValueLong(counter.handle)
		if err != nil {
			counterValue.Err.Error = err
			if value != nil {
				counterValue.Err.CStatus = value.CStatus
			}
		} else {
			counterValue.Measurement = value.Value
		}
	case PdhFmtLarge:
		_, value, err := PdhGetFormattedCounterValueLarge(counter.handle)
		if err != nil {
			counterValue.Err.Error = err
			if value != nil {
				counterValue.Err.CStatus = value.CStatus
			}
		} else {
			counterValue.Measurement = value.Value
		}
	case PdhFmtDouble:
		_, value, err := PdhGetFormattedCounterValueDouble(counter.handle)
		if err != nil {
			counterValue.Err.Error = err
			if value != nil {
				counterValue.Err.CStatus = value.CStatus
			}
		} else {
			counterValue.Measurement = value.Value
		}
	default:
		counterValue.Err.Error = errors.Errorf("initialization failed: format '%#v' "+
			"for instance '%s' is invalid (must be PdhFmtDouble, PdhFmtLarge or PdhFmtLong)",
			counter.format, counter.instanceName)
	}

	return counterValue
}

// getPDHFormat calculates data pdhformat.
func getPDHFormat(format string) PdhCounterFormat {
	switch format {
	case "long":
		return PdhFmtLong
	case "large":
		return PdhFmtLarge
	default:
		return PdhFmtDouble
	}
}

// UTF16ToStringArray converts list of Windows API NULL terminated strings  to Go string array.
func UTF16ToStringArray(buf []uint16) []string {
	var strings []string
	nextLineStart := 0
	stringLine := UTF16PtrToString(&buf[0])
	for stringLine != "" {
		strings = append(strings, stringLine)
		nextLineStart += len([]rune(stringLine)) + 1
		remainingBuf := buf[nextLineStart:]
		stringLine = UTF16PtrToString(&remainingBuf[0])
	}
	return strings
}

// UTF16PtrToString converts Windows API LPTSTR (pointer to string) to Go string.
func UTF16PtrToString(s *uint16) string {
	if s == nil {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 29]uint16)(unsafe.Pointer(s))[0:])
}
