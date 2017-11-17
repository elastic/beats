package state_replicaset

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"

	dto "github.com/prometheus/client_model/go"
)

func eventMapping(families []*dto.MetricFamily) ([]common.MapStr, error) {
	eventsMap := map[string]common.MapStr{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			replicaset := util.GetLabel(metric, "replicaset")
			if replicaset == "" {
				continue
			}
			event, ok := eventsMap[replicaset]
			if !ok {
				event = common.MapStr{}
				eventsMap[replicaset] = event
			}
			switch family.GetName() {
			case "kube_replicaset_metadata_generation":
				event.Put(mb.ModuleDataKey+".namespace", util.GetLabel(metric, "namespace"))
				event.Put(mb.NamespaceKey, "replicaset")

				event.Put("name", util.GetLabel(metric, "replicaset"))

			case "kube_replicaset_status_replicas":
				event.Put("replicas.available", metric.GetGauge().GetValue())

			case "kube_replicaset_spec_replicas":
				event.Put("replicas.desired", metric.GetGauge().GetValue())

			case "kube_replicaset_status_ready_replicas":
				event.Put("replicas.ready", metric.GetGauge().GetValue())

			case "kube_replicaset_status_observed_generation":
				event.Put("replicas.observed", metric.GetGauge().GetValue())

			case "kube_replicaset_status_fully_labeled_replicas":
				event.Put("replicas.labeled", metric.GetGauge().GetValue())

			default:
				// Ignore unknown metric
				continue
			}
		}
	}

	var events []common.MapStr
	for _, event := range eventsMap {
		events = append(events, event)
	}
	return events, nil
}
