// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/logp"

	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

func init() {
	mb.Registry.MustAddMetricSet("meraki", "device_health", New)
}

type MetricSet struct {
	mb.BaseMetricSet
	logger        *logp.Logger
	client        *sdk.Client
	organizations []string
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The meraki device_health metricset is beta.")

	logger := base.Logger().Named(base.FullyQualifiedName())

	config := meraki.DefaultConfig()
	if err := base.Module().UnpackConfig(config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	logger.Debugf("loaded config: BaseURL=%s, DebugMode=%s, Organizations=%v, Period=%s", config.BaseURL, config.DebugMode, config.Organizations, config.Period)
	client, err := sdk.NewClientWithOptions(config.BaseURL, config.ApiKey, config.DebugMode, "Metricbeat Elastic")
	if err != nil {
		logger.Error("creating Meraki dashboard API client failed: %w", err)
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		logger:        logger,
		client:        client,
		organizations: config.Organizations,
	}, nil
}

func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	for _, org := range m.organizations {
		// some metrics require a 'timespan' parameter; we match this to our
		// collection interval to only collect new metric values
		collectionPeriod := m.BaseMetricSet.Module().Config().Period

		// First we get the list of all devices for this org (and their metadata).
		// Devices are uniquely identified by their serial number, which are used to
		// associate the metrics we collect later with the devices returned here.
		devices, err := getDevices(m.client, org)
		if err != nil {
			return fmt.Errorf("getDevices failed; %w", err)
		}

		// Now we continue to populate the device data structure with health
		// attributes/statuses/metrics etc in the following functions...
		err = getDeviceStatuses(m.client, org, devices)
		if err != nil {
			return fmt.Errorf("getDeviceStatuses failed; %w", err)
		}

		getDevicePerformanceScores(m.logger, m.client, devices)

		deviceService := &DeviceServiceWrapper{
			service: m.client.Devices,
		}

		err = getDeviceChannelUtilization(deviceService, devices, collectionPeriod, m.organizations)
		if err != nil {
			return fmt.Errorf("getDeviceChannelUtilization failed; %w", err)
		}

		err = getDeviceLicenses(m.client, org, devices)
		if err != nil {
			return fmt.Errorf("getDeviceLicenses failed; %w", err)
		}

		err = getDeviceUplinks(m.client, org, devices, collectionPeriod)
		if err != nil {
			return fmt.Errorf("getDeviceUplinks failed; %w", err)
		}

		err = getDeviceSwitchports(m.client, org, devices, collectionPeriod)
		if err != nil {
			return fmt.Errorf("getDeviceSwitchports failed; %w", err)
		}

		err = getDeviceVPNStatuses(m.client, org, devices, m.logger)
		if err != nil {
			m.logger.Errorf("GetVPNStatuses failed; %w", err)
			// continue so we still report the rest of the device health metrics
		}

		// Once we have collected _all_ the data and associated it with the correct device
		// we can report the various device health metrics. These functions are split up
		// in this way primarily to allow better separation of the code, but also because
		// each function here corresponds to a distinct set of reported metric events
		// i.e. there is one event per device, one event per uplink (but multiple uplinks per device),
		// one event per switchport (but multiple switchports per device), etc.
		reportDeviceMetrics(reporter, org, devices)
		reportUplinkMetrics(reporter, org, devices)
		reportSwitchportMetrics(reporter, org, devices)
	}

	return nil
}
