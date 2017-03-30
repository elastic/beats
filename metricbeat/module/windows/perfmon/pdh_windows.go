package perfmon

import (
	"strconv"
	"unsafe"

	"time"

	"github.com/elastic/beats/libbeat/common"
)

// Windows API calls
//sys   _PdhOpenQuery(dataSource uintptr, userData uintptr, query *uintptr) (err uint32) = pdh.PdhOpenQuery
//sys   _PdhAddCounter(query uintptr, counterPath string, userData uintptr, counter *uintptr) (err uint32) = pdh.PdhAddEnglishCounterW
//sys   _PdhCollectQueryData(query uintptr) (err uint32) = pdh.PdhCollectQueryData
//sys   _PdhGetFormattedCounterValue(counter uintptr, format uint32, counterType int, value *PdhCounterValue) (err uint32) = pdh.PdhGetFormattedCounterValue
//sys   _PdhCloseQuery(query uintptr) (err uint32) = pdh.PdhCloseQuery

type Handle struct {
	status      error
	query       uintptr
	counterType int
	counters    []CounterGroup
}

type CounterGroup struct {
	GroupName string
	Counters  []Counter
}

type Counter struct {
	counterName  string
	counter      uintptr
	counterPath  string
	displayValue PdhCounterValue
}

type PdhError uint32

var errorMapping = map[PdhError]string{
	PDH_INVALID_DATA:        `PDH_INVALID_DATA`,
	PDH_INVALID_HANDLE:      `PDH_INVALID_HANDLE`,
	PDH_NO_DATA:             `PDH_NO_DATA`,
	PDH_NO_MORE_DATA:        `PDH_NO_MORE_DATA`,
	PDH_STATUS_INVALID_DATA: `PDH_STATUS_INVALID_DATA`,
	PDH_STATUS_NEW_DATA:     `PDH_STATUS_NEW_DATA`,
	PDH_STATUS_NO_COUNTER:   `PDH_STATUS_NO_COUNTER`,
	PDH_STATUS_NO_OBJECT:    `PDH_STATUS_NO_OBJECT`,
}

func GetHandle(config []CounterConfig) (*Handle, PdhError) {
	q := &Handle{}
	err := _PdhOpenQuery(0, 0, &q.query)
	if err != ERROR_SUCCESS {
		return nil, PdhError(err)
	}

	counterGroups := make([]CounterGroup, len(config))
	q.counters = counterGroups

	for i, v := range config {
		counterGroups[i] = CounterGroup{GroupName: v.Name, Counters: make([]Counter, len(v.Group))}
		for j, v1 := range v.Group {
			counterGroups[i].Counters[j] = Counter{counterName: v1.Alias, counterPath: v1.Query}
			err := _PdhAddCounter(q.query, counterGroups[i].Counters[j].counterPath, 0, &counterGroups[i].Counters[j].counter)
			if err != ERROR_SUCCESS {
				return nil, PdhError(err)
			}
		}
	}

	return q, 0
}

func (q *Handle) ReadData(firstFetch bool) (common.MapStr, PdhError) {

	err := _PdhCollectQueryData(q.query)

	if firstFetch {
		// Most counters require two sample values in order to compute a displayable value. So wait and then collect the second value
		time.Sleep(2000)
		err = _PdhCollectQueryData(q.query)
	}

	if err != ERROR_SUCCESS {
		return nil, PdhError(err)
	}

	result := common.MapStr{}

	for _, v := range q.counters {
		groupVal := make(map[string]interface{})
		for _, v1 := range v.Counters {
			err := _PdhGetFormattedCounterValue(v1.counter, PdhFmtDouble, q.counterType, &v1.displayValue)
			if err != ERROR_SUCCESS {
				switch err {
				case PDH_CALC_NEGATIVE_DENOMINATOR:
					{
						//Sometimes counters return negative values. We can ignore this error. See here for a good explanation https://www.netiq.com/support/kb/doc.php?id=7010545
						groupVal[v1.counterName] = 0
						continue
					}
				default:
					{
						return nil, PdhError(err)
					}
				}
			}
			doubleValue := (*float64)(unsafe.Pointer(&v1.displayValue.LongValue))
			groupVal[v1.counterName] = *doubleValue

		}
		result[v.GroupName] = groupVal
	}
	return result, 0
}

func CloseQuery(q uintptr) PdhError {
	err := _PdhCloseQuery(q)
	if err != ERROR_SUCCESS {
		return PdhError(err)
	}

	return 0
}

func (e PdhError) Error() string {
	if val, ok := errorMapping[e]; ok {
		return val
	}
	return strconv.FormatUint(uint64(e), 10)
}
