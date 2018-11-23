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

package index_recovery

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func init() {
	mb.Registry.MustAddMetricSet(elasticsearch.ModuleName, "index_recovery", New,
		mb.WithHostParser(elasticsearch.HostParser),
		mb.WithNamespace("elasticsearch.index.recovery"),
	)
}

const (
	recoveryPath = "/_recovery"
)

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*elasticsearch.MetricSet
	recoveryPath string
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The " + base.FullyQualifiedName() + " metricset is beta")

	config := struct {
		ActiveOnly bool `config:"index_recovery.active_only"`
		XPack      bool `config:"xpack.enabled"`
	}{
		ActiveOnly: true,
		XPack:      false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	localRecoveryPath := recoveryPath
	if !config.XPack && config.ActiveOnly {
		localRecoveryPath = localRecoveryPath + "?active_only=true"
	}

	ms, err := elasticsearch.NewMetricSet(base, localRecoveryPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms, recoveryPath: localRecoveryPath}, nil
}

// Fetch gathers stats for each index from the _stats API
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	isMaster, err := elasticsearch.IsMaster(m.HTTP, m.getServiceURI())
	if err != nil {
		err = errors.Wrap(err, "error determining if connected Elasticsearch node is master")
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	// Not master, no event sent
	if !isMaster {
		m.Log.Debug("trying to fetch index recovery stats from a non-master node")
		return
	}

	info, err := elasticsearch.GetInfo(m.HTTP, m.getServiceURI())
	if err != nil {
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		elastic.ReportAndLogError(err, r, m.Log)
		return
	}

	if m.MetricSet.XPack {
		err = eventsMappingXPack(r, m, *info, content)
	} else {
		err = eventsMapping(r, *info, content)
	}

	if err != nil {
		m.Log.Error(err)
		return
	}
}

func (m *MetricSet) getServiceURI() string {
	return m.HostData().SanitizedURI + m.recoveryPath

}
