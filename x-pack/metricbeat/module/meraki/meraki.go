package meraki

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

// ModuleName is the name of this module.
const ModuleName = "meraki"

// func init() {

//  if err := mb.Registry.AddModule(ModuleName, newModule); err != nil {
//      panic(err)
//  }
// }

// // Config defines all required and optional parameters for meraki metricsets
// type Config struct {
//  Token         string   `config:"apiKey" validate:"nonzero,required"`
//  Organizations []string `config:"organizations" validate:"nonzero,required"`
// }

// // newModule adds validation that hosts is non-empty, a requirement to use the
// // mssql module.
// func newModule(base mb.BaseModule) (mb.Module, error) {
//  // Validate that at least one host has been specified.
//  var config Config
//  if err := base.UnpackConfig(&config); err != nil {
//      return nil, err
//  }

//  return &base, nil
// }

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

func GetDevicesByProductType(client *meraki_api.Client, organizationID string, productTypes []string) (map[Serial]*Device, error) {
	val, res, err := client.Organizations.GetOrganizationDevices(organizationID, &meraki_api.GetOrganizationDevicesQueryParams{
		ProductTypes: productTypes,
	})

	if err != nil {
		return nil, fmt.Errorf("GetOrganizationDevices failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	devices := make(map[Serial]*Device)
	for _, d := range *val {
		device := Device{
			Firmware:    d.Firmware,
			Imei:        d.Imei,
			LanIP:       d.LanIP,
			Location:    []*float64{d.Lng, d.Lat}, // (lon, lat) order is important!
			Mac:         d.Mac,
			Model:       d.Model,
			Name:        d.Name,
			NetworkID:   d.NetworkID,
			Notes:       d.Notes,
			ProductType: d.ProductType,
			Serial:      d.Serial,
			Tags:        d.Tags,
		}
		if d.Details != nil {
			for _, detail := range *d.Details {
				device.Details[detail.Name] = detail.Value
			}
		}
		devices[Serial(device.Serial)] = &device
	}

	return devices, nil
}

func GetDevices(client *meraki_api.Client, organizationID string) (map[Serial]*Device, error) {
	val, res, err := client.Organizations.GetOrganizationDevices(organizationID, &meraki_api.GetOrganizationDevicesQueryParams{})

	if err != nil {
		return nil, fmt.Errorf("GetOrganizationDevices failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
	}

	devices := make(map[Serial]*Device)
	for _, d := range *val {
		device := Device{
			Firmware:    d.Firmware,
			Imei:        d.Imei,
			LanIP:       d.LanIP,
			Location:    []*float64{d.Lng, d.Lat}, // (lon, lat) order is important!
			Mac:         d.Mac,
			Model:       d.Model,
			Name:        d.Name,
			NetworkID:   d.NetworkID,
			Notes:       d.Notes,
			ProductType: d.ProductType,
			Serial:      d.Serial,
			Tags:        d.Tags,
		}
		if d.Details != nil {
			for _, detail := range *d.Details {
				device.Details[detail.Name] = detail.Value
			}
		}
		devices[Serial(device.Serial)] = &device
	}

	return devices, nil
}

func HttpGetRequestWithMerakiRetry(url string, token string, retry int) (*http.Response, error) {
	//https://developer.cisco.com/meraki/api-v1/get-device-appliance-performance/
	//url = m.meraki_url + "/api/v1/devices/" + serial + "/appliance/performance"

	// Create a Bearer string by appending string access token
	var bearer = "Bearer " + token

	// Create a new request using http
	req, _ := http.NewRequest("GET", url, nil)

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}
	response, err := client.Do(req)

	// NEED TO ADD RETRY LOGIC
	//https://github.com/meraki/dashboard-api-go/blob/25b775d00e5c392677399e4fb1dfb0cfb67badce/sdk/api_client.go#L104C1-L123C3
	//https://developer.cisco.com/meraki/api-v1/rate-limit/#rate-limit

	for i := 0; i < retry && response.StatusCode == 429; i++ {

		retryHeader := response.Header.Get("Retry-After")
		if _, err := strconv.Atoi(retryHeader); err == nil {
			time.ParseDuration(retryHeader + "s")
			// Debug
			// Should we log this?
			fmt.Println("Paused for time:" + retryHeader + "s")
		}
		response, err = client.Do(req)

	}

	return response, err

}
