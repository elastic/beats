package state_deployment

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"

	dto "github.com/prometheus/client_model/go"
)

const (
	// Nanocores conversion 10^9
	nanocores = 1000000000
)

func eventMapping(families []*dto.MetricFamily) ([]common.MapStr, error) {
	eventsMap := map[string]common.MapStr{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			deployment := util.GetLabel(metric, "deployment")
			if deployment == "" {
				continue
			}
			namespace := util.GetLabel(metric, "namespace")
			deploymentKey := namespace + "::" + deployment
			event, ok := eventsMap[deploymentKey]
			if !ok {
				event = common.MapStr{}
				eventsMap[deploymentKey] = event
			}

			switch family.GetName() {
			case "kube_deployment_metadata_generation":
				event.Put(mb.ModuleDataKey+".namespace", util.GetLabel(metric, "namespace"))
				event.Put(mb.NamespaceKey, "deployment")
				event.Put("name", util.GetLabel(metric, "deployment"))

			case "kube_deployment_spec_paused":
				event.Put("paused", metric.GetGauge().GetValue() == 1)

			case "kube_deployment_spec_replicas":
				event.Put("replicas.desired", metric.GetGauge().GetValue())

			case "kube_deployment_status_replicas_available":
				event.Put("replicas.available", metric.GetGauge().GetValue())

			case "kube_deployment_status_replicas_unavailable":
				event.Put("replicas.unavailable", metric.GetGauge().GetValue())

			case "kube_deployment_status_replicas_updated":
				event.Put("replicas.updated", metric.GetGauge().GetValue())

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
