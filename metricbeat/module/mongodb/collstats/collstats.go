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
	"errors"
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/mongodb"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	client, err := mongodb.NewClient(m.Metricset.Config, m.HostData().URI, m.Module().Config().Timeout, 0)
	if err != nil {
		return fmt.Errorf("could not create mongodb client: %w", err)
	}

	defer func() {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			m.Logger().Warn("client disconnection did not happen gracefully")
		}
	}()

	// This info is only stored in 'admin' database
	db := client.Database("admin")
	res := db.RunCommand(context.Background(), bson.D{bson.E{Key: "top"}})
	if err = res.Err(); err != nil {
		return fmt.Errorf("'top' command failed: %w", err)
	}

	var result map[string]interface{}
	if err = res.Decode(&result); err != nil {
		return fmt.Errorf("could not decode mongo response: %w", err)
	}

	if _, ok := result["totals"]; !ok {
		return errors.New("collection 'totals' key not found in mongodb response")
	}

	totals, ok := result["totals"].(map[string]interface{})
	if !ok {
		return errors.New("collection 'totals' are not a map")
	}

	if err = res.Err(); err != nil {
		return fmt.Errorf("'top' command failed: %w", err)
	}

	wg := &sync.WaitGroup{}

	for group, info := range totals {
		if group == "note" {
			continue
		}

		infoMap, ok := info.(map[string]interface{})
		if !ok {
			reporter.Error(errors.New("unexpected data returned by mongodb"))
			continue
		}

		wg.Add(1)
		go func(eventReporter mb.ReporterV2, mongoClient *mongo.Client, group string) {
			defer wg.Done()

			names, err := splitKey(group)
			if err != nil {
				eventReporter.Error(fmt.Errorf("splitting a collection key failed: %w", err))
				return
			}

			collStats, err := fetchCollStats(mongoClient, names[0], names[1])
			if err != nil {
				eventReporter.Error(fmt.Errorf("fetching collStats failed: %w", err))
				return
			}

			infoMap["stats"] = collStats

			event, err := eventMapping(group, infoMap)
			if err != nil {
				eventReporter.Error(fmt.Errorf("mapping of the event data failed: %w", err))
				return
			}

			eventReporter.Event(mb.Event{
				MetricSetFields: event,
			})
		}(reporter, client, group)
	}

	wg.Wait()

	return nil
}

func fetchCollStats(client *mongo.Client, dbName, collectionName string) (map[string]interface{}, error) {
	db := client.Database(dbName)
	colStats := db.RunCommand(context.Background(), bson.M{"collStats": collectionName})
	var statsRes map[string]interface{}
	if err := colStats.Decode(&statsRes); err != nil {
		return nil, fmt.Errorf("could not decode mongo response for database=%s, collection=%s: %w", dbName, collectionName, err)
	}

	return statsRes, nil
}
