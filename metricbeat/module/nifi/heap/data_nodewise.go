package heap

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// NodeHeapSnapshot ...
type NodeHeapSnapshot struct {
	NodeID   string `json:"nodeId"`
	Address  string `json:"address"`
	Snapshot struct {
		Heap
	} `json:"snapshot"`
}

// NodewiseHeapSnapshot ...
type NodewiseHeapSnapshot struct {
	SystemDiagnostics struct {
		NodeSnapshots []NodeHeapSnapshot `json:"nodeSnapshots"`
	}
}

func nodewiseEventMapping(body io.Reader, nodeID string) (common.MapStr, error) {
	var data NodewiseHeapSnapshot
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
