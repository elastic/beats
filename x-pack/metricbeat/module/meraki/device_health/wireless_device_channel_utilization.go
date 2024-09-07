package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func reportWirelessDeviceChannelUtilization(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, wirelessDevices WirelessDevicesChannelUtilizationByDevice) {

	metrics := []mapstr.M{}

	for _, wirelessDevice := range wirelessDevices {

		if device, ok := devices[Serial(wirelessDevice.Serial)]; ok {

			metric := mapstr.M{
				"wireless.device.address":      device.Address,
				"wireless.device.firmware":     device.Firmware,
				"wireless.device.imei":         device.Imei,
				"wireless.device.lan_ip":       device.LanIP,
				"wireless.device.location":     device.Location,
				"wireless.device.mac":          device.Mac,
				"wireless.device.model":        device.Model,
				"wireless.device.name":         device.Name,
				"wireless.device.network_id":   device.NetworkID,
				"wireless.device.notes":        device.Notes,
				"wireless.device.product_type": device.ProductType,
				"wireless.device.serial":       device.Serial,
				"wireless.device.tags":         device.Tags,
			}

			for _, v := range wirelessDevice.ByBand {
				metric[fmt.Sprintf("wireless.device.channel.utilization.band_%s.wifi.percentage", v.Band)] = v.Wifi.Percentage
				metric[fmt.Sprintf("wireless.device.channel.utilization.band_%s.nonwifi.percentage", v.Band)] = v.NonWifi.Percentage
				metric[fmt.Sprintf("wireless.device.channel.utilization.band_%s.total.percentage", v.Band)] = v.Total.Percentage
			}

			metrics = append(metrics, metric)

		}

	}
	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
