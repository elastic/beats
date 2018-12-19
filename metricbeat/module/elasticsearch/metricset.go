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

package elasticsearch

import (
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	pathConfigKey = "path"
)

var (
	// HostParser parses host urls for RabbitMQ management plugin
	HostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: pathConfigKey,
	}.Build()
)

// MetricSet can be used to build other metric sets that query RabbitMQ
// management plugin
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
	XPack bool
	Log   *logp.Logger
}

// NewMetricSet creates an metric set that can be used to build other metric
// sets that query RabbitMQ management plugin
func NewMetricSet(base mb.BaseMetricSet, subPath string) (*MetricSet, error) {
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	http.SetURI(http.GetURI() + subPath)

	config := struct {
		XPack bool `config:"xpack.enabled"`
	}{
		XPack: false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.XPack {
		cfgwarn.Experimental("The experimental xpack.enabled flag in " + base.FullyQualifiedName() + " metricset is enabled.")
	}

	return &MetricSet{
		base,
		http,
		config.XPack,
		logp.NewLogger(ModuleName),
	}, nil
}
