// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	meraki "github.com/meraki/dashboard-api-go/v3/sdk"
)

type uplink struct {
	lastReportedAt        string
	status                *meraki.ResponseItemApplianceGetOrganizationApplianceUplinkStatusesUplinks
	cellularGatewayStatus *meraki.ResponseItemCellularGatewayGetOrganizationCellularGatewayUplinkStatusesUplinks
	lossAndLatency        *meraki.ResponseItemOrganizationsGetOrganizationDevicesUplinksLossAndLatencyTimeSeries
}

func getDeviceUplinks(client *meraki.Client, organizationID string, devices map[Serial]*Device, period time.Duration) error {
	// there are two separate APIs for uplink statuses depending on the type of device (MG or MX/Z).
	// there is a single API for getting the loss and latency metrics regardless of the type of device.
	// in this function we combine loss and latency metrics with device-specific status information,
	// and attach it to the relevant device in the supplied `devices`` data structure.
	applicanceUplinks, res, err := client.Appliance.GetOrganizationApplianceUplinkStatuses(organizationID, &meraki.GetOrganizationApplianceUplinkStatusesQueryParams{})
	if err != nil {
		return fmt.Errorf("GetOrganizationApplianceUplinkStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	cellularGatewayUplinks, res, err := client.CellularGateway.GetOrganizationCellularGatewayUplinkStatuses(organizationID, &meraki.GetOrganizationCellularGatewayUplinkStatusesQueryParams{})
	if err != nil {
		return fmt.Errorf("GetOrganizationCellularGatewayUplinkStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	lossAndLatency, res, err := client.Organizations.GetOrganizationDevicesUplinksLossAndLatency(
		organizationID,
		&meraki.GetOrganizationDevicesUplinksLossAndLatencyQueryParams{
			Timespan: period.Seconds(),
		},
	)
	if err != nil {
		return fmt.Errorf("GetOrganizationDevicesUplinksLossAndLatency failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	for _, device := range *applicanceUplinks {
		if device.HighAvailability != nil {
			devices[Serial(device.Serial)].haStatus = device.HighAvailability
		}

		if device.Uplinks != nil {
			var uplinks []*uplink
			for _, uplinkStatus := range *device.Uplinks {
				uplink := &uplink{
					lastReportedAt: device.LastReportedAt,
					status:         &uplinkStatus,
				}

				for _, metrics := range *lossAndLatency {
					if metrics.TimeSeries != nil && metrics.Serial == device.Serial && metrics.Uplink == uplinkStatus.Interface {
						// only one bucket per collection is supported
						uplink.lossAndLatency = &(*metrics.TimeSeries)[0]
						break
					}
				}

				uplinks = append(uplinks, uplink)
			}

			devices[Serial(device.Serial)].uplinks = uplinks
		}
	}

	for _, device := range *cellularGatewayUplinks {
		if device.Uplinks == nil {
			continue
		}

		var uplinks []*uplink
		for _, uplinkStatus := range *device.Uplinks {
			uplink := &uplink{
				lastReportedAt:        device.LastReportedAt,
				cellularGatewayStatus: &uplinkStatus,
			}

			for _, metrics := range *lossAndLatency {
				if metrics.TimeSeries != nil && metrics.Serial == device.Serial && metrics.Uplink == uplinkStatus.Interface {
					uplink.lossAndLatency = &(*metrics.TimeSeries)[0]
					break
				}
			}

			uplinks = append(uplinks, uplink)
		}

		devices[Serial(device.Serial)].uplinks = uplinks
	}

	return nil
}

func reportUplinkMetrics(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device) {
	metrics := []mapstr.M{}
	for _, device := range devices {
		if len(device.uplinks) == 0 {
			continue
		}

		for _, uplink := range device.uplinks {
			metric := deviceDetailsToMapstr(device.details)
			metric["uplink.last_reported_at"] = uplink.lastReportedAt

			if uplink.lossAndLatency != nil {
				metric["@timestamp"] = uplink.lossAndLatency.Ts
				metric["uplink.loss.pct"] = uplink.lossAndLatency.LossPercent
				metric["uplink.latency.ms"] = uplink.lossAndLatency.LatencyMs
			}

			if uplink.status != nil {
				metric["uplink.gateway"] = uplink.status.Gateway
				metric["uplink.interface"] = uplink.status.Interface
				metric["uplink.ip"] = uplink.status.IP
				metric["uplink.primary_dns"] = uplink.status.PrimaryDNS
				metric["uplink.secondary_dns"] = uplink.status.SecondaryDNS
				metric["uplink.public_ip"] = uplink.status.PublicIP
				metric["uplink.status"] = uplink.status.Status
				metric["uplink.ip_assigned_by"] = uplink.status.IPAssignedBy
			}

			if uplink.cellularGatewayStatus != nil {
				metric["uplink.gateway"] = uplink.cellularGatewayStatus.Gateway
				metric["uplink.interface"] = uplink.cellularGatewayStatus.Interface
				metric["uplink.ip"] = uplink.cellularGatewayStatus.IP
				metric["uplink.primary_dns"] = uplink.cellularGatewayStatus.DNS1
				metric["uplink.secondary_dns"] = uplink.cellularGatewayStatus.DNS2
				metric["uplink.public_ip"] = uplink.cellularGatewayStatus.PublicIP
				metric["uplink.status"] = uplink.cellularGatewayStatus.Status
				metric["uplink.apn"] = uplink.cellularGatewayStatus.Apn
				metric["uplink.connection_type"] = uplink.cellularGatewayStatus.ConnectionType
				metric["uplink.iccid"] = uplink.cellularGatewayStatus.Iccid
				metric["uplink.model"] = uplink.cellularGatewayStatus.Model
				metric["uplink.provider"] = uplink.cellularGatewayStatus.Provider
				metric["uplink.signal_type"] = uplink.cellularGatewayStatus.SignalType

				if uplink.cellularGatewayStatus.SignalStat != nil {
					metric["uplink.rsrp"] = uplink.cellularGatewayStatus.SignalStat.Rsrp
					metric["uplink.rsrq"] = uplink.cellularGatewayStatus.SignalStat.Rsrq
				}
			}

			metrics = append(metrics, metric)
		}
	}

	reportMetricsForOrganization(reporter, organizationID, metrics)
}
