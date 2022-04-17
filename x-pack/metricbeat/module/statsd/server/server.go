// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
	serverhelper "github.com/menderesk/beats/v7/metricbeat/helper/server"
	"github.com/menderesk/beats/v7/metricbeat/helper/server/udp"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("statsd", "server", New, mb.DefaultMetricSet())
}

type errInvalidMapping struct {
	metricLabels []string
	attrLabels   []string
}

func (e errInvalidMapping) Error() string {
	return fmt.Sprintf(
		"labels in metric (%s) don't match labels attributes (%s)",
		strings.Join(e.metricLabels, ","),
		strings.Join(e.attrLabels, ","))
}

func newErrInvalidMapping(metricLabels []string, attrLabels []Label) error {
	attrLabelsStringSlice := make([]string, len(attrLabels))
	for i, attrLabel := range attrLabels {
		attrLabelsStringSlice[i] = attrLabel.Attr
	}

	if len(metricLabels) > 0 {
		metricLabels = metricLabels[1:]
	} else {
		metricLabels = []string{}
	}

	return errInvalidMapping{
		metricLabels: metricLabels,
		attrLabels:   attrLabelsStringSlice,
	}
}

// Config for the statsd server metricset.
type Config struct {
	TTL      time.Duration   `config:"ttl"`
	Mappings []StatsdMapping `config:"statsd.mappings"`
}

func defaultConfig() Config {
	return Config{
		TTL:      time.Second * 30,
		Mappings: nil,
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	server        serverhelper.Server
	serverStarted bool
	processor     *metricProcessor
	mappings      map[string]StatsdMapping
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	svc, err := udp.NewUdpServer(base)
	if err != nil {
		return nil, err
	}

	processor := newMetricProcessor(config.TTL)

	mappings, err := buildMappings(config.Mappings)
	if err != nil {
		return nil, fmt.Errorf("invalid mapping configuration for `statsd.mapping`: %w", err)
	}
	return &MetricSet{
		BaseMetricSet: base,
		server:        svc,
		mappings:      mappings,
		processor:     processor,
	}, nil
}

// Host returns the hostname or other module specific value that identifies a
// specific host or service instance from which to collect metrics.
func (b *MetricSet) Host() string {
	return b.server.(*udp.UdpServer).GetHost()
}

func buildMappings(config []StatsdMapping) (map[string]StatsdMapping, error) {
	mappings := make(map[string]StatsdMapping, len(config))
	replacer := strings.NewReplacer(".", `\.`, "<", "(?P<", ">", ">[^.]+)")
	for _, mapping := range config {
		regexPattern := replacer.Replace(mapping.Metric)
		var err error
		mapping.regex, err = regexp.Compile(fmt.Sprintf("^%s$", regexPattern))
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %s: %w", mapping.Metric, err)
		}

		var matchingLabels int
		names := mapping.regex.SubexpNames()
		if len(names)-1 != len(mapping.Labels) {
			return nil, newErrInvalidMapping(names, mapping.Labels)
		}

		repeatedLabelFields := make([]string, 0)
		uniqueLabelFields := make(map[string]struct{})
		for _, label := range mapping.Labels {
			for _, name := range names {
				if label.Attr == name {
					matchingLabels++
				}
			}

			if _, ok := uniqueLabelFields[label.Field]; !ok {
				uniqueLabelFields[label.Field] = struct{}{}
			} else {
				repeatedLabelFields = append(repeatedLabelFields, label.Field)
			}

			if label.Field == mapping.Value.Field {
				return nil, fmt.Errorf(`collision between label field "%s" and value field "%s"`, label.Field, mapping.Value.Field)
			}
		}

		if matchingLabels != len(mapping.Labels) {
			return nil, newErrInvalidMapping(names, mapping.Labels)
		}

		if len(uniqueLabelFields) != len(mapping.Labels) {
			return nil, fmt.Errorf(`repeated label fields "%s"`, strings.Join(repeatedLabelFields, ","))
		}

		mappings[mapping.Metric] = mapping

	}

	return mappings, nil
}

func (m *MetricSet) getEvents() []*mb.Event {
	groups := m.processor.GetAll()
	events := make([]*mb.Event, len(groups))

	for idx, tagGroup := range groups {

		mapstrTags := common.MapStr{}
		for k, v := range tagGroup.tags {
			mapstrTags[k] = v
		}

		sanitizedMetrics := common.MapStr{}
		for k, v := range tagGroup.metrics {
			eventMapping(k, v, sanitizedMetrics, m.mappings)
		}

		if len(sanitizedMetrics) == 0 {
			continue
		}

		events[idx] = &mb.Event{
			MetricSetFields: sanitizedMetrics,
			RootFields:      common.MapStr{"labels": mapstrTags},
			Namespace:       m.Module().Name(),
		}
	}
	return events
}

// ServerStart starts the underlying m.server
func (m *MetricSet) ServerStart() {
	if m.serverStarted {
		return
	}
	m.server.Start()
	m.serverStarted = true
}

// ServerStop stops the underlying m.server
func (m *MetricSet) ServerStop() {
	if !m.serverStarted {
		return
	}

	m.server.Stop()
	m.serverStarted = false
}

// Run method provides the module with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	period := m.Module().Config().Period

	// Start event watcher
	m.ServerStart()
	defer m.ServerStop()

	reportPeriod := time.NewTicker(period)
	for {
		select {
		case <-reporter.Done():
			return
		case <-reportPeriod.C:
			for _, e := range m.getEvents() {
				if e == nil {
					continue
				}

				reporter.Event(*e)
			}
		case msg := <-m.server.GetEvents():
			err := m.processor.Process(msg)
			if err != nil {
				reporter.Error(err)
			}
		}
	}
}
