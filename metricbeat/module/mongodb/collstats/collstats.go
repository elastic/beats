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

package collstats

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/mongodb"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	mb.Registry.MustAddMetricSet("mongodb", "collstats", New,
		mb.WithHostParser(mongodb.ParseURL),
		mb.DefaultMetricSet(),
	)
}

// Metricset type defines all fields of the Metricset
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type Metricset struct {
	*mongodb.Metricset
}

// New creates a new instance of the Metricset
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := mongodb.NewMetricset(base)
	if err != nil {
		return nil, fmt.Errorf("could not create mongodb metricset: %w", err)
	}

	return &Metricset{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *Metricset) Fetch(reporter mb.ReporterV2) error {
	client, err := mongodb.NewClient(m.Metricset.Config, m.Module().Config().Timeout, 0)
	if err != nil {
		return fmt.Errorf("could not create mongodb client: %w", err)
	}

	defer func() {
		client.Disconnect(context.Background())
	}()

	if err != nil {
		return errors.Wrap(err, "could not get a list of databases")
	}

	// This info is only stored in 'admin' database
	db := client.Database("admin")
	res := db.RunCommand(context.Background(), bson.D{bson.E{Key: "top"}})
	if err = res.Err(); err != nil {
		return fmt.Errorf("'top' command failed: %w", err)
	}

	var result map[string]interface{}
	if err = res.Decode(&result); err != nil {
		return errors.Wrap(err, "could not decode mongo response")
	}

	if _, ok := result["totals"]; !ok {
		return errors.New("collection 'totals' key not found in mongodb response")
	}

	totals, ok := result["totals"].(map[string]interface{})
	if !ok {
		return errors.New("collection 'totals' are not a map")
	}

	for group, info := range totals {
		if group == "note" {
			continue
		}

		infoMap, ok := info.(map[string]interface{})
		if !ok {
			reporter.Error(errors.New("Unexpected data returned by mongodb"))
			continue
		}

		event, err := eventMapping(group, infoMap)
		if err != nil {
			reporter.Error(errors.Wrap(err, "Mapping of the event data filed"))
			continue
		}

		reporter.Event(mb.Event{
			MetricSetFields: event,
		})
	}

	return nil
}
