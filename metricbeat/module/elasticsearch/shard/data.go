package shard

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/metricbeat/mb"
)

var (
	schema = s.Schema{
		"state":           c.Str("state"),
		"primary":         c.Bool("primary"),
		"node":            c.Str("node"),
		"index":           c.Str("index"),
		"shard":           c.Int("number"),
		"relocating_node": c.Str("relocating_node"),
	}
)

type stateStruct struct {
	ClusterName  string `json:"cluster_name"`
	StateID      string `json:"state_uuid"`
	MasterNode   string `json:"master_node"`
	RoutingTable struct {
		Indices map[string]struct {
			Shards map[string][]map[string]interface{} `json:"shards"`
		} `json:"indices"`
	} `json:"routing_table"`
}

func eventsMapping(r mb.ReporterV2, content []byte) {
	stateData := &stateStruct{}
	err := json.Unmarshal(content, stateData)
	if err != nil {
		r.Error(err)
		return
	}

	for _, index := range stateData.RoutingTable.Indices {
		for _, shards := range index.Shards {
			for _, shard := range shards {
				event := mb.Event{}

				fields, _ := schema.Apply(shard)

				event.ModuleFields = common.MapStr{}
				event.ModuleFields.Put("node.name", fields["node"])
				delete(fields, "node")
				event.ModuleFields.Put("index.name", fields["index"])
				delete(fields, "index")
				event.MetricSetFields = fields
				event.ModuleFields.Put("cluster.state.id", stateData.StateID)
				event.ModuleFields.Put("cluster.name", stateData.ClusterName)

				event.RootFields = common.MapStr{}
				event.RootFields.Put("service.name", "elasticsearch")

				r.Event(event)
			}
		}
	}
}
