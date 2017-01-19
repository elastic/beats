package heap

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Heap ...
type Heap struct {
	TotalNonHeap      string `json:"totalNonHeap"`
	TotalNonHeapBytes int64  `json:"totalNonHeapBytes"`
	UsedNonHeap       string `json:"usedNonHeap"`
	UsedNonHeapBytes  int64  `json:"usedNonHeapBytes"`
	FreeNonHeap       string `json:"freeNonHeap"`
	FreeNonHeapBytes  int64  `json:"freeNonHeapBytes"`
	MaxNonHeap        string `json:"maxNonHeap"`
	MaxNonHeapBytes   int64  `json:"maxNonHeapBytes"`
	TotalHeap         string `json:"totalHeap"`
	TotalHeapBytes    int64  `json:"totalHeapBytes"`
	UsedHeap          string `json:"usedHeap"`
	UsedHeapBytes     int64  `json:"usedHeapBytes"`
	FreeHeap          string `json:"freeHeap"`
	FreeHeapBytes     int64  `json:"freeHeapBytes"`
	MaxHeap           string `json:"maxHeap"`
	MaxHeapBytes      int64  `json:"maxHeapBytes"`
	HeapUtilization   string `json:"heapUtilization"`
}

// Data ...
type Data struct {
	SystemDiagnostics struct {
		AggregateSnapshot struct {
			Heap
		} `json:"aggregateSnapshot"`
	} `json:"systemDiagnostics"`
}

func eventMapping(body io.Reader) common.MapStr {
	var data Data
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	heap := data.SystemDiagnostics.AggregateSnapshot

	event := common.MapStr{
		"total_non_heap":       heap.TotalNonHeap,
		"total_non_heap_bytes": heap.TotalNonHeapBytes,
		"used_non_heap":        heap.UsedNonHeap,
		"used_non_heap_bytes":  heap.UsedNonHeapBytes,
		"free_non_heap":        heap.FreeNonHeap,
		"free_non_heap_bytes":  heap.FreeNonHeapBytes,
		"max_non_heap":         heap.MaxNonHeap,
		"max_non_heap_bytes":   heap.MaxNonHeapBytes,
		"total_heap":           heap.TotalHeap,
		"total_heap_bytes":     heap.TotalHeapBytes,
		"used_heap":            heap.UsedHeap,
		"used_heap_bytes":      heap.UsedHeapBytes,
		"free_heap":            heap.FreeHeap,
		"free_heap_bytes":      heap.FreeHeapBytes,
		"max_heap":             heap.MaxHeap,
		"max_heap_bytes":       heap.MaxHeapBytes,
		"heap_utilization":     heap.HeapUtilization,
	}

	return event
}

// NodeSnapshot ...
type NodeSnapshot struct {
	NodeID   string `json:"nodeId"`
	Address  string `json:"address"`
	Snapshot struct {
		Heap
	} `json:"snapshot"`
}

// NodewiseData ...
type NodewiseData struct {
	SystemDiagnostics struct {
		NodeSnapshots []NodeSnapshot `json:"nodeSnapshots"`
	}
}

func nodewiseEventMapping(body io.Reader, nodeID string) (common.MapStr, error) {
	var data NodewiseData
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	snapshots := data.SystemDiagnostics.NodeSnapshots
	var slice Heap

	for i, snapshot := range snapshots {
		if snapshot.NodeID == nodeID {
			slice = snapshot.Snapshot.Heap
			break
		}
		if i == len(snapshots)-1 {
			return nil, errors.New("Failed to find data for specific nodeID")
		}
	}

	event := common.MapStr{
		"total_non_heap":       slice.TotalNonHeap,
		"total_non_heap_bytes": slice.TotalNonHeapBytes,
		"used_non_heap":        slice.UsedNonHeap,
		"used_non_heap_bytes":  slice.UsedNonHeapBytes,
		"free_non_heap":        slice.FreeNonHeap,
		"free_non_heap_bytes":  slice.FreeNonHeapBytes,
		"max_non_heap":         slice.MaxNonHeap,
		"max_non_heap_bytes":   slice.MaxNonHeapBytes,
		"total_heap":           slice.TotalHeap,
		"total_heap_bytes":     slice.TotalHeapBytes,
		"used_heap":            slice.UsedHeap,
		"used_heap_bytes":      slice.UsedHeapBytes,
		"free_heap":            slice.FreeHeap,
		"free_heap_bytes":      slice.FreeHeapBytes,
		"max_heap":             slice.MaxHeap,
		"max_heap_bytes":       slice.MaxHeapBytes,
		"heap_utilization":     slice.HeapUtilization,
	}

	return event, nil
}
