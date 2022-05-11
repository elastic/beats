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

package replstatus

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/mongodb"
)

func init() {
	mb.Registry.MustAddMetricSet("mongodb", "replstatus", New,
		mb.WithHostParser(mongodb.ParseURL))
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	*mongodb.Metricset
}

// New creates a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := mongodb.NewMetricset(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	client, err := mongodb.NewClient(m.Metricset.Config, m.Module().Config().Timeout, readpref.PrimaryMode)
	if err != nil {
		return fmt.Errorf("could not create mongodb client: %w", err)
	}

	defer func() {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			m.Logger().Warn("client disconnection did not happen gracefully")
		}
	}()

	oplogInfo, err := getReplicationInfo(client)
	if err != nil {
		return fmt.Errorf("error getting replication info: %w", err)
	}

	replStatus, err := getReplicationStatus(client)
	if err != nil {
		return fmt.Errorf("error getting replication status: %w", err)
	}

	event := eventMapping(*oplogInfo, *replStatus)

	reporter.Event(mb.Event{MetricSetFields: event})

	return nil
}
