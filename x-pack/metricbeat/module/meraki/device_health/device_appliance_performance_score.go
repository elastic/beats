package device_health

import (
	"fmt"
	"strings"

	meraki_api "github.com/tommyers-elastic/dashboard-api-go/v3/sdk"
)

func getDevicePerformanceScores(client *meraki_api.Client, devices map[Serial]*Device) (map[Serial]*DevicePerformanceScore, error) {

	mx_devices := pruneDevicesForMxOnly(devices)

	scores := make(map[Serial]*DevicePerformanceScore)
	for _, device := range mx_devices {

		score_val, score_res, score_err := client.Appliance.GetDeviceAppliancePerformance(device.Serial)
		if score_err != nil {
			return nil, fmt.Errorf("Appliance.GetDeviceAppliancePerformance failed;  %w", score_err)
		}

		if score_res.StatusCode() != 204 {
			scores[Serial(device.Serial)] = &DevicePerformanceScore{
				PerformanceScore: *score_val.PerfScore,
				HttpStatusCode:   200,
			}
		}
	}

	return scores, nil
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
