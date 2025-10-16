// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

type uplink struct {
	lastReportedAt        string
	status                *sdk.ResponseItemApplianceGetOrganizationApplianceUplinkStatusesUplinks
	cellularGatewayStatus *sdk.ResponseItemCellularGatewayGetOrganizationCellularGatewayUplinkStatusesUplinks
	lossAndLatency        *sdk.ResponseItemOrganizationsGetOrganizationDevicesUplinksLossAndLatency
}

func getDeviceUplinks(client *sdk.Client, organizationID string, devices map[Serial]*Device, period time.Duration, logger *logp.Logger) error {
	// there are two separate APIs for uplink statuses depending on the type of device (MG or MX/Z).
	// there is a single API for getting the loss and latency metrics regardless of the type of device.
	// in this function we combine loss and latency metrics with device-specific status information,
	// and attach it to the relevant device in the supplied `devices` data structure.

	// Fetch loss and latency metrics once (non-paginated)
	lossAndLatency, res, err := client.Organizations.GetOrganizationDevicesUplinksLossAndLatency(
		organizationID,
		&sdk.GetOrganizationDevicesUplinksLossAndLatencyQueryParams{
			Timespan: period.Seconds(),
		},
	)
	if err != nil {
		if res != nil {
			return fmt.Errorf("GetOrganizationDevicesUplinksLossAndLatency failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		return fmt.Errorf("GetOrganizationDevicesUplinksLossAndLatency failed; %w", err)
	}

	if lossAndLatency == nil {
		return errors.New("unexpected response from Meraki API: lossAndLatency is nil")
	}

	// Paginate through appliance uplinks
	applianceParams := &sdk.GetOrganizationApplianceUplinkStatusesQueryParams{}
	setApplianceStart := func(s string) { applianceParams.StartingAfter = s }

	doApplianceRequest := func() (*sdk.ResponseApplianceGetOrganizationApplianceUplinkStatuses, *resty.Response, error) {
		return client.Appliance.GetOrganizationApplianceUplinkStatuses(organizationID, applianceParams)
	}

	onApplianceError := func(err error, res *resty.Response) error {
		if res != nil {
			return fmt.Errorf("GetOrganizationApplianceUplinkStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		return fmt.Errorf("GetOrganizationApplianceUplinkStatuses failed; %w", err)
	}

	onApplianceSuccess := func(applicanceUplinks *sdk.ResponseApplianceGetOrganizationApplianceUplinkStatuses) error {
		if applicanceUplinks == nil {
			return errors.New("unexpected response from Meraki API: applicanceUplinks is nil")
		}

		for _, device := range *applicanceUplinks {
			deviceObj, ok := devices[Serial(device.Serial)]
			if device.HighAvailability != nil && ok && deviceObj != nil {
				devices[Serial(device.Serial)].haStatus = device.HighAvailability
			}

			if device.Uplinks != nil {
				var uplinks []*uplink
				for i := range *device.Uplinks {
					uplinkStatus := (*device.Uplinks)[i]
					uplink := &uplink{
						lastReportedAt: device.LastReportedAt,
						status:         &uplinkStatus,
					}

					for j := range *lossAndLatency {
						metrics := (*lossAndLatency)[j]
						if metrics.TimeSeries != nil && metrics.Serial == device.Serial && metrics.Uplink == uplinkStatus.Interface {
							uplink.lossAndLatency = &metrics
							break
						}
					}

					uplinks = append(uplinks, uplink)
				}

				if ok && deviceObj != nil {
					devices[Serial(device.Serial)].uplinks = uplinks
				}
			}
		}

		return nil
	}

	err = meraki.NewPaginator(
		setApplianceStart,
		doApplianceRequest,
		onApplianceError,
		onApplianceSuccess,
		logger,
	).GetAllPages()

	if err != nil {
		return err
	}

	// Paginate through cellular gateway uplinks
	cellularParams := &sdk.GetOrganizationCellularGatewayUplinkStatusesQueryParams{}
	setCellularStart := func(s string) { cellularParams.StartingAfter = s }

	doCellularRequest := func() (*sdk.ResponseCellularGatewayGetOrganizationCellularGatewayUplinkStatuses, *resty.Response, error) {
		return client.CellularGateway.GetOrganizationCellularGatewayUplinkStatuses(organizationID, cellularParams)
	}

	onCellularError := func(err error, res *resty.Response) error {
		if res != nil {
			return fmt.Errorf("GetOrganizationCellularGatewayUplinkStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		return fmt.Errorf("GetOrganizationCellularGatewayUplinkStatuses failed; %w", err)
	}

	onCellularSuccess := func(cellularGatewayUplinks *sdk.ResponseCellularGatewayGetOrganizationCellularGatewayUplinkStatuses) error {
		if cellularGatewayUplinks == nil {
			return errors.New("unexpected response from Meraki API: cellularGatewayUplinks is nil")
		}

		for _, device := range *cellularGatewayUplinks {
			if device.Uplinks == nil {
				continue
			}

			var uplinks []*uplink
			for i := range *device.Uplinks {
				uplinkStatus := (*device.Uplinks)[i]
				uplink := &uplink{
					lastReportedAt:        device.LastReportedAt,
					cellularGatewayStatus: &uplinkStatus,
				}

				for j := range *lossAndLatency {
					metrics := (*lossAndLatency)[j]
					if metrics.TimeSeries != nil && metrics.Serial == device.Serial && metrics.Uplink == uplinkStatus.Interface {
						uplink.lossAndLatency = &metrics
						break
					}
				}

				uplinks = append(uplinks, uplink)
			}

			deviceObj, ok := devices[Serial(device.Serial)]
			if ok && deviceObj != nil {
				devices[Serial(device.Serial)].uplinks = uplinks
			}
		}

		return nil
	}

	err = meraki.NewPaginator(
		setCellularStart,
		doCellularRequest,
		onCellularError,
		onCellularSuccess,
		logger,
	).GetAllPages()

	return err
}

func reportUplinkMetrics(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device) {
	metrics := []mapstr.M{}
	for _, device := range devices {
		if device == nil || device.details == nil || len(device.uplinks) == 0 {
			continue
		}

		for _, uplink := range device.uplinks {
			if uplink == nil {
				continue
			}
			if uplink.lossAndLatency != nil {
				// each loss and latency metric can have multiple values per collection.
				// we report each value as it's own (smaller) metric event, containing
				// the identifying device/uplink fields.
				for _, dataPoint := range *uplink.lossAndLatency.TimeSeries {
					// for some reason there are sometimes empty buckets
					if dataPoint.LatencyMs != nil || dataPoint.LossPercent != nil {
						metrics = append(metrics, mapstr.M{
							"@timestamp":        dataPoint.Ts,
							"uplink.latency.ms": dataPoint.LatencyMs,
							"uplink.loss.pct":   dataPoint.LossPercent,

							"device.serial":     uplink.lossAndLatency.Serial,    // _should_ be the same as `device.Serial`
							"device.network_id": uplink.lossAndLatency.NetworkID, // _should_ be the same as `device.NetworkID`
							"uplink.interface":  uplink.lossAndLatency.Uplink,
							"uplink.ip":         uplink.lossAndLatency.IP,
						})
					}
				}
			}

			statusMetric := deviceDetailsToMapstr(device.details)
			statusMetric["uplink.last_reported_at"] = uplink.lastReportedAt

			if uplink.status != nil {
				statusMetric["uplink.gateway"] = uplink.status.Gateway
				statusMetric["uplink.interface"] = uplink.status.Interface
				statusMetric["uplink.ip"] = uplink.status.IP
				statusMetric["uplink.primary_dns"] = uplink.status.PrimaryDNS
				statusMetric["uplink.secondary_dns"] = uplink.status.SecondaryDNS
				statusMetric["uplink.public_ip"] = uplink.status.PublicIP
				statusMetric["uplink.status"] = uplink.status.Status
				statusMetric["uplink.ip_assigned_by"] = uplink.status.IPAssignedBy
			}

			if uplink.cellularGatewayStatus != nil {
				statusMetric["uplink.gateway"] = uplink.cellularGatewayStatus.Gateway
				statusMetric["uplink.interface"] = uplink.cellularGatewayStatus.Interface
				statusMetric["uplink.ip"] = uplink.cellularGatewayStatus.IP
				statusMetric["uplink.primary_dns"] = uplink.cellularGatewayStatus.DNS1
				statusMetric["uplink.secondary_dns"] = uplink.cellularGatewayStatus.DNS2
				statusMetric["uplink.public_ip"] = uplink.cellularGatewayStatus.PublicIP
				statusMetric["uplink.status"] = uplink.cellularGatewayStatus.Status
				statusMetric["uplink.apn"] = uplink.cellularGatewayStatus.Apn
				statusMetric["uplink.connection_type"] = uplink.cellularGatewayStatus.ConnectionType
				statusMetric["uplink.iccid"] = uplink.cellularGatewayStatus.Iccid
				statusMetric["uplink.model"] = uplink.cellularGatewayStatus.Model
				statusMetric["uplink.provider"] = uplink.cellularGatewayStatus.Provider
				statusMetric["uplink.signal_type"] = uplink.cellularGatewayStatus.SignalType

				if uplink.cellularGatewayStatus.SignalStat != nil {
					statusMetric["uplink.rsrp"] = uplink.cellularGatewayStatus.SignalStat.Rsrp
					statusMetric["uplink.rsrq"] = uplink.cellularGatewayStatus.SignalStat.Rsrq
				}
			}

			metrics = append(metrics, statusMetric)
		}
	}

	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}
