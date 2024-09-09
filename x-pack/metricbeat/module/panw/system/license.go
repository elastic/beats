// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const licenseQuery = "<request><license><info></info></license></request>"

func getLicenseEvents(m *MetricSet) ([]mb.Event, error) {

	var response LicenseResponse

	output, err := m.client.Op(licenseQuery, vsys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("empty response from PanOS")
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML response: %w", err)
	}

	if len(response.Result.Licenses) == 0 {
		m.logger.Warn("No licenses found in the response")
		return nil, nil
	}

	return formatLicenseEvents(m, response.Result.Licenses), nil
}

func formatLicenseEvents(m *MetricSet, licenses []License) []mb.Event {
	events := make([]mb.Event, 0, len(licenses))
	timestamp := time.Now()

	for _, license := range licenses {
		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"license.feature":     license.Feature,
				"license.description": license.Description, // Fixed typo
				"license.serial":      license.Serial,
				"license.issued":      license.Issued,
				"license.expires":     license.Expires,
				"license.expired":     license.Expired,
				"license.auth_code":   license.AuthCode,
			},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
			},
		}

		events = append(events, event)
	}

	return events
}
