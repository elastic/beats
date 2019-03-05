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
	"bytes"
	"regexp"
	"strconv"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/winlogbeat/sys"
)

// Windows API calls
//sys _PdhOpenQuery(dataSource *uint16, userData uintptr, query *PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhOpenQueryW
//sys _PdhAddCounter(query PdhQueryHandle, counterPath string, userData uintptr, counter *PdhCounterHandle) (errcode error) [failretval!=0] = pdh.PdhAddEnglishCounterW
//sys _PdhCollectQueryData(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCollectQueryData
//sys _PdhGetFormattedCounterValue(counter PdhCounterHandle, format PdhCounterFormat, counterType *uint32, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhGetFormattedCounterValue
//sys _PdhGetFormattedCounterArray(counter PdhCounterHandle, format PdhCounterFormat, bufferSize *uint32, bufferCount *uint32, itemBuffer *byte) (errcode error) [failretval!=0] = pdh.PdhGetFormattedCounterArrayW
//sys _PdhGetRawCounterValue(counter PdhCounterHandle, counterType *uint32, value *PdhRawCounter) (errcode error) [failretval!=0] = pdh.PdhGetRawCounterValue
//sys _PdhGetRawCounterArray(counter PdhCounterHandle, bufferSize *uint32, bufferCount *uint32, itemBuffer *pdhRawCounterItem) (errcode error) [failretval!=0] = pdh.PdhGetRawCounterArray
//sys _PdhCalculateCounterFromRawValue(counter PdhCounterHandle, format PdhCounterFormat, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhCalculateCounterFromRawValue
//sys _PdhFormatFromRawValue(counterType uint32, format PdhCounterFormat, timeBase *uint64, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhFormatFromRawValue
//sys _PdhCloseQuery(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCloseQuery

var (
	sizeofPdhCounterValueItem = (int)(unsafe.Sizeof(pdhCounterValueItem{}))
	wildcardRegexp            = regexp.MustCompile(`.*\(\*\).*`)
	instanceNameRegexp        = regexp.MustCompile(`.*\((.*)\).*`)
)

type PdhQueryHandle uintptr

var InvalidQueryHandle = ^PdhQueryHandle(0)

type PdhCounterHandle uintptr

var InvalidCounterHandle = ^PdhCounterHandle(0)

type pdhCounterValueItem struct {
	SzName   uintptr
	FmtValue PdhCounterValue
}

type pdhRawCounterItem struct {
	SzName   uintptr
	RawValue PdhRawCounter
}

type CounterValueItem struct {
	Name  string
	Value PdhCounterValue
}

func PdhOpenQuery(dataSource string, userData uintptr) (PdhQueryHandle, error) {
	var dataSourcePtr *uint16
	if dataSource != "" {
		var err error
		dataSourcePtr, err = syscall.UTF16PtrFromString(dataSource)
		if err != nil {
			return InvalidQueryHandle, err
		}
	}

	var handle PdhQueryHandle
	if err := _PdhOpenQuery(dataSourcePtr, userData, &handle); err != nil {
		return InvalidQueryHandle, PdhErrno(err.(syscall.Errno))
	}

	return handle, nil
}

func PdhAddCounter(query PdhQueryHandle, counterPath string, userData uintptr) (PdhCounterHandle, error) {
	var handle PdhCounterHandle
	if err := _PdhAddCounter(query, counterPath, userData, &handle); err != nil {
		return InvalidCounterHandle, PdhErrno(err.(syscall.Errno))
	}

	return handle, nil
}

func PdhCollectQueryData(query PdhQueryHandle) error {
	if err := _PdhCollectQueryData(query); err != nil {
		return PdhErrno(err.(syscall.Errno))
	}

	return nil
}

func PdhGetFormattedCounterValue(counter PdhCounterHandle, format PdhCounterFormat) (uint32, *PdhCounterValue, error) {
	var counterType uint32
	var value PdhCounterValue
	if err := _PdhGetFormattedCounterValue(counter, format, &counterType, &value); err != nil {
		return 0, nil, PdhErrno(err.(syscall.Errno))
	}

	return counterType, &value, nil
}

func PdhGetFormattedCounterArray(counter PdhCounterHandle, format PdhCounterFormat) ([]CounterValueItem, error) {
	var bufferSize uint32
	var bufferCount uint32

	if err := _PdhGetFormattedCounterArray(counter, format, &bufferSize, &bufferCount, nil); err != nil {
		// From MSDN: You should call this function twice, the first time to get the required
		// buffer size (set ItemBuffer to NULL and lpdwBufferSize to 0), and the second time to get the data.
		if PdhErrno(err.(syscall.Errno)) != PDH_MORE_DATA {
			return nil, PdhErrno(err.(syscall.Errno))
		}

		// Buffer holds PdhCounterValueItems at the beginning and then null-terminated
		// strings at the end.
		buffer := make([]byte, bufferSize)
		if err := _PdhGetFormattedCounterArray(counter, format, &bufferSize, &bufferCount, &buffer[0]); err != nil {
			return nil, PdhErrno(err.(syscall.Errno))
		}

		values := make([]CounterValueItem, bufferCount)
		nameBuffer := new(bytes.Buffer)
		for i := 0; i < len(values); i++ {
			pdhValueItem := (*pdhCounterValueItem)(unsafe.Pointer(&buffer[i*sizeofPdhCounterValueItem]))

			// The strings are appended to the end of the buffer.
			nameOffset := pdhValueItem.SzName - (uintptr)(unsafe.Pointer(&buffer[0]))
			nameBuffer.Reset()
			if err := sys.UTF16ToUTF8Bytes(buffer[nameOffset:], nameBuffer); err != nil {
				return nil, err
			}

			values[i].Name = nameBuffer.String()
			values[i].Value = pdhValueItem.FmtValue
		}

		return values, nil
	}

	return nil, nil
}

func PdhGetRawCounterValue(counter PdhCounterHandle) (uint32, *PdhRawCounter, error) {
	var counterType uint32
	var value PdhRawCounter
	if err := _PdhGetRawCounterValue(counter, &counterType, &value); err != nil {
		return 0, nil, PdhErrno(err.(syscall.Errno))
	}

	return counterType, &value, nil
}

func PdhCalculateCounterFromRawValue(counter PdhCounterHandle, format PdhCounterFormat, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter) (*PdhCounterValue, error) {
	var value PdhCounterValue
	if err := _PdhCalculateCounterFromRawValue(counter, format, rawValue1, rawValue2, &value); err != nil {
		return nil, PdhErrno(err.(syscall.Errno))
	}

	return &value, nil
}

func PdhFormatFromRawValue(format PdhCounterFormat, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter) (*PdhCounterValue, error) {
	var counterType uint32
	var value PdhCounterValue
	var timeBase uint64
	if err := _PdhFormatFromRawValue(counterType, format, &timeBase, rawValue1, rawValue2, &value); err != nil {
		return nil, PdhErrno(err.(syscall.Errno))
	}

	return &value, nil
}

func PdhCloseQuery(query PdhQueryHandle) error {
	if err := _PdhCloseQuery(query); err != nil {
		return PdhErrno(err.(syscall.Errno))
	}

	return nil
}

type Counter struct {
	handle       PdhCounterHandle
	format       PdhCounterFormat
	instanceName string
	wildcard     bool // wildcard indicates that the path contains a wildcard.
}

type Counters map[string]*Counter

type Query struct {
	handle   PdhQueryHandle
	counters Counters
}

type Format int

const (
	FloatFormat Format = iota
	LongFormat
)

func NewQuery(dataSource string) (*Query, error) {
	h, err := PdhOpenQuery(dataSource, 0)
	if err != nil {
		return nil, err
	}

	return &Query{
		handle:   h,
		counters: make(Counters),
	}, nil
}

func (q *Query) AddCounter(counterPath string, format Format, instanceName string) error {
	if _, found := q.counters[counterPath]; found {
		return errors.New("counter already added")
	}

	h, err := PdhAddCounter(q.handle, counterPath, 0)
	if err != nil {
		return err
	}

	wildcard := wildcardRegexp.MatchString(counterPath)

	// Extract the instance name from the counterPath for non-wildcard paths.
	if !wildcard && instanceName == "" {
		matches := instanceNameRegexp.FindStringSubmatch(counterPath)
		if len(matches) != 2 {
			return errors.New("query doesn't contain an instance name. In this case you have to define 'instance_name'")
		}
		instanceName = matches[1]
	}

	q.counters[counterPath] = &Counter{
		handle:       h,
		instanceName: instanceName,
		wildcard:     wildcard,
	}
	switch format {
	case FloatFormat:
		q.counters[counterPath].format = PdhFmtDouble
	case LongFormat:
		q.counters[counterPath].format = PdhFmtLarge
	}
	return nil
}

func (q *Query) Execute() error {
	return PdhCollectQueryData(q.handle)
}

type Value struct {
	Instance    string
	Measurement interface{}
	Err         error
}

func (q *Query) Values() (map[string][]Value, error) {
	rtn := make(map[string][]Value, len(q.counters))

	for path, counter := range q.counters {
		if counter.wildcard {
			values, err := PdhGetFormattedCounterArray(counter.handle, counter.format|PdhFmtNoCap100)
			if err != nil {
				rtn[path] = append(rtn[path], Value{Err: err})
				continue
			}

			for i := 0; i < len(values); i++ {
				var val interface{}

				switch counter.format {
				case PdhFmtDouble:
					val = *(*float64)(unsafe.Pointer(&values[i].Value.LongValue))
				case PdhFmtLarge:
					val = *(*int64)(unsafe.Pointer(&values[i].Value.LongValue))
				}

				rtn[path] = append(rtn[path], Value{Instance: values[i].Name, Measurement: val})
			}
		} else {
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
	}

	return rtn, nil
}

// Closes the query and all of its counters.
func (q *Query) Close() error {
	return PdhCloseQuery(q.handle)
}

type PerfmonReader struct {
	query             *Query            // PDH Query
	instanceLabel     map[string]string // Mapping of counter path to key used for the label (e.g. processor.name)
	measurement       map[string]string // Mapping of counter path to key used for the value (e.g. processor.cpu_time).
	executed          bool              // Indicates if the query has been executed.
	log               *logp.Logger      //
	groupMeasurements bool              // Indicates if measurements with the same instance label should be sent in the same event
}

// NewPerfmonReader creates a new instance of PerfmonReader.
func NewPerfmonReader(config Config) (*PerfmonReader, error) {
	query, err := NewQuery("")
	if err != nil {
		return nil, err
	}

	r := &PerfmonReader{
		query:             query,
		instanceLabel:     map[string]string{},
		measurement:       map[string]string{},
		log:               logp.NewLogger("perfmon"),
		groupMeasurements: config.GroupMeasurements,
	}

	for _, counter := range config.CounterConfig {
		var format Format
		switch counter.Format {
		case "float":
			format = FloatFormat
		case "long":
			format = LongFormat
		}
		if err := query.AddCounter(counter.Query, format, counter.InstanceName); err != nil {
			if config.IgnoreNECounters {
				switch err {
				case PDH_CSTATUS_NO_COUNTER, PDH_CSTATUS_NO_COUNTERNAME,
					PDH_CSTATUS_NO_INSTANCE, PDH_CSTATUS_NO_OBJECT:
					r.log.Infow("Ignoring non existent counter", "error", err,
						logp.Namespace("perfmon"), "query", counter.Query)
					continue
				}
			}
			query.Close()
			return nil, errors.Wrapf(err, `failed to add counter (query="%v")`, counter.Query)
		}

		r.instanceLabel[counter.Query] = counter.InstanceLabel
		r.measurement[counter.Query] = counter.MeasurementLabel

	}

	return r, nil
}

func (r *PerfmonReader) Read() ([]mb.Event, error) {
	if err := r.query.Execute(); err != nil {
		return nil, errors.Wrap(err, "failed querying counter values")
	}

	// Get the values.
	values, err := r.query.Values()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}

	eventMap := make(map[string]*mb.Event)

	for counterPath, values := range values {
		for ind, val := range values {
			if val.Err != nil && !r.executed {
				r.log.Debugw("Ignoring the first measurement because the data isn't ready",
					"error", val.Err, logp.Namespace("perfmon"), "query", counterPath)
				continue
			}

			var eventKey string
			if r.groupMeasurements && val.Err == nil {
				// Send measurements with the same instance label as part of the same event
				eventKey = val.Instance
			} else {
				// Send every measurement as an individual event
				// If a counter contains an error, it will always be sent as an individual event
				eventKey = counterPath + strconv.Itoa(ind)
			}

			// Create a new event if the key doesn't exist in the map
			if _, ok := eventMap[eventKey]; !ok {
				eventMap[eventKey] = &mb.Event{
					MetricSetFields: common.MapStr{},
					Error:           errors.Wrapf(val.Err, "failed on query=%v", counterPath),
				}

				if val.Instance != "" {
					eventMap[eventKey].MetricSetFields.Put(r.instanceLabel[counterPath], val.Instance)
				}
			}

			event := eventMap[eventKey]

			if val.Measurement != nil {
				event.MetricSetFields.Put(r.measurement[counterPath], val.Measurement)
			} else {
				event.MetricSetFields.Put(r.measurement[counterPath], 0)
			}
		}
	}

	// Write the values into the map.
	events := make([]mb.Event, 0, len(eventMap))
	for _, val := range eventMap {
		events = append(events, *val)
	}

	r.executed = true
	return events, nil
}

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
