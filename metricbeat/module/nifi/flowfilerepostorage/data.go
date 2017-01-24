package flowfilerepostorage

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// FlowFileRepositoryStorageUsage ...
type FlowFileRepositoryStorageUsage struct {
	FreeSpace       string `json:"freeSpace"`
	FreeSpaceBytes  int64  `json:"freeSpaceBytes"`
	TotalSpace      string `json:"totalSpace"`
	TotalSpaceBytes int64  `json:"totalSpaceBytes"`
	UsedSpace       string `json:"usedSpace"`
	UsedSpaceBytes  int64  `json:"usedSpaceBytes"`
	Utilization     string `json:"utilization"`
}

// SystemDiagnosticsResponse ...
type SystemDiagnosticsResponse struct {
	SystemDiagnostics struct {
		AggregateSnapshot struct {
			FlowFileRepositoryStorageUsage `json:"flowFileRepositoryStorageUsage"`
		} `json:"aggregateSnapshot"`

		NodeSnapshots []struct {
			NodeID   string `json:"nodeId"`
			Address  string `json:"address"`
			Snapshot struct {
				FlowFileRepositoryStorageUsage `json:"flowFileRepositoryStorageUsage"`
			} `json:"snapshot"`
		} `json:"nodeSnapshots"`
	} `json:"systemDiagnostics"`
}

func nodewiseEventMapping(body io.Reader, nodeID string) (common.MapStr, error) {
	var data SystemDiagnosticsResponse
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	snapshots := data.SystemDiagnostics.NodeSnapshots
	var usage FlowFileRepositoryStorageUsage

	for i, snapshot := range snapshots {
		if snapshot.NodeID == nodeID {
			usage = snapshot.Snapshot.FlowFileRepositoryStorageUsage
			break
		}

		if i == len(snapshots)-1 {
			return nil, errors.New("Failed to find data for specific nodeID")
		}
	}

	return mapFields(usage), nil
}

func aggregateEventMapping(body io.Reader) common.MapStr {
	var data SystemDiagnosticsResponse
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	usage := data.SystemDiagnostics.AggregateSnapshot.FlowFileRepositoryStorageUsage

	return mapFields(usage)
}

func mapFields(usage FlowFileRepositoryStorageUsage) common.MapStr {

	event := common.MapStr{
		"free_space":        usage.FreeSpace,
		"free_space_bytes":  usage.FreeSpaceBytes,
		"total_space":       usage.TotalSpace,
		"total_space_bytes": usage.TotalSpaceBytes,
		"used_space":        usage.UsedSpace,
		"used_space_bytes":  usage.UsedSpaceBytes,
		"utilization":       usage.Utilization,
	}

	return event
}
