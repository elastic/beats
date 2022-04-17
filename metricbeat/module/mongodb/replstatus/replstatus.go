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
	"gopkg.in/mgo.v2"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/module/mongodb"
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
	*mongodb.MetricSet
}

// New creates a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := mongodb.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	// instantiate direct connections to each of the configured Mongo hosts
	mongoSession, err := mongodb.NewDirectSession(m.DialInfo)
	if err != nil {
		return errors.Wrap(err, "error creating new Session")
	}
	defer mongoSession.Close()

	mongoSession.SetMode(mgo.Strong, true)

	oplogInfo, err := getReplicationInfo(mongoSession)
	if err != nil {
		return errors.Wrap(err, "error getting replication info")
	}

	replStatus, err := getReplicationStatus(mongoSession)
	if err != nil {
		return errors.Wrap(err, "error getting replication status")
	}

	event := eventMapping(*oplogInfo, *replStatus)

	reporter.Event(mb.Event{MetricSetFields: event})

	return nil
}
