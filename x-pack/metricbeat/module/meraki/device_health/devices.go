// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/go-resty/resty/v2"
	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

// Serial is the unique identifier for all devices
type Serial string

// Device contains attributes, statuses and metrics for Meraki devices
type Device struct {
	details          *sdk.ResponseItemOrganizationsGetOrganizationDevices
	status           *sdk.ResponseItemOrganizationsGetOrganizationDevicesStatuses
	haStatus         *sdk.ResponseItemApplianceGetOrganizationApplianceUplinkStatusesHighAvailability
	performanceScore *sdk.ResponseApplianceGetDeviceAppliancePerformance
	license          *sdk.ResponseItemOrganizationsGetOrganizationLicenses
	bandUtilization  map[string]*sdk.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBand

	uplinks     []*uplink
	switchports []*switchport
}

func getDevices(client *sdk.Client, organizationID string) (map[Serial]*Device, error) {
	val, res, err := client.Organizations.GetOrganizationDevices(organizationID, &sdk.GetOrganizationDevicesQueryParams{})

	if err != nil {
		if res != nil {
			return nil, fmt.Errorf("GetOrganizationDevices failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		return nil, fmt.Errorf("GetOrganizationDevices failed; %w", err)
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

func getDeviceStatuses(client *sdk.Client, organizationID string, devices map[Serial]*Device) error {
	val, res, err := client.Organizations.GetOrganizationDevicesStatuses(organizationID, &sdk.GetOrganizationDevicesStatusesQueryParams{})

	if err != nil {
		if res != nil {
			return fmt.Errorf("GetOrganizationDevicesStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		return fmt.Errorf("GetOrganizationDevicesStatuses failed; %w", err)
	}

	if val == nil {
		return errors.New("GetOrganizationDevicesStatuses returned nil response")
	}

	for i := range *val {
		status := (*val)[i]
		if device, ok := devices[Serial(status.Serial)]; ok {
			device.status = &status
		}
	}

	return nil
}

func getDevicePerformanceScores(logger *logp.Logger, client *sdk.Client, devices map[Serial]*Device) {
	for _, device := range devices {
		if device == nil || device.details == nil {
			continue
		}
		// attempting to get a performance score for a non-MX device returns a 400
		if strings.Index(device.details.Model, "MX") != 0 {
			continue
		}

		val, res, err := client.Appliance.GetDeviceAppliancePerformance(device.details.Serial)
		if err != nil {
			if res != nil && (res.StatusCode() == http.StatusBadRequest && !strings.Contains(string(res.Body()), "Feature not supported")) {
				logger.Errorf("GetDeviceAppliancePerformance failed; [%d] %s. %v", res.StatusCode(), res.Body(), err)
			}

			continue
		}

		// 204 indicates there is no data for the device, it's likely 'offline' or 'dormant'
		if res != nil && res.StatusCode() != 204 {
			device.performanceScore = val
		}
	}
}

type NetworkHealthService interface {
	GetNetworkNetworkHealthChannelUtilization(networkID string, getNetworkNetworkHealthChannelUtilizationQueryParams *sdk.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*sdk.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error)
}

type NetworkHealthServiceWrapper struct {
	service *sdk.NetworksService
}

func (w *NetworkHealthServiceWrapper) GetNetworkNetworkHealthChannelUtilization(networkID string, getNetworkNetworkHealthChannelUtilizationQueryParams *sdk.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*sdk.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error) {
	return w.service.GetNetworkNetworkHealthChannelUtilization(networkID, getNetworkNetworkHealthChannelUtilizationQueryParams)
}

type DeviceService interface {
	GetOrganizationWirelessDevicesChannelUtilizationByDevice(organizationID string, getOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams *sdk.GetOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams) (*resty.Response, error)
}

type DeviceServiceWrapper struct {
	service *sdk.DevicesService
}

func (w *DeviceServiceWrapper) GetOrganizationWirelessDevicesChannelUtilizationByDevice(organizationID string, getOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams *sdk.GetOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams) (*resty.Response, error) {
	return w.service.GetOrganizationWirelessDevicesChannelUtilizationByDevice(organizationID, getOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams)
}

func getDeviceChannelUtilization(client DeviceService, devices map[Serial]*Device, period time.Duration, organizations []string) error {
	// Updated API endpoint for getting Channel Utilization data.
	// Previously, we used `GetNetworkNetworkHealthChannelUtilization`, but the Meraki SDK
	// did not properly parse its response, leading to loss of channel utilization data.
	// We are now using `GetOrganizationWirelessDevicesChannelUtilizationByDevice`.
	// However, the response format differs slightly:
	// - Bands are now labeled as 2.4/5 (GHz) instead of wifi0/wifi1.
	// - Utilization categories are now named `wifi/nonWifi` instead of `80211/non80211`.

	for _, orgID := range organizations {
		res, err := client.GetOrganizationWirelessDevicesChannelUtilizationByDevice(orgID, &sdk.GetOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams{
			// The API requires the interval to be at least 300s, and the timespan can't be less than the interval.
			// Since our max collection period is also 300s, we set both values to 300s.
			Timespan: 300,
			Interval: 300,
		})
		if err != nil {
			if res != nil {
				return fmt.Errorf("GetOrganizationWirelessDevicesChannelUtilizationByDevice for organization %s failed; [%d] %s. %w", orgID, res.StatusCode(), res.Body(), err)
			}
			return fmt.Errorf("GetOrganizationWirelessDevicesChannelUtilizationByDevice for organization %s failed: %w", orgID, err)
		}

		if res == nil {
			continue
		}

		var result sdk.ResponseOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDevice
		if err := json.Unmarshal(res.Body(), &result); err != nil {
			return fmt.Errorf("failed to unmarshal response body for organization %s: %w", orgID, err)
		}

		for _, d := range result {
			if d.ByBand == nil {
				continue
			}
			for _, band := range *d.ByBand {
				if device, ok := devices[Serial(d.Serial)]; ok {
					if device.bandUtilization == nil {
						device.bandUtilization = make(map[string]*sdk.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBand)
					}
					device.bandUtilization[band.Band] = &sdk.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBand{
						Wifi:    band.Wifi,
						NonWifi: band.NonWifi,
						Total:   band.Total,
					}
				}
			}
		}
	}
	return nil
}

func getDeviceLicenses(client *sdk.Client, organizationID string, devices map[Serial]*Device) error {
	val, res, err := client.Organizations.GetOrganizationLicenses(organizationID, &sdk.GetOrganizationLicensesQueryParams{})
	if err != nil {
		// Ignore 400 error for per-device licensing not supported
		if res != nil && res.StatusCode() == 400 && strings.Contains(string(res.Body()), "does not support per-device licensing") {
			return nil
		}

		if strings.Contains(err.Error(), "does not support per-device licensing") {
			return nil
		}

		if res != nil {
			return fmt.Errorf("GetOrganizationLicenses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		return fmt.Errorf("GetOrganizationLicenses failed; %w", err)
	}

	if val == nil {
		return errors.New("GetOrganizationLicenses returned nil response")
	}

	for i := range *val {
		license := (*val)[i]
		if device, ok := devices[Serial(license.DeviceSerial)]; ok {
			device.license = &license
		}
	}

	return nil
}

func deviceDetailsToMapstr(details *sdk.ResponseItemOrganizationsGetOrganizationDevices) mapstr.M {
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
		if device == nil || device.details == nil {
			continue
		}
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

		if device.bandUtilization != nil {
			for band, v := range device.bandUtilization {
				// Avoid nested object mappings
				metricBand := strings.ReplaceAll(band, ".", "_")
				metric[fmt.Sprintf("device.channel_utilization.%s.utilization_80211", metricBand)] = v.Wifi.Percentage
				metric[fmt.Sprintf("device.channel_utilization.%s.utilization_non_80211", metricBand)] = v.NonWifi.Percentage
				metric[fmt.Sprintf("device.channel_utilization.%s.utilization_total", metricBand)] = v.Total.Percentage
			}
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

	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}
