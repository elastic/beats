// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/tommyers-elastic/dashboard-api-go/v3/sdk"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {

	mb.Registry.MustAddMetricSet("meraki", "device_health", New)

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
	meraki_url    string
	meraki_apikey string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The meraki device_health metricset is beta.")

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
		meraki_url:    config.BaseURL,
		meraki_apikey: config.ApiKey,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {

	for _, org := range m.organizations {

		//Get Devices
		devices, err := GetDevices(m.client, org)
		if err != nil {
			return fmt.Errorf("getDevices() failed; %w", err)
		}

		//Get & Report Device Status
		deviceStatuses, err := getDeviceStatuses(m.client, org)
		if err != nil {
			return fmt.Errorf("getDeviceStatuses() failed; %w", err)
		}

		//Get mx device performance score
		mx_scores, err := getDevicePerformanceScores(m.client, devices)
		if err != nil {
			return fmt.Errorf("getDevicePerformanceScores() failed; %w", err)
		}
		reportDeviceStatusMetrics(reporter, org, devices, deviceStatuses, mx_scores)

		// //Get &  Report Organization Appliance Uplink
		appliance_val, appliance_res, appliance_err := m.client.Appliance.GetOrganizationApplianceUplinkStatuses(org, &meraki_api.GetOrganizationApplianceUplinkStatusesQueryParams{})
		if appliance_err != nil {
			return fmt.Errorf("Appliance.GetOrganizationApplianceUplinkStatuses failed; [%d] %s. %w", appliance_res.StatusCode(), appliance_res.Body(), appliance_err)
		}
		//Get & Report Device Uplink Status
		lossLatencyuplinks, err := getDeviceUplinkLossLatencyMetrics(m.client, org, m.BaseMetricSet.Module().Config().Period)
		if err != nil {
			return fmt.Errorf("getDeviceUplinkMetrics() failed; %w", err)
		}
		reportApplianceUplinkStatuses(reporter, org, devices, appliance_val, lossLatencyuplinks)

		//Get & Report Device License State
		cotermLicenses, perDeviceLicenses, systemsManagerLicense, err := getLicenseStates(m.client, org)
		if err != nil {
			return fmt.Errorf("getLicenseStates() failed; %w", err)
		}
		reportLicenseMetrics(reporter, org, cotermLicenses, perDeviceLicenses, systemsManagerLicense)

		//Get & Report Org Celluar Uplink Status
		cullular_val, cullular_res, cullular_err := m.client.CellularGateway.GetOrganizationCellularGatewayUplinkStatuses(org, &meraki_api.GetOrganizationCellularGatewayUplinkStatusesQueryParams{})
		if cullular_err != nil {
			return fmt.Errorf("CellularGateway.GetOrganizationCellularGatewayUplinkStatuses failed; [%d] %s. %w", cullular_res.StatusCode(), cullular_res.Body(), cullular_err)
		}
		reportCellularGatewayApplianceUplinkStatuses(reporter, org, devices, cullular_val)

		//Get Org Networks
		//Get Network Health by Org Network
		//Report NetworkHalthChannelUtilization
		orgNetworks, orgNetwork_res, orgNetwork_err := m.client.Organizations.GetOrganizationNetworks(org, &meraki_api.GetOrganizationNetworksQueryParams{})
		if orgNetwork_err != nil {
			return fmt.Errorf("Organizations.GetOrganizationNetworks failed; [%d] %s. %w", orgNetwork_res.StatusCode(), orgNetwork_res.Body(), orgNetwork_err)
		}
		networkHealthUtilizations, err := getNetworkHealthChannelUtilization(m.client, orgNetworks)
		if err != nil {
			return err
		}
		reportNetworkHealthChannelUtilization(reporter, org, devices, networkHealthUtilizations)

		// Get and Report Organization Wireless Devices Channel Utilization
		wireless_res, wireless_err := m.client.Devices.GetOrganizationWirelessDevicesChannelUtilizationByDevice(org, &meraki_api.GetOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams{})
		if wireless_err != nil {
			return fmt.Errorf("GetOrganizationWirelessDevicesChannelUtilizationByDevice failed; [%d] %s. %w", wireless_res.StatusCode(), wireless_res.Body(), wireless_err)
		}
		var wirelessDevices *meraki_api.ResponseOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDevice
		unmashal_err := json.Unmarshal(wireless_res.Body(), &wirelessDevices)
		if unmashal_err != nil {
			return fmt.Errorf("device_network_health_channel_utilization json umarshal failed; %w", unmashal_err)
		}
		reportWirelessDeviceChannelUtilization(reporter, org, devices, wirelessDevices)

		//Use Org Networks Retrieved above
		//Get VPN site_to_site
		//Report VPN site_to_site
		networkVPNSiteToSites, err := getNetworkApplianceVPNSiteToSite(m.client, orgNetworks)
		if err != nil {
			return err
		}
		reportNetwrokApplianceVPNSiteToSite(reporter, org, devices, networkVPNSiteToSites)

		//Get &  Report Organization License by Device
		license_val, license_res, license_err := m.client.Organizations.GetOrganizationLicenses(org, &meraki_api.GetOrganizationLicensesQueryParams{})
		if license_err != nil {
			return fmt.Errorf("Organizations.GetOrganizationLicenses failed; [%d] %s. %w", license_res.StatusCode(), license_res.Body(), license_err)
		}
		reportOrganizationDeviceLicenses(reporter, org, devices, license_val)

		//Use Org Networks Retrieved above
		//Get Network Ports
		//Report Network Ports By Device
		networkPorts, err := getNetworkAppliancePorts(m.client, orgNetworks)
		if err != nil {
			return err
		}
		reportNetwrokAppliancePorts(reporter, org, devices, networkPorts)

		//Get &  Report Organization License by Device
		switchPorts_val, switchPorts_res, switchPorts_err := m.client.Switch.GetOrganizationSwitchPortsBySwitch(org, &meraki_api.GetOrganizationSwitchPortsBySwitchQueryParams{})
		if switchPorts_err != nil {
			return fmt.Errorf("Switch.GetOrganizationSwitchPortsBySwitch failed; [%d] %s. %w", switchPorts_res.StatusCode(), switchPorts_res.Body(), switchPorts_err)
		}
		reportOrganizationDeviceSwitchPortBySwitch(reporter, org, devices, switchPorts_val)

		//Use Org Networks Retrieved above
		//Get Network Ports
		//Report Network Ports By Device
		switchPortStatusBySerials, err := getSwitchPortStatusBySerial(m.client, org)
		if err != nil {
			return err
		}
		reportSwitchPortStatusBySerial(reporter, org, devices, switchPortStatusBySerials)

	}

	return nil
}

func ReportMetricsForOrganization(reporter mb.ReporterV2, organizationID string, metrics ...[]mapstr.M) {

	for _, metricSlice := range metrics {
		for _, metric := range metricSlice {
			event := mb.Event{ModuleFields: mapstr.M{"organization_id": organizationID}}
			if ts, ok := metric["@timestamp"].(time.Time); ok {
				event.Timestamp = ts
				delete(metric, "@timestamp")
			}
			event.ModuleFields.Update(metric)
			reporter.Event(event)
		}
	}
}

func HttpGetRequestWithMerakiRetry(url string, token string, retry int) (*http.Response, error) {
	//https://developer.cisco.com/meraki/api-v1/get-device-appliance-performance/

	// Create a Bearer string by appending string access token
	var bearer = "Bearer " + token

	// Create a new request using http
	req, _ := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}
	response, err := client.Do(req)

	// Rate Limt Retry After Needed due to only 10 requests per second allowed by API
	//https://developer.cisco.com/meraki/api-v1/rate-limit/#rate-limit
	for i := 0; i < retry && response.StatusCode == 429; i++ {

		retryHeader := response.Header.Get("Retry-After")
		if _, err := strconv.Atoi(retryHeader); err == nil {
			log.Printf("Retry Limit Paused for %s second", retryHeader)
			time.ParseDuration(retryHeader + "s")
		}
		response, err = client.Do(req)

	}

	return response, err

}
