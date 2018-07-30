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
	"fmt"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
)

func init() {
	mb.Registry.MustAddMetricSet("elasticsearch", "index_recovery", New,
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
	cfgwarn.Beta("The elasticsearch index_recovery metricset is beta")

	config := struct {
		ActiveOnly bool `config:"index_recovery.active_only"`
	}{
		ActiveOnly: true,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	localRecoveryPath := recoveryPath
	if config.ActiveOnly {
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

	isMaster, err := elasticsearch.IsMaster(m.HTTP, m.HostData().SanitizedURI+m.recoveryPath)
	if err != nil {
		r.Error(fmt.Errorf("Error fetch master info: %s", err))
		return
	}

	// Not master, no event sent
	if !isMaster {
		logp.Debug("elasticsearch", "Trying to fetch index recovery stats from a non master node.")
		return
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		r.Error(err)
		return
	}

	err = eventsMapping(r, content)
	if err != nil {
		r.Error(err)
	}
}
