package device_health

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getDevicePerformanceScores(url string, token string, devices map[Serial]*Device) (map[Serial]*Device, map[Serial]*DevicePerformanceScore, error) {

	mx_devices := pruneDevicesForMxOnly(devices)

	scores := make(map[Serial]*DevicePerformanceScore)
	for _, device := range mx_devices {

		perf_score, status_code, err := getDevicePerformanceScoresBySerialId(url, token, device.Serial)
		if err != nil {
			return nil, nil, fmt.Errorf("getDevicePerformanceScores ->  getDevicePerformanceScoresBySerialId failed;  %w", err)
		}

		scores[Serial(device.Serial)] = &DevicePerformanceScore{
			PerformanceScore: perf_score,
			HttpStatusCode:   status_code,
		}
	}

	return mx_devices, scores, nil
}

func pruneDevicesForMxOnly(devices map[Serial]*Device) map[Serial]*Device {

	mx_devices := make(map[Serial]*Device)
	for k, v := range devices {
		if strings.Index(v.Model, "MX") == 0 {
			mx_devices[k] = v
		}
	}
	return mx_devices
}

func getDevicePerformanceScoresBySerialId(base_url string, token string, serial string) (float64, int, error) {
	//https://developer.cisco.com/meraki/api-v1/get-device-appliance-performance/
	url := base_url + "/api/v1/devices/" + serial + "/appliance/performance"

	response, err := HttpGetRequestWithMerakiRetry(url, token, 5)
	if err != nil {
		return 0, 0, fmt.Errorf("getDevicePerformanceScoresBySerialId HttpGetRequestWithMerakiRetry failed; %w", err)
	}

	if response.StatusCode == 204 {
		return -1, response.StatusCode, nil
	}

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("getDevicePerformanceScoresBySerialId io.ReadAll failed; %w", err)
	}

	var responseObject PerfScore
	err = json.Unmarshal(responseData, &responseObject)
	if err != nil {
		fmt.Printf("\nresponse.status=%d \nresponse.body=%s", response.StatusCode, response.Body)
		fmt.Printf("\nresponseData\n %s", responseData)
		return 0, 0, fmt.Errorf("getDevicePerformanceScoresBySerialId json.Unmarshal failed; %w", err)
	}

	//var tmp_float float64
	tmp_float := responseObject.PerformanceScore

	return tmp_float, response.StatusCode, nil

}

func reportPerformanceScoreMetrics(reporter mb.ReporterV2, organizationID string, devices map[Serial]*Device, devicePerformanceScores map[Serial]*DevicePerformanceScore) {
	devicePerformanceScoreMetrics := []mapstr.M{}
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

		if score, ok := devicePerformanceScores[serial]; ok {
			if score.HttpStatusCode == 204 {
				metric["device.performance.http_status_code"] = score.HttpStatusCode
			} else {
				metric["device.performance.score"] = score.PerformanceScore
			}

		}
		devicePerformanceScoreMetrics = append(devicePerformanceScoreMetrics, metric)
	}

	ReportMetricsForOrganization(reporter, organizationID, devicePerformanceScoreMetrics)

}
