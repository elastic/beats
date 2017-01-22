package flowfilerepostorage

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// NodeFlowFileRepositoryStorageSnapshot ...
type NodeFlowFileRepositoryStorageSnapshot struct {
	NodeID   string `json:"nodeId"`
	Address  string `json:"address"`
	Snapshot struct {
		FlowFileRepositoryStorageUsage `json:"flowFileRepositoryStorageUsage"`
	} `json:"snapshot"`
}

// NodewiseFlowFileRepositoryStorageSnapshot ...
type NodewiseFlowFileRepositoryStorageSnapshot struct {
	SystemDiagnostics struct {
		NodeSnapshots []NodeFlowFileRepositoryStorageSnapshot `json:"nodeSnapshots"`
	}
}

func nodewiseEventMapping(body io.Reader, nodeID string) (common.MapStr, error) {
	var data NodewiseFlowFileRepositoryStorageSnapshot
	err := json.NewDecoder(body).Decode(&data)
	if err != nil {
		logp.Err("Error: ", err)
	}

	snapshots := data.SystemDiagnostics.NodeSnapshots
	var slice FlowFileRepositoryStorageUsage

	for i, snapshot := range snapshots {
		if snapshot.NodeID == nodeID {
			slice = snapshot.Snapshot.FlowFileRepositoryStorageUsage
			break
		}

		if i == len(snapshots)-1 {
			return nil, errors.New("Failed to find data for specific nodeID")
		}
	}

	event := common.MapStr{
		"free_space":        slice.FreeSpace,
		"free_space_bytes":  slice.FreeSpaceBytes,
		"total_space":       slice.TotalSpace,
		"total_space_bytes": slice.TotalSpaceBytes,
		"used_space":        slice.UsedSpace,
		"used_space_bytes":  slice.UsedSpaceBytes,
		"utilization":       slice.Utilization,
	}

	return event, nil
}
