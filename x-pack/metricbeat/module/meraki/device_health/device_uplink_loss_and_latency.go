// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getDeviceUplinkLossLatencyMetrics(client *meraki_api.Client, organizationID string, period time.Duration) ([]*Uplink, error) {
	val, res, err := client.Organizations.GetOrganizationDevicesUplinksLossAndLatency(
		organizationID,
		&meraki_api.GetOrganizationDevicesUplinksLossAndLatencyQueryParams{
			Timespan: period.Seconds() + 10, // slightly longer than the fetch period to ensure we don't miss measurements due to jitter
		},
	)

	if err != nil {
		return nil, fmt.Errorf("GetOrganizationDevicesUplinksLossAndLatency failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	var uplinks []*Uplink

	for _, device := range *val {
		uplink := &Uplink{
			DeviceSerial: Serial(device.Serial),
			IP:           device.IP,
			Interface:    device.Uplink,
			NetworkID:    device.NetworkID,
		}

		for _, measurement := range *device.TimeSeries {
			if measurement.LossPercent != nil || measurement.LatencyMs != nil {
				timestamp, err := time.Parse(time.RFC3339, measurement.Ts)
				if err != nil {
					return nil, fmt.Errorf("failed to parse timestamp [%s] in ResponseOrganizationsGetOrganizationDevicesUplinksLossAndLatency: %w", measurement.Ts, err)
				}

				metric := UplinkMetric{Timestamp: timestamp}
				if measurement.LossPercent != nil {
					metric.LossPercent = measurement.LossPercent
				}
				if measurement.LatencyMs != nil {
					metric.LatencyMs = measurement.LatencyMs
				}
				uplink.Metrics = append(uplink.Metrics, &metric)
			}
		}

		if len(uplink.Metrics) != 0 {
			uplinks = append(uplinks, uplink)
		}
	}

	return uplinks, nil
}

func getDeviceUplinkMetrics(client *meraki_api.Client, organizationID string, period time.Duration) ([]*Uplink, error) {
	val, res, err := client.Organizations.GetOrganizationDevicesUplinksLossAndLatency(
		organizationID,
		&meraki_api.GetOrganizationDevicesUplinksLossAndLatencyQueryParams{
			Timespan: period.Seconds() + 10, // slightly longer than the fetch period to ensure we don't miss measurements due to jitter
		},
	)

	if err != nil {
		return nil, fmt.Errorf("GetOrganizationDevicesUplinksLossAndLatency failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	var uplinks []*Uplink

	for _, device := range *val {
		uplink := &Uplink{
			DeviceSerial: Serial(device.Serial),
			IP:           device.IP,
			Interface:    device.Uplink,
		}

		for _, measurement := range *device.TimeSeries {
			if measurement.LossPercent != nil || measurement.LatencyMs != nil {
				timestamp, err := time.Parse(time.RFC3339, measurement.Ts)
				if err != nil {
					return nil, fmt.Errorf("failed to parse timestamp [%s] in ResponseOrganizationsGetOrganizationDevicesUplinksLossAndLatency: %w", measurement.Ts, err)
				}

				metric := UplinkMetric{Timestamp: timestamp}
				if measurement.LossPercent != nil {
					metric.LossPercent = measurement.LossPercent
				}
				if measurement.LatencyMs != nil {
					metric.LatencyMs = measurement.LatencyMs
				}
				uplink.Metrics = append(uplink.Metrics, &metric)
			}
		}

		if len(uplink.Metrics) != 0 {
			uplinks = append(uplinks, uplink)
		}
	}

	return uplinks, nil
}

func reportDeviceUplinkMetrics(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, uplinks []*Uplink) {
	metrics := []mapstr.M{}

	for _, uplink := range uplinks {
		if device, ok := devices[uplink.DeviceSerial]; ok {
			metric := mapstr.M{
				"uplink.ip":        uplink.IP,
				"uplink.interface": uplink.Interface,
				// fixme: repeated code serializing device metadata to mapstr
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

			for _, uplinkMetric := range uplink.Metrics {
				metrics = append(metrics, mapstr.Union(metric, mapstr.M{
					"@timestamp":          uplinkMetric.Timestamp,
					"uplink.loss_percent": uplinkMetric.LossPercent,
					"uplink.latency_ms":   uplinkMetric.LatencyMs,
				}))
			}
		} else {
			// missing device metadata; ignore
		}
	}
	ReportMetricsForOrganization(reporter, organizationID, metrics)
}
