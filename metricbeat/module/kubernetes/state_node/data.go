package state_node

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
			node := util.GetLabel(metric, "node")
			if node == "" {
				continue
			}
			event, ok := eventsMap[node]
			if !ok {
				event = common.MapStr{}
				eventsMap[node] = event
			}
			switch family.GetName() {
			case "kube_node_info":
				event.Put(mb.NamespaceKey, "node")
				event.Put("name", util.GetLabel(metric, "node"))

			case "kube_node_status_allocatable_cpu_cores":
				event.Put("cpu.allocatable.cores", metric.GetGauge().GetValue())

			case "kube_node_status_capacity_cpu_cores":
				event.Put("cpu.capacity.cores", metric.GetGauge().GetValue())

			case "kube_node_status_allocatable_memory_bytes":
				event.Put("memory.allocatable.bytes", metric.GetGauge().GetValue())

			case "kube_node_status_capacity_memory_bytes":
				event.Put("memory.capacity.bytes", metric.GetGauge().GetValue())

			case "kube_node_status_capacity_pods":
				event.Put("pod.capacity.total", metric.GetGauge().GetValue())

			case "kube_node_status_allocatable_pods":
				event.Put("pod.allocatable.total", metric.GetGauge().GetValue())

			case "kube_node_status_ready":
				if metric.GetGauge().GetValue() == 1 {
					event.Put("status.ready", util.GetLabel(metric, "condition"))
				}

			case "kube_node_spec_unschedulable":
				event.Put("status.unschedulable", metric.GetGauge().GetValue() == 1)

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
