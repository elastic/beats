package perfmon

import (
	"unsafe"

	"github.com/elastic/beats/libbeat/common"
)

type Handle struct {
	Status      error
	Query       uintptr
	CounterType int
	Counters    []CounterGroup
}

type CounterGroup struct {
	GroupName string
	Counters  []Counter
}

type Counter struct {
	CounterName  string
	Counter      uintptr
	CounterPath  string
	DisplayValue PdhCounterValue
}

func getHandle(config []CounterConfig) (*Handle, int) {
	q := &Handle{}
	err := _PdhOpenQuery(0, 0, &q.Query)
	if err != ERROR_SUCCESS {
		return nil, err
	}

	counterGroups := make([]CounterGroup, len(config))
	q.Counters = counterGroups

	for i, v := range config {
		counterGroups[i] = CounterGroup{GroupName: v.Name, Counters: make([]Counter, len(v.Group))}
		for j, v1 := range v.Group {
			counterGroups[i].Counters[j] = Counter{CounterName: v1.Alias, CounterPath: v1.Query}
			err := _PdhAddCounter(q.Query, counterGroups[i].Counters[j].CounterPath, 0, &counterGroups[i].Counters[j].Counter)
			if err != ERROR_SUCCESS {
				return q, err
			}
		}
	}

	return q, 0
}

func (q *Handle) readData() (common.MapStr, int) {

	err := _PdhCollectQueryData(q.Query)

	if err != ERROR_SUCCESS {
		return nil, err
	}

	result := common.MapStr{}

	for _, v := range q.Counters {
		groupVal := make(map[string]interface{})
		for _, v1 := range v.Counters {
			err := _PdhGetFormattedCounterValue(v1.Counter, PdhFmtDouble, q.CounterType, &v1.DisplayValue)
			if err != ERROR_SUCCESS {
				return nil, err
			}
			doubleValue := (*float64)(unsafe.Pointer(&v1.DisplayValue.LongValue))
			groupVal[v1.CounterName] = *doubleValue

		}
		result[v.GroupName] = groupVal
	}
	return result, 0
}

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output pdh_windows.go pdh.go
// Windows API calls
//sys   _PdhOpenQuery(dataSource uintptr, userData uintptr, query *uintptr) (err int) = pdh.PdhOpenQuery
//sys   _PdhAddCounter(query uintptr, counterPath string, userData uintptr, counter *uintptr) (err int) = pdh.PdhAddEnglishCounterW
//sys   _PdhCollectQueryData(query uintptr) (err int) = pdh.PdhCollectQueryData
//sys   _PdhGetFormattedCounterValue(counter uintptr, format uint32, counterType int, value *PdhCounterValue) (err int) = pdh.PdhGetFormattedCounterValue
