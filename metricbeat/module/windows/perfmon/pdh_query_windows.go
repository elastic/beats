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
	"strconv"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

var (
	instanceNameRegexp = regexp.MustCompile(`.*\((.*)\).*`)
	objectNameRegexp   = regexp.MustCompile(`(?:^\\\\[^\\]+\\|^\\)([^\\]+)`)
)

type Counter struct {
	handle       PdhCounterHandle
	format       PdhCounterFormat
	instanceName string
}

type Counters map[string]*Counter

type Query struct {
	handle   PdhQueryHandle
	counters Counters
}

type Value struct {
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
	q.counters = make(Counters)
	return nil
}

// AddCounter adds the specified counter to the query.
func (q *Query) AddCounter(counterPath string, counter CounterConfig, wildcard bool) error {
	if _, found := q.counters[counterPath]; found {
		return errors.Errorf("Counter %s has been already added", counterPath)
	}
	var err error
	var instanceName string
	// Extract the instance name from the counterPath.
	if wildcard || (!wildcard && counter.InstanceName == "") {
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
		format:       getFormat(counter.Format),
	}
	return nil
}

// CollectData collects the value for all counters in the query.
func (q *Query) CollectData() error {
	return PdhCollectQueryData(q.handle)
}

// GetFormattedCounterValues returns an array of formatted values for a query.
func (q *Query) GetFormattedCounterValues() (map[string][]Value, error) {
	rtn := make(map[string][]Value, len(q.counters))

	for path, counter := range q.counters {
		_, value, err := PdhGetFormattedCounterValue(counter.handle, counter.format|PdhFmtNoCap100)
		if err != nil {
			rtn[path] = append(rtn[path], Value{Err: err})
			continue
		}

		switch counter.format {
		case PdhFmtDouble:
			rtn[path] = append(rtn[path], Value{Measurement: *(*float64)(unsafe.Pointer(&value.LongValue)), Instance: counter.instanceName})
		case PdhFmtLarge:
			rtn[path] = append(rtn[path], Value{Measurement: *(*int64)(unsafe.Pointer(&value.LongValue)), Instance: counter.instanceName})

		}
	}

	return rtn, nil
}

// ExpandWildCardPath  examines local computer and returns those counter paths that match the given counter path which contains wildcard characters.
func (q *Query) ExpandWildCardPath(wildCardPath string) ([]string, error) {
	if wildCardPath == "" {
		return nil, errors.New("no wildcardpath given")
	}

	expdPaths, err := PdhExpandWildCardPath(wildCardPath)
	if err != nil {
		return nil, err
	}
	return UTF16ToStringArray(expdPaths), nil

}

// Close closes the query and all of its counters.
func (q *Query) Close() error {
	return PdhCloseQuery(q.handle)
}

// Error returns a more explicit error message
func (e PdhErrno) Error() string {
	// If the value is not one of the known PDH errors then assume its a
	// general windows error.
	if _, found := pdhErrors[e]; !found {
		return syscall.Errno(e).Error()
	}

	// Use FormatMessage to convert the PDH errno to a string.
	// Example: https://msdn.microsoft.com/en-us/library/windows/desktop/aa373046(v=vs.85).aspx
	var flags uint32 = windows.FORMAT_MESSAGE_FROM_HMODULE | windows.FORMAT_MESSAGE_ARGUMENT_ARRAY | windows.FORMAT_MESSAGE_IGNORE_INSERTS
	b := make([]uint16, 300)
	n, err := windows.FormatMessage(flags, modpdh.Handle(), uint32(e), 0, b, nil)
	if err != nil {
		return "pdh error #" + strconv.Itoa(int(e))
	}

	// Trim terminating \r and \n
	for ; n > 0 && (b[n-1] == '\n' || b[n-1] == '\r'); n-- {
	}
	return string(utf16.Decode(b[:n]))
}

// matchInstanceName will check first for instance and then for any objects names
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

// calculates data format
func getFormat(format string) PdhCounterFormat {
	switch format {
	case "double":
		return PdhFmtDouble
	case "long":
		return PdhFmtLarge
	}
	return PdhFmtDouble
}

// UTF16ToStringArray converts list of Windows API NULL terminated strings  to go string array
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

// UTF16PtrToString converts Windows API LPTSTR (pointer to string) to go string
func UTF16PtrToString(s *uint16) string {
	if s == nil {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 29]uint16)(unsafe.Pointer(s))[0:])
}
