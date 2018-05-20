package shard

import (
	"encoding/json"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func eventsMappingXPack(r mb.ReporterV2, m *MetricSet, content []byte) {
	stateData := &stateStruct{}
	err := json.Unmarshal(content, stateData)
	if err != nil {
		r.Error(err)
		return
	}

	nodeInfo, err := elasticsearch.GetNodeInfo(m.HTTP, m.HostData().SanitizedURI+statePath, stateData.MasterNode)
	if err != nil {
		r.Error(err)
		return
	}

	// TODO: This is currently needed because the cluser_uuid is `na` in stateData in case not the full state is requested.
	// Will be fixed in: https://github.com/elastic/elasticsearch/pull/30656
	clusterID, err := elasticsearch.GetClusterID(m.HTTP, m.HostData().SanitizedURI+statePath, stateData.MasterNode)
	if err != nil {
		r.Error(err)
		return
	}

	sourceNode := common.MapStr{
		"uuid":              stateData.MasterNode,
		"host":              nodeInfo.Host,
		"transport_address": nodeInfo.TransportAddress,
		"ip":                nodeInfo.IP,
		// This seems to be in the x-pack data a subset of the cluster_uuid not the name?
		"name":      stateData.ClusterName,
		"timestamp": common.Time(time.Now()),
	}

	for _, index := range stateData.RoutingTable.Indices {
		for _, shards := range index.Shards {
			for _, shard := range shards {
				event := mb.Event{}
				fields, _ := schema.Apply(shard)

				fields["shard"] = fields["number"]
				delete(fields, "number")

				event.RootFields = common.MapStr{}

				event.RootFields = common.MapStr{
					"timestamp":    time.Now(),
					"cluster_uuid": clusterID,
					"interval_ms":  m.Module().Config().Period.Nanoseconds() / 1000 / 1000,
					"type":         "shards",
					"source_node":  sourceNode,
					"shard":        fields,
					"state_uuid":   stateData.StateID,
				}
				event.Index = ".monitoring-es-6-mb"

				r.Event(event)

			}
		}
	}
}
