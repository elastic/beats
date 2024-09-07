package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func reportCellularGatewayApplianceUplinkStatuses(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, responseCellularGatewayUplinkStatuses *meraki_api.ResponseCellularGatewayGetOrganizationCellularGatewayUplinkStatuses) {

	metrics := []mapstr.M{}

	for _, uplink := range *responseCellularGatewayUplinkStatuses {

		if device, ok := devices[Serial(uplink.Serial)]; ok {
			metric := mapstr.M{
				"cellular.gateway.uplink.network_id":       uplink.NetworkID,
				"cellular.gateway.uplink.last_reported_at": uplink.LastReportedAt,
				"device.address":                           device.Address,
				"device.firmware":                          device.Firmware,
				"device.imei":                              device.Imei,
				"device.lan_ip":                            device.LanIP,
				"device.location":                          device.Location,
				"device.mac":                               device.Mac,
				"device.model":                             device.Model,
				"device.name":                              device.Name,
				"device.network_id":                        device.NetworkID,
				"device.notes":                             device.Notes,
				"device.product_type":                      device.ProductType,
				"device.serial":                            device.Serial,
				"device.tags":                              device.Tags,
			}

			for i, item := range *uplink.Uplinks {
				metrics = append(metrics, mapstr.Union(metric, mapstr.M{
					fmt.Sprintf("cellular.gateway.uplink.item_%d.apn", i):              item.Apn,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.connection_type", i):  item.ConnectionType,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.dns1", i):             item.DNS1,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.dns2", i):             item.DNS2,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.gateway", i):          item.Gateway,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.iccid", i):            item.Iccid,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.interface", i):        item.Interface,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.ip", i):               item.IP,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.model", i):            item.Model,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.provider", i):         item.Provider,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.public_ip", i):        item.PublicIP,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.signal_stat.rsrp", i): item.SignalStat.Rsrp,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.signal_stat.rsrq", i): item.SignalStat.Rsrq,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.signal_type", i):      item.SignalType,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.status", i):           item.Status,
				}))

			}
		}
	}
	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
