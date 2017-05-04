// +build windows

package perfmon

import (
	"strconv"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/elastic/beats/libbeat/common"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
)

// Windows API calls
//sys _PdhOpenQuery(dataSource *uint16, userData uintptr, query *PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhOpenQueryW
//sys _PdhAddCounter(query PdhQueryHandle, counterPath string, userData uintptr, counter *PdhCounterHandle) (errcode error) [failretval!=0] = pdh.PdhAddEnglishCounterW
//sys _PdhCollectQueryData(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCollectQueryData
//sys _PdhGetFormattedCounterValue(counter PdhCounterHandle, format PdhCounterFormat, counterType *uint32, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhGetFormattedCounterValue
//sys _PdhGetRawCounterValue(counter PdhCounterHandle, counterType *uint32, value *PdhRawCounter) (errcode error) [failretval!=0] = pdh.PdhGetRawCounterValue
//sys _PdhCalculateCounterFromRawValue(counter PdhCounterHandle, format PdhCounterFormat, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhCalculateCounterFromRawValue
//sys _PdhFormatFromRawValue(counterType uint32, format PdhCounterFormat, timeBase *uint64, rawValue1 *PdhRawCounter, rawValue2 *PdhRawCounter, value *PdhCounterValue) (errcode error) [failretval!=0] = pdh.PdhFormatFromRawValue
//sys _PdhCloseQuery(query PdhQueryHandle) (errcode error) [failretval!=0] = pdh.PdhCloseQuery

type PdhQueryHandle uintptr

var InvalidQueryHandle = ^PdhQueryHandle(0)

type PdhCounterHandle uintptr

var InvalidCounterHandle = ^PdhCounterHandle(0)

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
	handle PdhCounterHandle
	format PdhCounterFormat
}

type Counters map[string]*Counter

type Query struct {
	handle   PdhQueryHandle
	counters Counters
}

type Format int

const (
	FloatFlormat Format = iota
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

func (q *Query) AddCounter(counterPath string, format Format) error {
	if _, found := q.counters[counterPath]; found {
		return errors.New("counter already added")
	}

	h, err := PdhAddCounter(q.handle, counterPath, 0)
	if err != nil {
		return errors.Wrapf(err, `failed to add counter (path="%v")`, counterPath)
	}

	q.counters[counterPath] = &Counter{handle: h}
	switch format {
	case FloatFlormat:
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
	Num interface{}
	Err error
}

func (q *Query) Values() (map[string]Value, error) {
	rtn := make(map[string]Value, len(q.counters))
	for path, counter := range q.counters {
		_, value, err := PdhGetFormattedCounterValue(counter.handle, counter.format|PdhFmtNoCap100)
		if err != nil {
			rtn[path] = Value{Err: err}
			continue
		}

		switch counter.format {
		case PdhFmtDouble:
			rtn[path] = Value{Num: *(*float64)(unsafe.Pointer(&value.LongValue))}
		case PdhFmtLarge:
			rtn[path] = Value{Num: *(*int64)(unsafe.Pointer(&value.LongValue))}
		}

	}

	return rtn, nil
}

// Closes the query and all of its counters.
func (q *Query) Close() error {
	return PdhCloseQuery(q.handle)
}

type PerfmonReader struct {
	query     *Query            // PDH Query
	pathToKey map[string]string // Mapping of counter path to key used in output.
	executed  bool              // Indicates if the query has been executed.
}

func NewPerfmonReader(config []CounterConfig) (*PerfmonReader, error) {
	query, err := NewQuery("")
	if err != nil {
		return nil, err
	}

	r := &PerfmonReader{
		query:     query,
		pathToKey: map[string]string{},
	}

	for _, counter := range config {
		var format Format
		switch counter.Format {
		case "float":
			format = FloatFlormat
		case "long":
			format = LongFormat
		}
		if err := query.AddCounter(counter.Query, format); err != nil {
			query.Close()
			return nil, err
		}

		r.pathToKey[counter.Query] = counter.Alias

	}

	return r, nil
}

func (r *PerfmonReader) Read() (common.MapStr, error) {
	if err := r.query.Execute(); err != nil {
		return nil, err
	}

	// Get the values.
	values, err := r.query.Values()
	if err != nil {
		return nil, errors.Wrap(err, "failed formatting counter values")
	}

	// Write the values into the map.
	result := common.MapStr{}
	var errs multierror.Errors

	for counterPath, value := range values {
		key := r.pathToKey[counterPath]
		result.Put(key, value.Num)

		if value.Err != nil {
			switch value.Err {
			case PDH_CALC_NEGATIVE_DENOMINATOR:
			case PDH_INVALID_DATA:
				if r.executed {
					errs = append(errs, errors.Wrapf(value.Err, "key=%v", key))
				}
			default:
				errs = append(errs, errors.Wrapf(value.Err, "key=%v", key))
			}
		}
	}

	if !r.executed {
		r.executed = true
	}

	return result, errs.Err()
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
