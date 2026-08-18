// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package network_health

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/go-resty/resty/v2"
	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

func getNetworkVPNStats(client *sdk.Client, organizationID string, period time.Duration, logger *logp.Logger) ([]*sdk.ResponseItemApplianceGetOrganizationApplianceVpnStats, error) {
	var stats []*sdk.ResponseItemApplianceGetOrganizationApplianceVpnStats

	params := &sdk.GetOrganizationApplianceVpnStatsQueryParams{
		Timespan: period.Seconds(),
	}
	setStart := func(s string) { params.StartingAfter = s }

	doRequest := func() (*sdk.ResponseApplianceGetOrganizationApplianceVpnStats, *resty.Response, error) {
		return client.Appliance.GetOrganizationApplianceVpnStats(organizationID, params)
	}

	onError := func(err error, res *resty.Response) error {
		if res != nil {
			return fmt.Errorf("GetOrganizationApplianceVpnStats failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}

		return fmt.Errorf("GetOrganizationApplianceVpnStats failed; %w", err)
	}

	onSuccess := func(val *sdk.ResponseApplianceGetOrganizationApplianceVpnStats) error {
		if val != nil {
			for _, stat := range *val {
				stats = append(stats, &stat)
			}
		}
		return nil
	}

	err := meraki.NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logger,
	).GetAllPages()

	return stats, err
}
