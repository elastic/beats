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

// +build windows

package perfmon

import (
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

var (
	instanceNameRegexp = regexp.MustCompile(`.*?\((.*?)\).*`)
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
	handle   PdhQueryHandle
	counters map[string]*Counter
}

// CounterValue contains the performance counter values.
type CounterValue struct {
	Instance    string
	Measurement interface{}
	Err         error
}

// Open creates a new query.
func (q *Query) Open() error {
	h, err := PdhOpenQuery("", 0)
	if err != nil {
		return err
	}
	q.handle = h
	q.counters = make(map[string]*Counter)
	return nil
}

// AddCounter adds the specified counter to the query.
func (q *Query) AddCounter(counterPath string, counter CounterConfig, wildcard bool) error {
	if _, found := q.counters[counterPath]; found {
		return nil
	}
	var err error
	var instanceName string
	// Extract the instance name from the counterPath.
	if counter.InstanceName == "" || wildcard {
		instanceName, err = matchInstanceName(counterPath)
		if err != nil {
			return err
		}
	} else {
		instanceName = counter.InstanceName
	}
	h, err := PdhAddCounter(q.handle, counterPath, 0)
	if err != nil {
		return err
	}

	q.counters[counterPath] = &Counter{
		handle:       h,
		instanceName: instanceName,
		format:       getPDHFormat(counter.Format),
	}
	return nil
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
	for counterPath, counter := range q.counters {
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
		delete(q.counters, counterPath)
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
	return PdhCollectQueryData(q.handle)
}

// GetFormattedCounterValues returns an array of formatted values for a query.
func (q *Query) GetFormattedCounterValues() (map[string][]CounterValue, error) {
	if q.counters == nil || len(q.counters) == 0 {
		return nil, errors.New("no counter list found")
	}
	rtn := make(map[string][]CounterValue, len(q.counters))
	for path, counter := range q.counters {
		rtn[path] = append(rtn[path], getCounterValue(counter))
	}
	return rtn, nil
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
		expdPaths, err = PdhExpandCounterPath(utfPath)
	} else {
		expdPaths, err = PdhExpandWildCardPath(utfPath)
	}
	if err != nil {
		return nil, err
	}
	if expdPaths == nil {
		return nil, errors.New("no counter paths found")
	}
	return UTF16ToStringArray(expdPaths), nil
}

// Close closes the query and all of its counters.
func (q *Query) Close() error {
	return PdhCloseQuery(q.handle)
}

// matchInstanceName will check first for instance and then for any objects names.
func matchInstanceName(counterPath string) (string, error) {
	matches := instanceNameRegexp.FindStringSubmatch(counterPath)
	if len(matches) != 2 {
		matches = objectNameRegexp.FindStringSubmatch(counterPath)
	}
	if len(matches) == 2 {
		return matches[1], nil
	}
	return "", errors.New("query doesn't contain an instance name. In this case you have to define 'instance_name'")
}

// getCounterValue will retrieve the counter value based on the format applied in the config options
func getCounterValue(counter *Counter) CounterValue {
	counterValue := CounterValue{Instance: counter.instanceName}
	switch counter.format {
	case PdhFmtLong:
		_, value, err := PdhGetFormattedCounterValueLong(counter.handle)
		if err != nil {
			counterValue.Err = err
		} else {
			counterValue.Measurement = value.Value
		}
	case PdhFmtLarge:
		_, value, err := PdhGetFormattedCounterValueLarge(counter.handle)
		if err != nil {
			counterValue.Err = err
		} else {
			counterValue.Measurement = value.Value
		}
	case PdhFmtDouble:
		_, value, err := PdhGetFormattedCounterValueDouble(counter.handle)
		if err != nil {
			counterValue.Err = err
		} else {
			counterValue.Measurement = value.Value
		}
	default:
		counterValue.Err = errors.Errorf("initialization failed: format '%#v' "+
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
		nextLineStart += len(stringLine) + 1
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
