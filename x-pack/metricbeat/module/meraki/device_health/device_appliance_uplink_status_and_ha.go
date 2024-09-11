package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/tommyers-elastic/dashboard-api-go/v3/sdk"
)

func reportApplianceUplinkStatuses(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, responseApplianceUplinkStatuses *meraki_api.ResponseApplianceGetOrganizationApplianceUplinkStatuses, lossLatencyUplinks []*Uplink) {

	metrics := []mapstr.M{}

	for _, uplink := range *responseApplianceUplinkStatuses {

		if device, ok := devices[Serial(uplink.Serial)]; ok {
			metric := mapstr.M{
				"uplink.high_availablity.enabled": uplink.HighAvailability.Enabled,
				"uplink.high_availablity.role":    uplink.HighAvailability.Role,
				"uplink.last_reported_at":         uplink.LastReportedAt,
				"device.address":                  device.Address,
				"device.firmware":                 device.Firmware,
				"device.imei":                     device.Imei,
				"device.lan_ip":                   device.LanIP,
				"device.location":                 device.Location,
				"device.mac":                      device.Mac,
				"device.model":                    device.Model,
				"device.name":                     device.Name,
				"device.network_id":               device.NetworkID,
				"device.notes":                    device.Notes,
				"device.product_type":             device.ProductType,
				"device.serial":                   device.Serial,
				"device.tags":                     device.Tags,
			}

			for k, v := range device.Details {
				metric[fmt.Sprintf("device.details.%s", k)] = v
			}

			for _, item := range *uplink.Uplinks {

				for _, lossLatencyMetric := range lossLatencyUplinks {
					if lossLatencyMetric.Interface == item.Interface && string(lossLatencyMetric.DeviceSerial) == device.Serial && lossLatencyMetric.NetworkID == device.NetworkID {
						for _, lossLatency := range lossLatencyMetric.Metrics {
							//It seems there is bug in the client.Organizations.GetOrganizationDevicesUplinksLossAndLatency code returning differnt IP
							//To mitigate, I am additionally printing the ip as seperate value, IMO it is odd these do not match.
							// client.Appliance.GetOrganizationApplianceUplinkStatuses
							metric["uplink.loss_latancy.ip"] = lossLatencyMetric.IP
							metric["uplink.loss_latancy.@timestamp"] = lossLatency.Timestamp
							metric["uplink.loss_latancy.loss_percent"] = lossLatency.LossPercent
							metric["uplink.loss_latancy.latency_ms"] = lossLatency.LatencyMs

						}
					}

				}
				metrics = append(metrics, mapstr.Union(metric, mapstr.M{
					"uplink.interface":      item.Interface,
					"uplink.status":         item.Status,
					"uplink.ip":             item.IP,
					"uplink.gateway":        item.Gateway,
					"uplink.public_ip":      item.PublicIP,
					"uplink.primary_dns":    item.PrimaryDNS,
					"uplink.secondary_dns":  item.SecondaryDNS,
					"uplink.ip_assigned_by": item.IPAssignedBy,
				}))

			}
		}
	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
