package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
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

				uplink_encountered := false

				metric["uplink.interface"] = item.Interface
				metric["uplink.status"] = item.Status
				metric["uplink.ip"] = item.IP
				metric["uplink.gateway"] = item.Gateway
				metric["uplink.public_ip"] = item.PublicIP
				metric["uplink.primary_dns"] = item.PrimaryDNS
				metric["uplink.secondary_dns"] = item.SecondaryDNS
				metric["uplink.ip_assigned_by"] = item.IPAssignedBy

				for _, lossLatencyMetric := range lossLatencyUplinks {
					if lossLatencyMetric.Interface == item.Interface && string(lossLatencyMetric.DeviceSerial) == device.Serial && lossLatencyMetric.NetworkID == device.NetworkID {
						for _, lossLatency := range lossLatencyMetric.Metrics {

							uplink_encountered = true
							//It seems there is bug in the client.Organizations.GetOrganizationDevicesUplinksLossAndLatency code returning differnt IP
							//To mitigate, I am additionally printing the ip as seperate value, IMO it is odd these do not match.
							// client.Appliance.GetOrganizationApplianceUplinkStatuses
							metrics = append(metrics, mapstr.Union(metric, mapstr.M{
								"uplink.loss_latancy.ip":           lossLatencyMetric.IP,
								"@timestamp":                       lossLatency.Timestamp,
								"uplink.loss_latancy.loss_percent": lossLatency.LossPercent,
								"uplink.loss_latancy.latency_ms":   lossLatency.LatencyMs,
							}))
						}
					}

				}
				if !uplink_encountered {
					metrics = append(metrics, metric)
				}

			}
		}
	}

	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
