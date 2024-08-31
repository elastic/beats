package cellular_gateway_uplink_status

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
	mb.Registry.MustAddMetricSet(meraki.ModuleName, "cellular_gateway_uplink_status", New)
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
	cfgwarn.Beta("The meraki cellular_gateway_uplink_status metricset is beta.")

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

		val, res, err := m.client.CellularGateway.GetOrganizationCellularGatewayUplinkStatuses(org, &meraki_api.GetOrganizationCellularGatewayUplinkStatusesQueryParams{})
		if err != nil {
			return fmt.Errorf("CellularGateway.GetOrganizationCellularGatewayUplinkStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}

		//fmt.Printf("CellularGateway.GetOrganizationCellularGatewayUplinkStatuses debug; [%d] %s.", res.StatusCode(), res.Body())

		reportApplianceUplinkStatuses(reporter, org, devices, val)
	}

	return nil
}

func reportApplianceUplinkStatuses(reporter mb.ReporterV2, organizationID string, devices map[meraki.Serial]*meraki.Device, responseCellularGatewayUplinkStatuses *meraki_api.ResponseCellularGatewayGetOrganizationCellularGatewayUplinkStatuses) {

	metrics := []mapstr.M{}

	for _, uplink := range *responseCellularGatewayUplinkStatuses {

		if device, ok := devices[meraki.Serial(uplink.Serial)]; ok {
			metric := mapstr.M{
				//this one should be deleted, I just want to see if it matches the device.network_id
				"cellular.gateway.uplink.networkd_id":      uplink.NetworkID,
				"cellular.gateway.uplink.last_reported_at": uplink.LastReportedAt,
				"cellular.gateway.address":                 device.Address,
				"cellular.gateway.firmware":                device.Firmware,
				"cellular.gateway.imei":                    device.Imei,
				"cellular.gateway.lan_ip":                  device.LanIP,
				"cellular.gateway.location":                device.Location,
				"cellular.gateway.mac":                     device.Mac,
				"cellular.gateway.model":                   device.Model,
				"cellular.gateway.name":                    device.Name,
				"cellular.gateway.network_id":              device.NetworkID,
				"cellular.gateway.notes":                   device.Notes,
				"cellular.gateway.product_type":            device.ProductType,
				"cellular.gateway.serial":                  device.Serial,
				"cellular.gateway.tags":                    device.Tags,
			}

			//Not sure if this is really needed on uplink status
			// for k, v := range device.Details {
			// 	metric[fmt.Sprintf("cellular.gateway.details.%s", k)] = v
			// }

			// #FIXME - Review might need to be like other uplinks
			for i, item := range *uplink.Uplinks {
				metrics = append(metrics, mapstr.Union(metric, mapstr.M{
					fmt.Sprintf("cellular.gateway.uplink.item_%d.apn", i):              item.Apn,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.connection_type", i):  item.ConnectionType,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.dns1", i):             item.DNS1,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.dns2", i):             item.DNS2,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.gateway", i):          item.Gateway,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.iccid", i):            item.Iccid,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.interface", i):        item.Interface,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.ip", i):               item.IP,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.model", i):            item.Model,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.provider", i):         item.Provider,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.public_ip", i):        item.PublicIP,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.signal_stat.rsrp", i): item.SignalStat.Rsrp,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.signal_stat.rsrq", i): item.SignalStat.Rsrq,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.signal_type", i):      item.SignalType,
					fmt.Sprintf("cellular.gateway.uplink.item_%d.status", i):           item.Status,
				}))

			}
		}
	}
	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}
