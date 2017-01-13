package flowfilerepostorage

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// FlowFileRepositoryStorageUsage ...
type FlowFileRepositoryStorageUsage struct {
	FreeSpace       string `json:"freeSpace"`
	FreeSpaceBytes  uint64 `json:"freeSpaceBytes"`
	TotalSpace      string `json:"totalSpace"`
	TotalSpaceBytes uint64 `json:"totalSpaceBytes"`
	UsedSpace       string `json:"usedSpace"`
	UsedSpaceBytes  uint64 `json:"usedSpaceBytes"`
	Utilization     string `json:"utilization"`
}

// Data ...
type Data struct {
	SystemDiagnostics struct {
		AggregateSnapshot struct {
			FlowFileRepositoryStorageUsage `json:"flowFileRepositoryStorageUsage"`
		} `json:"aggregateSnapshot"`
	} `json:"systemDiagnostics"`
}

func eventMapping(body io.Reader) common.MapStr {
	var data Data
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
