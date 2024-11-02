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

package status

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/mongodb"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	mb.Registry.MustAddMetricSet("mongodb", "status", New,
		mb.WithHostParser(mongodb.ParseURL),
		mb.DefaultMetricSet(),
	)
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

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	client, err := mongodb.NewClient(m.Metricset.Config, m.HostData().URI, m.Module().Config().Timeout, readpref.PrimaryMode)
	if err != nil {
		return fmt.Errorf("could not create mongodb client: %w", err)
	}

	defer func() {
		if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
			m.Logger().Warn("client disconnection did not happen gracefully")
		}
	}()

	db := client.Database("admin")
	res := db.RunCommand(context.Background(), bson.M{"serverStatus": 1})
	if err = res.Err(); err != nil {
		return fmt.Errorf("failed to retrieve 'serverStatus': %w", err)
	}

	result := map[string]interface{}{}
	if err = res.Decode(&result); err != nil {
		return fmt.Errorf("could not decode 'serverStatus' response: %w", err)
	}

	t, ok := result["localTime"]
	if ok {
		mongoTime, castOk := t.(primitive.DateTime)
		if castOk {
			result["localTime"] = mongoTime.Time()
		}
		//omit any other situation. This value has low relevance
	}

	event := mb.Event{
		RootFields: mapstr.M{},
	}
	event.MetricSetFields, _ = schema.Apply(result)

	if v, err := event.MetricSetFields.GetValue("version"); err == nil {
		_, _ = event.RootFields.Put("service.version", v)
		_ = event.MetricSetFields.Delete("version")
	}
	if v, err := event.MetricSetFields.GetValue("process"); err == nil {
		_, _ = event.RootFields.Put("process.name", v)
		_ = event.MetricSetFields.Delete("process")
	}
	r.Event(event)

	return nil
}
