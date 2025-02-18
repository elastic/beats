// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki "github.com/meraki/dashboard-api-go/v3/sdk"
)

// Serial is the unique identifier for all devices
type Serial string

// Device contains attributes, statuses and metrics for Meraki devices
type Device struct {
	details          *meraki.ResponseItemOrganizationsGetOrganizationDevices
	status           *meraki.ResponseItemOrganizationsGetOrganizationDevicesStatuses
	haStatus         *meraki.ResponseItemApplianceGetOrganizationApplianceUplinkStatusesHighAvailability
	performanceScore *meraki.ResponseApplianceGetDeviceAppliancePerformance
	wifi0            *meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi0
	wifi1            *meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi1
	license          *meraki.ResponseItemOrganizationsGetOrganizationLicenses

	uplinks     []*uplink
	switchports []*switchport
}

func getDevices(client *meraki.Client, organizationID string) (map[Serial]*Device, error) {
	val, res, err := client.Organizations.GetOrganizationDevices(organizationID, &meraki.GetOrganizationDevicesQueryParams{})

	if err != nil {
		return nil, fmt.Errorf("GetOrganizationDevices failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	devices := make(map[Serial]*Device)
	for i := range *val {
		device := (*val)[i]
		devices[Serial(device.Serial)] = &Device{
			details: &device,
		}
	}

	return devices, nil
}

func getDeviceStatuses(client *meraki.Client, organizationID string, devices map[Serial]*Device) error {
	val, res, err := client.Organizations.GetOrganizationDevicesStatuses(organizationID, &meraki.GetOrganizationDevicesStatusesQueryParams{})

	if err != nil {
		return fmt.Errorf("GetOrganizationDevicesStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	for i := range *val {
		status := (*val)[i]
		if device, ok := devices[Serial(status.Serial)]; ok {
			device.status = &status
		}
	}

	return nil
}

func getDevicePerformanceScores(logger *logp.Logger, client *meraki.Client, devices map[Serial]*Device) {
	for _, device := range devices {
		// attempting to get a performance score for a non-MX device returns a 400
		if strings.Index(device.details.Model, "MX") != 0 {
			continue
		}

		val, res, err := client.Appliance.GetDeviceAppliancePerformance(device.details.Serial)
		if err != nil {
			if !(res.StatusCode() != http.StatusBadRequest && strings.Contains(string(res.Body()), "Feature not supported")) {
				logger.Errorf("GetDeviceAppliancePerformance failed; [%d] %s. %v", res.StatusCode(), res.Body(), err)
			}

			continue
		}

		// 204 indicates there is no data for the device, it's likely 'offline' or 'dormant'
		if res.StatusCode() != 204 {
			device.performanceScore = val
		}
	}
}

type NetworkHealthService interface {
	GetNetworkNetworkHealthChannelUtilization(networkID string, getNetworkNetworkHealthChannelUtilizationQueryParams *meraki.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error)
}

type NetworkHealthServiceWrapper struct {
	service *meraki.NetworksService
}

func (w *NetworkHealthServiceWrapper) GetNetworkNetworkHealthChannelUtilization(networkID string, getNetworkNetworkHealthChannelUtilizationQueryParams *meraki.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error) {
	return w.service.GetNetworkNetworkHealthChannelUtilization(networkID, getNetworkNetworkHealthChannelUtilizationQueryParams)
}

func getDeviceChannelUtilization(client NetworkHealthService, devices map[Serial]*Device, period time.Duration) error {
	// There are two ways to get this information from the API.
	// An alternative to this would be to use `/organizations/{organizationId}/wireless/devices/channelUtilization/byDevice`,
	// avoids the need to extract the filtered network IDs below.
	// However, the SDK's implementation of that operation doesn't have proper type handling, so we perfer this one.
	// (The naming is also a bit different in the returned data, e.g. wifi0/wifi1 vs band 2.4/5; 80211/non80211 vs wifi/nonwifi)

	networkIDs := make(map[string]bool)
	for _, device := range devices {
		if device.details.ProductType != "wireless" {
			continue
		}

		if _, ok := networkIDs[device.details.NetworkID]; !ok {
			networkIDs[device.details.NetworkID] = true
		}
	}

	for networkID := range networkIDs {
		val, res, err := client.GetNetworkNetworkHealthChannelUtilization(
			networkID,
			&meraki.GetNetworkNetworkHealthChannelUtilizationQueryParams{
				Timespan: period.Seconds(),
			},
		)

		if err != nil {
			if strings.Contains(string(res.Body()), "MR 27.0") {
				// "This endpoint is only available for networks on MR 27.0 or above."
				continue
			}

			return fmt.Errorf("GetNetworkNetworkHealthChannelUtilization failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}

		for _, utilization := range *val {
			if device, ok := devices[Serial(utilization.Serial)]; ok {
				if utilization.Wifi0 != nil && len(*utilization.Wifi0) != 0 {
					// only take the first bucket - collection intervals which result in multiple buckets are not supported
					if device.wifi0 == nil {
						device.wifi0 = &meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi0{}
					}
					device.wifi0.Utilization80211 = (*utilization.Wifi0)[0].Utilization80211
					device.wifi0.UtilizationNon80211 = (*utilization.Wifi0)[0].UtilizationNon80211
					device.wifi0.UtilizationTotal = (*utilization.Wifi0)[0].UtilizationTotal
				}
				if utilization.Wifi1 != nil && len(*utilization.Wifi1) != 0 {
					if device.wifi1 == nil {
						device.wifi1 = &meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi1{}
					}
					device.wifi1.Utilization80211 = (*utilization.Wifi1)[0].Utilization80211
					device.wifi1.UtilizationNon80211 = (*utilization.Wifi1)[0].UtilizationNon80211
					device.wifi1.UtilizationTotal = (*utilization.Wifi1)[0].UtilizationTotal
				}
			}
		}
	}

	return nil
}

func getDeviceLicenses(client *meraki.Client, organizationID string, devices map[Serial]*Device) error {
	val, res, err := client.Organizations.GetOrganizationLicenses(organizationID, &meraki.GetOrganizationLicensesQueryParams{})
	if err != nil {
		// Ignore 400 error for per-device licensing not supported
		if res.StatusCode() == 400 && strings.Contains(string(res.Body()), "does not support per-device licensing") {
			return nil
		}
		return fmt.Errorf("GetOrganizationLicenses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	for i := range *val {
		license := (*val)[i]
		if device, ok := devices[Serial(license.DeviceSerial)]; ok {
			device.license = &license
		}
	}

	return nil
}

func deviceDetailsToMapstr(details *meraki.ResponseItemOrganizationsGetOrganizationDevices) mapstr.M {
	return mapstr.M{
		"device.serial":       details.Serial,
		"device.address":      details.Address,
		"device.firmware":     details.Firmware,
		"device.imei":         details.Imei,
		"device.lan_ip":       details.LanIP,
		"device.location":     []*float64{details.Lng, details.Lat}, // (lon, lat) order is important for geo_ip mapping type!
		"device.mac":          details.Mac,
		"device.model":        details.Model,
		"device.name":         details.Name,
		"device.network_id":   details.NetworkID,
		"device.notes":        details.Notes,
		"device.product_type": details.ProductType,
		"device.tags":         details.Tags,
	}
}

func reportDeviceMetrics(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device) {
	metrics := []mapstr.M{}
	for _, device := range devices {
		metric := deviceDetailsToMapstr(device.details)

		if device.haStatus != nil {
			metric["device.high_availability.enabled"] = device.haStatus.Enabled
			metric["device.high_availability.role"] = device.haStatus.Role
		}

		if device.status != nil {
			metric["device.status.gateway"] = device.status.Gateway
			metric["device.status.ip_type"] = device.status.IPType
			metric["device.status.last_reported_at"] = device.status.LastReportedAt
			metric["device.status.primary_dns"] = device.status.PrimaryDNS
			metric["device.status.public_ip"] = device.status.PublicIP
			metric["device.status.secondary_dns"] = device.status.SecondaryDNS
			metric["device.status.value"] = device.status.Status
		}

		if device.performanceScore != nil {
			metric["device.performance_score"] = device.performanceScore.PerfScore
		}

		if device.wifi0 != nil {
			metric["device.channel_utilization.wifi0.utilization_80211"] = device.wifi0.Utilization80211
			metric["device.channel_utilization.wifi0.utilization_non_80211"] = device.wifi0.UtilizationNon80211
			metric["device.channel_utilization.wifi0.utilization_total"] = device.wifi0.UtilizationTotal
		}

		if device.wifi1 != nil {
			metric["device.channel_utilization.wifi1.utilization_80211"] = device.wifi1.Utilization80211
			metric["device.channel_utilization.wifi1.utilization_non_80211"] = device.wifi1.UtilizationNon80211
			metric["device.channel_utilization.wifi1.utilization_total"] = device.wifi1.UtilizationTotal
		}

		if device.license != nil {
			metric["device.license.activation_date"] = device.license.ActivationDate
			metric["device.license.claim_date"] = device.license.ClaimDate
			metric["device.license.duration_in_days"] = device.license.DurationInDays
			metric["device.license.expiration_date"] = device.license.ExpirationDate
			metric["device.license.head_license_id"] = device.license.HeadLicenseID
			metric["device.license.id"] = device.license.ID
			metric["device.license.license_type"] = device.license.LicenseType
			metric["device.license.order_number"] = device.license.OrderNumber
			metric["device.license.seat_count"] = device.license.SeatCount
			metric["device.license.state"] = device.license.State
			metric["device.license.total_duration_in_days"] = device.license.TotalDurationInDays
		}

		metrics = append(metrics, metric)
	}

	reportMetricsForOrganization(reporter, organizationID, metrics)
}
