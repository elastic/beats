// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getDeviceStatuses(client *meraki_api.Client, organizationID string) (map[Serial]*DeviceStatus, error) {
	val, res, err := client.Organizations.GetOrganizationDevicesStatuses(organizationID, &meraki_api.GetOrganizationDevicesStatusesQueryParams{})

	if err != nil {
		return nil, fmt.Errorf("GetOrganizationDevicesStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	statuses := make(map[Serial]*DeviceStatus)
	for _, status := range *val {
		statuses[Serial(status.Serial)] = &DeviceStatus{
			Gateway:        status.Gateway,
			IPType:         status.IPType,
			LastReportedAt: status.LastReportedAt,
			PrimaryDNS:     status.PrimaryDNS,
			PublicIP:       status.PublicIP,
			SecondaryDNS:   status.SecondaryDNS,
			Status:         status.Status,
		}
	}

	return statuses, nil
}

func reportDeviceStatusMetrics(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, deviceStatuses map[Serial]*DeviceStatus) {
	deviceStatusMetrics := []mapstr.M{}
	for serial, device := range devices {
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

		for k, v := range device.Details {
			metric[fmt.Sprintf("device.details.%s", k)] = v
		}

		if status, ok := deviceStatuses[serial]; ok {
			metric["device.status.gateway"] = status.Gateway
			metric["device.status.ip_type"] = status.IPType
			metric["device.status.last_reported_at"] = status.LastReportedAt
			metric["device.status.primary_dns"] = status.PrimaryDNS
			metric["device.status.public_ip"] = status.PublicIP
			metric["device.status.secondary_dns"] = status.SecondaryDNS
			metric["device.status.status"] = status.Status
		}
		deviceStatusMetrics = append(deviceStatusMetrics, metric)
	}

	ReportMetricsForOrganization(reporter, organizationID, deviceStatusMetrics)

}
