package state_pod

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kubernetes/util"

	dto "github.com/prometheus/client_model/go"
)

func eventMapping(families []*dto.MetricFamily) ([]common.MapStr, error) {
	eventsMap := map[string]common.MapStr{}
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			pod := util.GetLabel(metric, "pod")
			if pod == "" {
				continue
			}
			event, ok := eventsMap[pod]
			if !ok {
				event = common.MapStr{}
				eventsMap[pod] = event
			}
			switch family.GetName() {
			case "kube_pod_info":
				event.Put(mb.ModuleDataKey+".node.name", util.GetLabel(metric, "node"))
				event.Put(mb.ModuleDataKey+".namespace", util.GetLabel(metric, "namespace"))
				event.Put(mb.NamespaceKey, "pod")

				event.Put("name", util.GetLabel(metric, "pod"))

				podIP := util.GetLabel(metric, "pod_ip")
				hostIP := util.GetLabel(metric, "host_ip")
				if podIP != "" {
					event.Put("ip", podIP)
				}
				if hostIP != "" {
					event.Put("host_ip", hostIP)
				}

			case "kube_pod_status_phase":
				event.Put("status.phase", strings.ToLower(util.GetLabel(metric, "phase")))

			case "kube_pod_status_ready":
				if metric.GetGauge().GetValue() == 1 {
					event.Put("status.ready", util.GetLabel(metric, "condition"))
				}

			case "kube_pod_status_scheduled":
				if metric.GetGauge().GetValue() == 1 {
					event.Put("status.scheduled", util.GetLabel(metric, "condition"))
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
