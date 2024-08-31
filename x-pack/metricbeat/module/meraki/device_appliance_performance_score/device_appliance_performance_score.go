package device_appliance_performance_score

import (
	"encoding/json"
	"fmt"
	"io"
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
	mb.Registry.MustAddMetricSet(meraki.ModuleName, "device_appliance_performance_score", New)
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
	cfgwarn.Beta("The meraki device_appliance_performance_score metricset is beta.")

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

		devices, err := meraki.GetDevices(m.client, org)
		if err != nil {
			return err
		}

		mx_devices := pruneDevicesForMxOnly(devices)
		mx_scores, _ := getDevicePerformanceScores(m.meraki_url, m.meraki_apikey, mx_devices)

		reportPerformanceScoreMetrics(reporter, org, mx_devices, mx_scores)

	}

	return nil
}

func getDevicePerformanceScores(url string, token string, mx_devices map[meraki.Serial]*meraki.Device) (map[meraki.Serial]*DevicePerformanceScore, error) {

	scores := make(map[meraki.Serial]*DevicePerformanceScore)
	for _, device := range mx_devices {

		perf_score, status_code := getDevicePerformanceScoresBySerialId(url, token, device.Serial)

		scores[meraki.Serial(device.Serial)] = &DevicePerformanceScore{
			PerformanceScore: perf_score,
			HttpStatusCode:   status_code,
		}
	}

	return scores, nil
}

func pruneDevicesForMxOnly(devices map[meraki.Serial]*meraki.Device) map[meraki.Serial]*meraki.Device {

	mx_devices := make(map[meraki.Serial]*meraki.Device)
	for k, v := range devices {
		if strings.Index(v.Model, "MX") == 0 {
			mx_devices[k] = v
		}
	}
	return mx_devices
}

func getDevicePerformanceScoresBySerialId(base_url string, token string, serial string) (float64, int) {
	//https://developer.cisco.com/meraki/api-v1/get-device-appliance-performance/
	url := base_url + "/api/v1/devices/" + serial + "/appliance/performance"

	// NEED TO ADD RETRY LOGIC
	//https://github.com/meraki/dashboard-api-go/blob/25b775d00e5c392677399e4fb1dfb0cfb67badce/sdk/api_client.go#L104C1-L123C3
	//https://developer.cisco.com/meraki/api-v1/rate-limit/#rate-limit

	response, err := meraki.HttpGetRequestWithMerakiRetry(url, token, 5)

	if err != nil {
		log.Fatal(err)
	}

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var responseObject PerfScore
	json.Unmarshal(responseData, &responseObject)

	//var tmp_float float64
	tmp_float := responseObject.PerformanceScore

	return tmp_float, response.StatusCode

}

func reportPerformanceScoreMetrics(reporter mb.ReporterV2, organizationID string, devices map[meraki.Serial]*meraki.Device, devicePerformanceScores map[meraki.Serial]*DevicePerformanceScore) {
	devicePerformanceScoreMetrics := []mapstr.M{}
	for serial, device := range devices {
		metric := mapstr.M{
			"device.appliance.address":      device.Address,
			"device.appliance.firmware":     device.Firmware,
			"device.appliance.imei":         device.Imei,
			"device.appliance.lan_ip":       device.LanIP,
			"device.appliance.location":     device.Location,
			"device.appliance.mac":          device.Mac,
			"device.appliance.model":        device.Model,
			"device.appliance.name":         device.Name,
			"device.appliance.network_id":   device.NetworkID,
			"device.appliance.notes":        device.Notes,
			"device.appliance.product_type": device.ProductType,
			"device.appliance.serial":       device.Serial,
			"device.appliance.tags":         device.Tags,
		}

		for k, v := range device.Details {
			metric[fmt.Sprintf("device.appliance.details.%s", k)] = v
		}

		if score, ok := devicePerformanceScores[serial]; ok {
			if score.HttpStatusCode == 204 {
				metric["device.appliance.performance.http_status_code"] = score.HttpStatusCode
			} else {
				metric["device.appliance.performance.score"] = score.PerformanceScore
			}

		}
		devicePerformanceScoreMetrics = append(devicePerformanceScoreMetrics, metric)
	}

	meraki.ReportMetricsForOrganization(reporter, organizationID, devicePerformanceScoreMetrics)

}
