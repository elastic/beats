package state_statefulset

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
			statefulset := util.GetLabel(metric, "statefulset")
			if statefulset == "" {
				continue
			}
			namespace := util.GetLabel(metric, "namespace")
			statefulsetKey := namespace + "::" + statefulset
			event, ok := eventsMap[statefulsetKey]
			if !ok {
				event = common.MapStr{}
				eventsMap[statefulsetKey] = event
			}
			switch family.GetName() {
			case "kube_statefulset_metadata_generation":
				event.Put(mb.ModuleDataKey+".namespace", util.GetLabel(metric, "namespace"))
				event.Put(mb.NamespaceKey, "statefulset")
				event.Put("name", util.GetLabel(metric, "statefulset"))
				event.Put("generation.desired", metric.GetGauge().GetValue())
			case "kube_statefulset_status_observed_generation":
				event.Put("generation.observed", metric.GetGauge().GetValue())
			case "kube_statefulset_created":
				event.Put("created", metric.GetGauge().GetValue())
			case "kube_statefulset_replicas":
				event.Put("replicas.desired", metric.GetGauge().GetValue())
			case "kube_statefulset_status_replicas":
				event.Put("replicas.observed", metric.GetGauge().GetValue())
			default:
				// Ignore unknown metric
				continue
			}
		}
	}

	// initialize, populate events array from values in eventsMap
	events := make([]common.MapStr, 0, len(eventsMap))
	for _, event := range eventsMap {
		events = append(events, event)
	}
	return events, nil
}
