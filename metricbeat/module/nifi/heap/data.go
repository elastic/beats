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

// SystemDiagnosticsResponse ...
type SystemDiagnosticsResponse struct {
	SystemDiagnostics struct {
		AggregateSnapshot struct {
			Heap
		} `json:"aggregateSnapshot"`

		NodeSnapshots []struct {
			NodeID   string `json:"nodeId"`
			Address  string `json:"address"`
			Snapshot struct {
				Heap
			} `json:"snapshot"`
		} `json:"nodeSnapshots"`
	} `json:"systemDiagnostics"`
}

func aggregateEventMapping(body io.Reader) common.MapStr {
	var data SystemDiagnosticsResponse
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}
	heap := data.SystemDiagnostics.AggregateSnapshot.Heap

	return mapFields(heap)
}

func nodewiseEventMapping(body io.Reader, nodeID string) (common.MapStr, error) {
	var data SystemDiagnosticsResponse
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	snapshots := data.SystemDiagnostics.NodeSnapshots
	var heap Heap

	for i, snapshot := range snapshots {
		if snapshot.NodeID == nodeID {
			heap = snapshot.Snapshot.Heap
			break
		}
		if i == len(snapshots)-1 {
			return nil, errors.New("Failed to find data for specific nodeID")
		}
	}

	return mapFields(heap), nil
}

func mapFields(heap Heap) common.MapStr {
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
