// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	licenseQuery   = "<request><license><info></info></license></request>"
	panwDateFormat = "January 2, 2006"
)

func getLicenseEvents(m *MetricSet) ([]mb.Event, error) {

	var response LicenseResponse

	output, err := m.client.Op(licenseQuery, panw.Vsys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("empty response from PanOS for license query")
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML response: %w", err)
	}

	if len(response.Result.Licenses) == 0 {
		m.logger.Warn("No licenses found in the response")
		return []mb.Event{}, nil
	}

	return formatLicenseEvents(m, response.Result.Licenses), nil
}

func formatLicenseEvents(m *MetricSet, licenses []License) []mb.Event {
	events := make([]mb.Event, 0, len(licenses))
	timestamp := time.Now().UTC()

	for _, license := range licenses {
		expired, err := panw.StringToBool(license.Expired)
		if err != nil {
			m.logger.Warn("Failed to convert expired value %s to boolean: %s. Defaulting to false.", license.Expired, err)
		}

		//
		// <issued>March 20, 2024</issued>
		// <expires>May 27, 2025</expires> or <expires>Never</expires>
		//
		issued, err := time.Parse(panwDateFormat, license.Issued)
		if err != nil {
			m.logger.Warn("Failed to parse license issued date %s: %s", license.Issued, err)
		}
		neverExpires := false
		expires, err := time.Parse(panwDateFormat, license.Expires)
		// The value of license.Expires is "never" when the license never expires
		if err != nil {
			if strings.ToLower(license.Expires) == "never" {
				neverExpires = true
			} else {
				m.logger.Warn("Failed to parse license expire date %s: %s", license.Expires, err)
			}
		}

		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"license.feature":       license.Feature,
				"license.description":   license.Description,
				"license.serial":        license.Serial,
				"license.issued":        issued.Format(time.RFC3339),
				"license.never_expires": neverExpires,
				"license.expired":       expired,
				"license.auth_code":     license.AuthCode,
			},
			RootFields: panw.MakeRootFields(m.config.HostIp),
		}
		// only set the expires field if the license expires
		if !neverExpires {
			event.MetricSetFields["license.expires"] = expires.Format(time.RFC3339)
		}

		events = append(events, event)
	}

	return events
}
