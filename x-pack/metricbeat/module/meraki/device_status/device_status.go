// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_status

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {

	mb.Registry.MustAddMetricSet(meraki.ModuleName, "device_status", New)

}

type config struct {
	BaseURL       string   `config:"apiBaseURL"`
	ApiKey        string   `config:"apiKey"`
	DebugMode     string   `config:"apiDebugMode"`
	Organizations []string `config:"organizations"`
	// todo: device filtering?
}

func defaultConfig() *config {
	return &config{
		BaseURL:   "https://api.meraki.com",
		DebugMode: "false",
	}
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	logger        *logp.Logger
	client        *meraki_api.Client
	organizations []string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The meraki device_status metricset is beta.")

	logger := logp.NewLogger(base.FullyQualifiedName())

	config := defaultConfig()
	if err := base.Module().UnpackConfig(config); err != nil {
		return nil, err
	}

	logger.Debugf("loaded config: %v", config)
	client, err := meraki_api.NewClientWithOptions(config.BaseURL, config.ApiKey, config.DebugMode, "Metricbeat Elastic")
	if err != nil {
		logger.Error("creating Meraki dashboard API client failed: %w", err)
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		logger:        logger,
		client:        client,
		organizations: config.Organizations,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {

	for _, org := range m.organizations {
		//devices, err := getDevices(m.client, org)
		devices, err := meraki.GetDevices(m.client, org)
		if err != nil {
			return err
		}

		deviceStatuses, err := getDeviceStatuses(m.client, org)
		if err != nil {
			return err
		}

		reportDeviceStatusMetrics(reporter, org, devices, deviceStatuses)

	}

	return nil
}

func getDeviceStatuses(client *meraki_api.Client, organizationID string) (map[meraki.Serial]*DeviceStatus, error) {
	val, res, err := client.Organizations.GetOrganizationDevicesStatuses(organizationID, &meraki_api.GetOrganizationDevicesStatusesQueryParams{})

	if err != nil {
		return nil, fmt.Errorf("GetOrganizationDevicesStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	statuses := make(map[meraki.Serial]*DeviceStatus)
	for _, status := range *val {
		statuses[meraki.Serial(status.Serial)] = &DeviceStatus{
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

func reportDeviceStatusMetrics(reporter mb.ReporterV2, organizationID string, devices map[meraki.Serial]*meraki.Device, deviceStatuses map[meraki.Serial]*DeviceStatus) {
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

	meraki.ReportMetricsForOrganization(reporter, organizationID, deviceStatusMetrics)

}
