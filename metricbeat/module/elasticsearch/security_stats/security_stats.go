// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package security_stats

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/elastic-agent-libs/version"
)

func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "security_stats", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.WithNamespace("elasticsearch.security.stats"),
	)
}

const (
	securityStatsPath = "/_security/stats"
)

// MetricSet collects per-node Elasticsearch security statistics, currently the
// document-level security (DLS) cache counters returned by the
// /_security/stats REST endpoint.
type MetricSet struct {
	*elasticsearch.MetricSet
	lastUnavailableMessageTimestamp time.Time
}

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := elasticsearch.NewMetricSet(base, securityStatsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

// Fetch retrieves stats from the /_security/stats endpoint and reports one
// event per node returned in the response.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	shouldSkip, err := m.ShouldSkipFetch()
	if err != nil {
		return err
	}
	if shouldSkip {
		return nil
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.GetServiceURI())
	if err != nil {
		return err
	}

	unavailableMessage, err := m.checkAvailability(info.Version.Number)
	if err != nil {
		return fmt.Errorf("error determining if %s is available: %w", m.FullyQualifiedName(), err)
	}
	if unavailableMessage != "" {
		// Throttle the message so we don't spam logs: when security is off it
		// stays off for the lifetime of the process. Hourly still gives an
		// operational heartbeat without flooding the log.
		if time.Since(m.lastUnavailableMessageTimestamp) > time.Hour {
			m.lastUnavailableMessageTimestamp = time.Now()
			m.Logger().Debug(unavailableMessage)
		}
		return nil
	}

	content, err := m.FetchContent()
	if err != nil {
		return err
	}

	// One bulk /_nodes call per scrape, shared across all per-node events. A
	// failure here is non-fatal: we still emit the security counters with just
	// the node id, and surface the failure via the joined return error so it
	// lands in self-monitoring rather than being silently swallowed (matches
	// the per-node error pattern in node_stats).
	var fetchErrs []error
	enrichment, err := m.GetNodesEnrichment()
	if err != nil {
		fetchErrs = append(fetchErrs,
			fmt.Errorf("could not fetch node enrichment for %s: %w", m.FullyQualifiedName(), err))
	}

	if err := eventsMapping(r, info, content, m.XPackEnabled, enrichment); err != nil {
		fetchErrs = append(fetchErrs, err)
	}
	return errors.Join(fetchErrs...)
}

// checkAvailability returns a non-empty message when the metricset has nothing
// to report against the target cluster, either because the cluster predates
// the introduction of /_security/stats or because the security feature is
// turned off (xpack.security.enabled=false unregisters the /_security/*
// routes entirely, so the endpoint would 400).
//
// The version check is a pure comparison against info we already have, so it
// runs first to short-circuit any HTTP work on too-old clusters. The feature
// check that follows mirrors the pattern used by ccr and ml_job: probe the
// state proactively via /_xpack rather than discover it reactively from a
// failed stats request, so the operator-facing log message can be specific
// and so we don't keep hitting an endpoint we know won't answer.
//
// Unlike ccr and ml_job, we don't gate on license: /_security/stats returns
// the DLS bitset cache structure (zeroed) even on a basic license. DLS as a
// feature is gold+, but the stats endpoint itself isn't license-gated. If a
// future security subsystem surfaced here is license-gated, this function is
// the right place to add the corresponding elasticsearch.GetLicense check.
func (m *MetricSet) checkAvailability(currentElasticsearchVersion *version.V) (string, error) {
	if !elastic.IsFeatureAvailable(currentElasticsearchVersion, elasticsearch.SecurityStatsAPIAvailableVersion) {
		return "the " + m.FullyQualifiedName() + " is only supported with Elasticsearch >= " +
			elasticsearch.SecurityStatsAPIAvailableVersion.String() + ". " +
			"You are currently running Elasticsearch " + currentElasticsearchVersion.String() + ".", nil
	}

	xpack, err := elasticsearch.GetXPack(m.HTTP, m.GetServiceURI())
	if err != nil {
		return "", fmt.Errorf("error determining xpack features: %w", err)
	}
	if !xpack.Features.Security.Enabled {
		return "the security feature is not enabled on your Elasticsearch cluster.", nil
	}

	return "", nil
}
