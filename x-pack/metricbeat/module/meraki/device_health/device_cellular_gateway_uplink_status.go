package device_health

import (
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

			for _, item := range *uplink.Uplinks {
				metrics = append(metrics, mapstr.Union(metric, mapstr.M{
					"cellular.gateway.uplink.apn":              item.Apn,
					"cellular.gateway.uplink.connection_type":  item.ConnectionType,
					"cellular.gateway.uplink.dns1":             item.DNS1,
					"cellular.gateway.uplink.dns2":             item.DNS2,
					"cellular.gateway.uplink.gateway":          item.Gateway,
					"cellular.gateway.uplink.iccid":            item.Iccid,
					"cellular.gateway.uplink.interface":        item.Interface,
					"cellular.gateway.uplink.ip":               item.IP,
					"cellular.gateway.uplink.model":            item.Model,
					"cellular.gateway.uplink.provider":         item.Provider,
					"cellular.gateway.uplink.public_ip":        item.PublicIP,
					"cellular.gateway.uplink.signal_stat.rsrp": item.SignalStat.Rsrp,
					"cellular.gateway.uplink.signal_stat.rsrq": item.SignalStat.Rsrq,
					"cellular.gateway.uplink.signal_type":      item.SignalType,
					"cellular.gateway.uplink.status":           item.Status,
				}))

			}
		}
	}
	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
