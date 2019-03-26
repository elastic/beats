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

package datastore

import (
	"context"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
)

func init() {
	mb.Registry.MustAddMetricSet("vsphere", "datastore", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	HostURL  *url.URL
	Insecure bool
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The vsphere datastore metricset is beta")

	config := struct {
		Username string `config:"username"`
		Password string `config:"password"`
		Insecure bool   `config:"insecure"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	u, err := url.Parse(base.HostData().URI)
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(config.Username, config.Password)

	return &MetricSet{
		BaseMetricSet: base,
		HostURL:       u,
		Insecure:      config.Insecure,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := govmomi.NewClient(ctx, m.HostURL, m.Insecure)
	if err != nil {
		return errors.Wrap(err, "error in NewClient")
	}

	defer client.Logout(ctx)

	c := client.Client

	// Create a view of Datastore objects
	mgr := view.NewManager(c)

	v, err := mgr.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"Datastore"}, true)
	if err != nil {
		return errors.Wrap(err, "error in CreateContainerView")
	}

	defer v.Destroy(ctx)

	// Retrieve summary property for all datastores
	var dst []mo.Datastore
	err = v.Retrieve(ctx, []string{"Datastore"}, []string{"summary"}, &dst)
	if err != nil {
		return errors.Wrap(err, "error in Retrieve")
	}

	for _, ds := range dst {
		var usedSpacePercent int64
		if ds.Summary.Capacity > 0 {
			usedSpacePercent = 100 * (ds.Summary.Capacity - ds.Summary.FreeSpace) / ds.Summary.Capacity
		}
		usedSpaceBytes := ds.Summary.Capacity - ds.Summary.FreeSpace

		event := common.MapStr{
			"name":   ds.Summary.Name,
			"fstype": ds.Summary.Type,
			"capacity": common.MapStr{
				"total": common.MapStr{
					"bytes": ds.Summary.Capacity,
				},
				"free": common.MapStr{
					"bytes": ds.Summary.FreeSpace,
				},
				"used": common.MapStr{
					"bytes": usedSpaceBytes,
					"pct":   usedSpacePercent,
				},
			},
		}

		reporter.Event(mb.Event{
			MetricSetFields: event,
		})
	}

	return nil
}
