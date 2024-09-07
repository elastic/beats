// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getLicenseEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<request><license><info></info></license></request>"
	var response LicenseResponse

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	events := formatLicenseEvents(m, response.Result.Licenses)

	return events, nil

}

func formatLicenseEvents(m *MetricSet, licenses []License) []mb.Event {
	events := make([]mb.Event, 0, len(licenses))

	currentTime := time.Now()

	for _, license := range licenses {
		event := mb.Event{MetricSetFields: mapstr.M{
			"license.feature":    license.Feature,
			"license.escription": license.Description,
			"license.serial":     license.Serial,
			"license.issued":     license.Issued,
			"license.expires":    license.Expires,
			"license.expired":    license.Expired,
			"license.auth_code":  license.AuthCode,
		},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
				"@Timestamp":      currentTime,
			}}

		events = append(events, event)
	}

	return events

}
