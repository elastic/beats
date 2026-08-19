// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package network_health

import (
	"fmt"

	"github.com/elastic/beats/v9/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v9/metricbeat/mb"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/logp"

	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

func init() {
	mb.Registry.MustAddMetricSet("meraki", "network_health", New)
}

type MetricSet struct {
	mb.BaseMetricSet
	logger        *logp.Logger
	client        *sdk.Client
	organizations []string
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logger := base.Logger().Named(base.FullyQualifiedName())
	logger.Warn(cfgwarn.Beta("The meraki network_health metricset is beta."))

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
	// some metrics require a 'timespan' parameter; we match this to our
	// collection interval to only collect new metric values
	collectionPeriod := m.BaseMetricSet.Module().Config().Period

	for _, org := range m.organizations {
		stats, err := getNetworkVPNStats(m.client, org, collectionPeriod, m.logger)
		if err != nil {
			return fmt.Errorf("getNetworkVPNStats failed; %w", err)
		}

		networks := make(map[ID]*Network)
		for _, vpn := range stats {
			if vpn != nil {
				networks[ID(vpn.NetworkID)] = &Network{
					id:       ID(vpn.NetworkID),
					name:     vpn.NetworkName,
					vpnPeers: vpn.MerakiVpnpeers,
				}
			}
		}

		reportNetworkMetrics(reporter, org, networks)
	}

	return nil
}
