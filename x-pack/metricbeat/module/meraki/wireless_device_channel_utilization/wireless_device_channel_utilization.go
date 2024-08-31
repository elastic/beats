package wireless_device_channel_utilization

import (
	"encoding/json"
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
	mb.Registry.MustAddMetricSet(meraki.ModuleName, "wireless_device_channel_utilization", New)
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
	cfgwarn.Beta("The meraki wireless_device_channel_utilization metricset is beta.")

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

		devices, err := meraki.GetDevices(m.client, org)
		if err != nil {
			return err
		}

		res, err := m.client.Devices.GetOrganizationWirelessDevicesChannelUtilizationByDevice(org, &meraki_api.GetOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams{})
		if err != nil {
			return fmt.Errorf("GetOrganizationWirelessDevicesChannelUtilizationByDevice failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}

		//debug
		//fmt.Printf("\n #DEBUG Devices.GetOrganizationWirelessDevicesChannelUtilizationByDevice; [%d] %s.", res.StatusCode(), res.Body())

		var wirelessDevices WirelessDevicesChannelUtilizationByDevice
		err = json.Unmarshal(res.Body(), &wirelessDevices)
		if err != nil {
			return fmt.Errorf("device_network_health_channel_utilization json umarshal failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}

		reportWirelessDeviceChannelUtilization(reporter, org, devices, wirelessDevices)

	}

	return nil
}

func reportWirelessDeviceChannelUtilization(reporter mb.ReporterV2, organizationID string, devices map[meraki.Serial]*meraki.Device, wirelessDevices WirelessDevicesChannelUtilizationByDevice) {

	metrics := []mapstr.M{}

	for _, wirelessDevice := range wirelessDevices {

		if device, ok := devices[meraki.Serial(wirelessDevice.Serial)]; ok {

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

			// for k, v := range device.Details {
			// metric[fmt.Sprintf("wireless.device.details.%s", k)] = v
			// }

			metrics = append(metrics, metric)

		}

	}
	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}
