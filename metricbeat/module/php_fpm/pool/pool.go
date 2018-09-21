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

package pool

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/helper"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"net/url"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("php_fpm", "pool", New,
 mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

const (
	defaultScheme = "http"
	defaultPath   = "/status"
	defaultQueryParams = "json"
)

// hostParser is used for parsing the configured php-fpm hosts.
var hostParser = parse.URLHostParserBuilder{
	DefaultScheme: defaultScheme,
	DefaultPath:   defaultPath,
	QueryParams:   defaultQueryParams,
	PathConfigKey: "status_path",
}.Build()


// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	*helper.HTTP
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The php_fpm pool metricset is beta")
	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		http,
	},nil
}

// Fetch gathers data for the pool metricset
func (m *MetricSet) Fetch() (common.MapStr, error) {
	conf := map[string]interface{}{}
	var l bool
	if err := m.Module().UnpackConfig(&conf); err != nil {
		return nil, err
	}
	t,ok := conf["status_full"]
	if ok {
		l, ok = t.(bool)
		if ok && l{
			u, err := url.Parse(m.GetURI())
			if err != nil {
				return nil, fmt.Errorf("error parsing URL: %v", err)
			}
			u, err = parse.SetQueryParams(u,"full")
			if err== nil {
				m.SetURI(u.String())
			}
		}
	}
content, err := m.HTTP.FetchContent()
	if err != nil {
		return nil, err
	}
	return eventsMapping(content)
}
