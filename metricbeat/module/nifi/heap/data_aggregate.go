package heap

import (
	"encoding/json"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// AggregateHeapSnapshot ...
type AggregateHeapSnapshot struct {
	SystemDiagnostics struct {
		AggregateSnapshot struct {
			Heap
		} `json:"aggregateSnapshot"`
	} `json:"systemDiagnostics"`
}

func aggregateEventMapping(body io.Reader) common.MapStr {
	var data AggregateHeapSnapshot
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	slice := data.SystemDiagnostics.AggregateSnapshot

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

	return event
}
