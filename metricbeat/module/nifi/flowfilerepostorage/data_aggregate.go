package flowfilerepostorage

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// AggregateFlowFileRepositoryStorageUsage ...
type AggregateFlowFileRepositoryStorageUsage struct {
	SystemDiagnostics struct {
		AggregateSnapshot struct {
			FlowFileRepositoryStorageUsage `json:"flowFileRepositoryStorageUsage"`
		} `json:"aggregateSnapshot"`
	} `json:"systemDiagnostics"`
}

func aggregateEventMapping(body io.Reader) common.MapStr {
	var data AggregateFlowFileRepositoryStorageUsage
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	slice := data.SystemDiagnostics.AggregateSnapshot.FlowFileRepositoryStorageUsage

	fmt.Printf("%v", data)
	fmt.Printf("%v", slice)

	event := common.MapStr{
		"free_space":        slice.FreeSpace,
		"free_space_bytes":  slice.FreeSpaceBytes,
		"total_space":       slice.TotalSpace,
		"total_space_bytes": slice.TotalSpaceBytes,
		"used_space":        slice.UsedSpace,
		"used_space_bytes":  slice.UsedSpaceBytes,
		"utilization":       slice.Utilization,
	}

	return event
}
