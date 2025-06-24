// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"

	"github.com/go-resty/resty/v2"
	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getDeviceVPNStatuses(client *sdk.Client, organizationID string, devices map[Serial]*Device) error {
	params := &sdk.GetOrganizationApplianceVpnStatusesQueryParams{}
	setStart := func(s string) { params.StartingAfter = s }

	doRequest := func() (*sdk.ResponseApplianceGetOrganizationApplianceVpnStatuses, *resty.Response, error) {
		return client.Appliance.GetOrganizationApplianceVpnStatuses(organizationID, params)
	}

	onError := func(err error, res *resty.Response) error {
		if res != nil {
			return fmt.Errorf("GetOrganizationApplianceVpnStatuses failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}

		return fmt.Errorf("GetOrganizationApplianceVpnStatuses failed; %w", err)
	}

	onSuccess := func(val *sdk.ResponseApplianceGetOrganizationApplianceVpnStatuses) error {
		if val != nil {
			for _, status := range *val {
				if device, ok := devices[Serial(status.DeviceSerial)]; ok {
					device.vpnStatus = &status
				}
			}
		}

		return nil
	}

	err := meraki.NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
	).GetAllPages()

	return err
}
