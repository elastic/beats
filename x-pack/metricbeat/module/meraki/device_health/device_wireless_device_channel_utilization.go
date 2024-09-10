package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func reportWirelessDeviceChannelUtilization(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, wirelessDevices *meraki_api.ResponseOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDevice) {

	metrics := []mapstr.M{}

	for _, wirelessDevice := range *wirelessDevices {

		if device, ok := devices[Serial(wirelessDevice.Serial)]; ok {

			metric := mapstr.M{
				"device.address":      device.Address,
				"device.firmware":     device.Firmware,
				"device.imei":         device.Imei,
				"device.lan_ip":       device.LanIP,
				"device.location":     device.Location,
				"device.mac":          device.Mac,
				"device.model":        device.Model,
				"device.name":         device.Name,
				"device.network_id":   device.NetworkID,
				"device.notes":        device.Notes,
				"device.product_type": device.ProductType,
				"device.serial":       device.Serial,
				"device.tags":         device.Tags,
			}

			for _, v := range *wirelessDevice.ByBand {
				metric[fmt.Sprintf("wireless.device.channel.utilization.band_%s.wifi.percentage", common.DeDot(v.Band))] = v.Wifi.Percentage
				metric[fmt.Sprintf("wireless.device.channel.utilization.band_%s.nonwifi.percentage", common.DeDot(v.Band))] = v.NonWifi.Percentage
				metric[fmt.Sprintf("wireless.device.channel.utilization.band_%s.total.percentage", common.DeDot(v.Band))] = v.Total.Percentage
			}

			metrics = append(metrics, metric)

		}

	}
	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
