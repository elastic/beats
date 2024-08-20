package network

import (
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func (m *MetricSet) eventMapping(net mo.Network, data *metricData) mapstr.M {
	event := mapstr.M{}

	event.Put("name", net.Name)
	event.Put("status", net.OverallStatus)
	event.Put("accessible", net.Summary.GetNetworkSummary().Accessible)
	event.Put("config.status", net.ConfigStatus)

	if len(data.assetsName.outputHostNames) > 0 {
		event.Put("host.names", data.assetsName.outputHostNames)
		event.Put("host.count", len(data.assetsName.outputHostNames))
	}

	if len(data.assetsName.outputVmNames) > 0 {
		event.Put("vm.names", data.assetsName.outputVmNames)
		event.Put("vm.count", len(data.assetsName.outputVmNames))
	}

	return event
}
