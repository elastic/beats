package device_uplink_loss_and_latency

import (
	"fmt"
	"time"

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
	mb.Registry.MustAddMetricSet(meraki.ModuleName, "device_uplink_loss_and_latency", New)
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
	cfgwarn.Beta("The meraki device_uplink_loss_and_latency metricset is beta.")

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

		uplinks, err := getDeviceUplinkMetrics(m.client, org, m.BaseMetricSet.Module().Config().Period)
		if err != nil {
			return err
		}

		reportDeviceUplinkMetrics(reporter, org, devices, uplinks)

	}

	return nil
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
			DeviceSerial: meraki.Serial(device.Serial),
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

func reportDeviceUplinkMetrics(reporter mb.ReporterV2, organizationID string, devices map[meraki.Serial]*meraki.Device, uplinks []*Uplink) {
	metrics := []mapstr.M{}

	for _, uplink := range uplinks {
		if device, ok := devices[uplink.DeviceSerial]; ok {
			metric := mapstr.M{
				"uplink.ip":         uplink.IP,
				"upliink.interface": uplink.Interface,
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
	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}
