package osd_df

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type Node struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Used        int64  `json:"kb_used"`
	Available   int64  `json:"kb_avail"`
	Total       int64  `json:"kb"`
	PgNum       int64  `json:"pgs"`
	DeviceClass string `json:"device_class"`
}

type Output struct {
	Nodes []Node `json:"nodes"`
}

type OsdDfRequest struct {
	Status string `json:"status"`
	Output Output `json:"output"`
}

func eventsMapping(content []byte) ([]common.MapStr, error) {
	var d OsdDfRequest
	err := json.Unmarshal(content, &d)
	if err != nil {
		logp.Err("Error: ", err)
		return nil, err
	}

	nodeList := d.Output.Nodes

	//osd node list
	events := []common.MapStr{}
	for _, node := range nodeList {
		nodeInfo := common.MapStr{}
		nodeInfo["id"] = node.ID
		nodeInfo["name"] = node.Name
		nodeInfo["total_byte"] = node.Total
		nodeInfo["used_byte"] = node.Used
		nodeInfo["avail_byte"] = node.Available
		nodeInfo["device_class"] = node.DeviceClass
		nodeInfo["pg_num"] = node.PgNum

		if 0 != node.Total {
			var usedPct float64
			usedPct = float64(node.Used) / float64(node.Total)
			nodeInfo["used_pct"] = usedPct
		}

		events = append(events, nodeInfo)
	}

	return events, nil
}
