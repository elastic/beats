package network_health_channel_utilization

import (
	"fmt"
	"log"
	"strings"

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
	mb.Registry.MustAddMetricSet(meraki.ModuleName, "network_health_channel_utilization", New)
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
	cfgwarn.Beta("The meraki network_health_channel_utilization metricset is beta.")

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

		//productTypes := []string{"wireless"}
		//devices, err := meraki.GetDevicesByProductType(m.client, org, productTypes)
		devices, err := meraki.GetDevices(m.client, org)
		if err != nil {
			return err
		}

		orgNetworks, res, err := m.client.Organizations.GetOrganizationNetworks(org, &meraki_api.GetOrganizationNetworksQueryParams{})
		if err != nil {
			log.Printf("Organizations.GetOrganizationNetworks failed; [%d] %s.", res.StatusCode(), res.Body())
			return fmt.Errorf("Organizations.GetOrganizationNetworks failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		//fmt.Printf("\n#DEBUG Organizations.GetOrganizationNetworks; [%d] %s.\n###################", res.StatusCode(), res.Body())

		networkHealthUtilizations, err := getNetworkHealthChannelUtilization(m.client, orgNetworks)
		if err != nil {
			return err
		}

		reportNetworkHealthChannelUtilization(reporter, org, devices, networkHealthUtilizations)

	}

	return nil
}

func getNetworkHealthChannelUtilization(client *meraki_api.Client, networks *meraki_api.ResponseOrganizationsGetOrganizationNetworks) ([]*meraki_api.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, error) {

	var networkHealthUtilizations []*meraki_api.ResponseNetworksGetNetworkNetworkHealthChannelUtilization

	for _, network := range *networks {

		for _, product_type := range network.ProductTypes {

			if strings.Compare(product_type, "wireless") == 0 {

				networkHealthUtilization, _, err := client.Networks.GetNetworkNetworkHealthChannelUtilization(network.ID, &meraki_api.GetNetworkNetworkHealthChannelUtilizationQueryParams{})
				if err != nil {
					//return nil, fmt.Errorf("Networks.GetNetworkNetworkHealthChannelUtilization failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
					//log.Printf("\n#ERROR Networks.GetNetworkNetworkHealthChannelUtilization failed; [%d] %s.", res.StatusCode(), res.Body())
					//fmt.Printf("\n#ERROR Networks.GetNetworkNetworkHealthChannelUtilization failed; [%d] %s.\n###################", res.StatusCode(), res.Body())
				} else {
					//fmt.Printf("\n#DEBUG Networks.GetNetworkNetworkHealthChannelUtilization; [%d] %s. \n###################", res.StatusCode(), res.Body())
					networkHealthUtilizations = append(networkHealthUtilizations, networkHealthUtilization)
				}

			}
		}
	}

	return networkHealthUtilizations, nil
}

func reportNetworkHealthChannelUtilization(reporter mb.ReporterV2, organizationID string, devices map[meraki.Serial]*meraki.Device, networkHealthUtilizations []*meraki_api.ResponseNetworksGetNetworkNetworkHealthChannelUtilization) {

	metrics := []mapstr.M{}
	//Note: API does not specifiy if wireless devices only, so iterating through all network devices. API is to ambiguous
	//for _, device := range devices {

	//fmt.Printf("\n#DEBUG device Info: \nSerial: %s \nName: %s \nMode: %s \nNetworkID: %s \nProducttype: %s", device.Serial, device.Name, device.Model, device.NetworkID, device.ProductType)

	//fmt.Printf("\n#DEBUG reportNetworkHealthCHannelUtilization")

	for _, networkHealthUtil := range networkHealthUtilizations {
		//fmt.Printf("\n#DEBUG reportNetworkHealthCHannelUtilization For LOOP 1")

		for _, network := range *networkHealthUtil {

			//fmt.Printf("\n#DEBUG reportNetworkHealthCHannelUtilization For LOOP 2")

			metric := mapstr.M{
				"network.health.channel.radio.serial": network.Serial,
				"network.health.channel.radio.model":  network.Model,
				"network.health.channel.radio.tags":   network.Tags,
			}

			for _, wifi0 := range *network.Wifi0 {
				metric["network.health.channel.radio.wifi0.start_time"] = wifi0.StartTime
				metric["network.health.channel.radio.wifi0.end_time"] = wifi0.EndTime
				metric["network.health.channel.radio.wifi0.utilization80211"] = wifi0.Utilization80211
				metric["network.health.channel.radio.wifi0.utilizationNon80211"] = wifi0.UtilizationNon80211
				metric["network.health.channel.radio.wifi0.utilizationTotal"] = wifi0.UtilizationTotal
			}

			for _, wifi1 := range *network.Wifi1 {
				metric["network.health.channel.radio.wifi1.start_time"] = wifi1.StartTime
				metric["network.health.channel.radio.wifi1.end_time"] = wifi1.EndTime
				metric["network.health.channel.radio.wifi1.utilization80211"] = wifi1.Utilization80211
				metric["network.health.channel.radio.wifi1.utilizationNon80211"] = wifi1.UtilizationNon80211
				metric["network.health.channel.radio.wifi1.utilizationTotal"] = wifi1.UtilizationTotal
			}

			if device, ok := devices[meraki.Serial(network.Serial)]; ok {
				metric["device.address"] = device.Address
				metric["device.firmware"] = device.Firmware
				metric["device.imei"] = device.Imei
				metric["device.lan_ip"] = device.LanIP
				metric["device.location"] = device.Location
				metric["device.mac"] = device.Mac
				metric["device.model"] = device.Model
				metric["device.name"] = device.Name
				metric["device.network_id"] = device.NetworkID
				metric["device.notes"] = device.Notes
				metric["device.product_type"] = device.ProductType
				metric["device.serial"] = device.Serial
				metric["device.tags"] = device.Tags

			}
			metrics = append(metrics, metric)
		}
		// for k, v := range device.Details {
		// metric[fmt.Sprintf("device.details.%s", k)] = v
		// }
	}
	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}
