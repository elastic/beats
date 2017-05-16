package state_container

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
			container := util.GetLabel(metric, "container")
			if container == "" {
				continue
			}
			event, ok := eventsMap[container]
			if !ok {
				event = common.MapStr{}
				eventsMap[container] = event
			}
			switch family.GetName() {
			case "kube_pod_container_info":
				event.Put(mb.ModuleDataKey+".pod.name", util.GetLabel(metric, "pod"))
				event.Put(mb.ModuleDataKey+".namespace", util.GetLabel(metric, "namespace"))
				event.Put(mb.NamespaceKey, "container")

				event.Put("name", util.GetLabel(metric, "container"))
				event.Put("id", util.GetLabel(metric, "container_id"))
				event.Put("image", util.GetLabel(metric, "image"))

			case "kube_pod_container_resource_limits_cpu_cores":
				event.Put(mb.ModuleDataKey+".node.name", util.GetLabel(metric, "node"))
				event.Put("cpu.limit.nanocores", metric.GetGauge().GetValue()*nanocores)

			case "kube_pod_container_resource_requests_cpu_cores":
				event.Put(mb.ModuleDataKey+".node.name", util.GetLabel(metric, "node"))
				event.Put("cpu.request.nanocores", metric.GetGauge().GetValue()*nanocores)

			case "kube_pod_container_resource_limits_memory_bytes":
				event.Put(mb.ModuleDataKey+".node.name", util.GetLabel(metric, "node"))
				event.Put("memory.limit.bytes", metric.GetGauge().GetValue())

			case "kube_pod_container_resource_requests_memory_bytes":
				event.Put(mb.ModuleDataKey+".node.name", util.GetLabel(metric, "node"))
				event.Put("memory.request.bytes", metric.GetGauge().GetValue())

			case "kube_pod_container_status_ready":
				event.Put("status.ready", metric.GetGauge().GetValue() == 1)

			case "kube_pod_container_status_restarts":
				event.Put("status.restarts", metric.GetCounter().GetValue())

			case "kube_pod_container_status_running":
				if metric.GetGauge().GetValue() == 1 {
					event.Put("status.phase", "running")
				}

			case "kube_pod_container_status_terminated":
				if metric.GetGauge().GetValue() == 1 {
					event.Put("status.phase", "terminate")
				}

			case "kube_pod_container_status_waiting":
				if metric.GetGauge().GetValue() == 1 {
					event.Put("status.phase", "waiting")
				}

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
