// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"fmt"

	meraki_api "github.com/meraki/dashboard-api-go/v3/sdk"
)

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
