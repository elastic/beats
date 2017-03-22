package perfmon

import (
	"unsafe"

	"time"

	"github.com/elastic/beats/libbeat/common"
)

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

func GetHandle(config []CounterConfig) (*Handle, uint32) {
	q := &Handle{}
	err := _PdhOpenQuery(0, 0, &q.query)
	if err != ERROR_SUCCESS {
		return nil, err
	}

	counterGroups := make([]CounterGroup, len(config))
	q.counters = counterGroups

	for i, v := range config {
		counterGroups[i] = CounterGroup{GroupName: v.Name, Counters: make([]Counter, len(v.Group))}
		for j, v1 := range v.Group {
			counterGroups[i].Counters[j] = Counter{counterName: v1.Alias, counterPath: v1.Query}
			err := _PdhAddCounter(q.query, counterGroups[i].Counters[j].counterPath, 0, &counterGroups[i].Counters[j].counter)
			if err != ERROR_SUCCESS {
				return nil, err
			}
		}
	}

	return q, 0
}

func (q *Handle) ReadData(firstFetch bool) (common.MapStr, uint32) {

	err := _PdhCollectQueryData(q.query)

	if firstFetch {
		// Most counters require two sample values in order to compute a displayable value. So wait and then collect the second value
		time.Sleep(2000)
		err = _PdhCollectQueryData(q.query)
	}

	if err != ERROR_SUCCESS {
		return nil, err
	}

	result := common.MapStr{}

	for _, v := range q.counters {
		groupVal := make(map[string]interface{})
		for _, v1 := range v.Counters {
			err := _PdhGetFormattedCounterValue(v1.counter, PdhFmtDouble, q.counterType, &v1.displayValue)
			if err != ERROR_SUCCESS {
				return nil, err
			}
			doubleValue := (*float64)(unsafe.Pointer(&v1.displayValue.LongValue))
			groupVal[v1.counterName] = *doubleValue

		}
		result[v.GroupName] = groupVal
	}
	return result, 0
}

func CloseQuery(q uintptr) uint32 {
	err := _PdhCloseQuery(q)
	if err != ERROR_SUCCESS {
		return err
	}

	return 0
}

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output syscall_windows.go pdh.go
// Windows API calls
//sys   _PdhOpenQuery(dataSource uintptr, userData uintptr, query *uintptr) (err uint32) = pdh.PdhOpenQuery
//sys   _PdhAddCounter(query uintptr, counterPath string, userData uintptr, counter *uintptr) (err uint32) = pdh.PdhAddEnglishCounterW
//sys   _PdhCollectQueryData(query uintptr) (err uint32) = pdh.PdhCollectQueryData
//sys   _PdhGetFormattedCounterValue(counter uintptr, format uint32, counterType int, value *PdhCounterValue) (err uint32) = pdh.PdhGetFormattedCounterValue
//sys   _PdhCloseQuery(query uintptr) (err uint32) = pdh.PdhCloseQuery
