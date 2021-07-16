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

package json

import (
	"encoding/json"
	"io/ioutil"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("http", "json", New,
		mb.WithHostParser(hostParser),
	)
}

const (
	// defaultScheme is the default scheme to use when it is not specified in the host config.
	defaultScheme = "http"

	// defaultPath is the dto use when it is not specified in the host config.
	defaultPath = ""
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		PathConfigKey: "path",
		DefaultPath:   defaultPath,
	}.Build()
)

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	namespace       string
	http            *helper.HTTP
	method          string
	body            string
	requestEnabled  bool
	responseEnabled bool
	jsonIsArray     bool
	deDotEnabled    bool
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		Namespace       string `config:"namespace" validate:"required"`
		Method          string `config:"method"`
		Body            string `config:"body"`
		RequestEnabled  bool   `config:"request.enabled"`
		ResponseEnabled bool   `config:"response.enabled"`
		JSONIsArray     bool   `config:"json.is_array"`
		DeDotEnabled    bool   `config:"dedot.enabled"`
	}{
		Method:          "GET",
		Body:            "",
		RequestEnabled:  false,
		ResponseEnabled: false,
		JSONIsArray:     false,
		DeDotEnabled:    false,
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	http.SetMethod(config.Method)
	http.SetBody([]byte(config.Body))

	return &MetricSet{
		BaseMetricSet:   base,
		namespace:       config.Namespace,
		method:          config.Method,
		body:            config.Body,
		http:            http,
		requestEnabled:  config.RequestEnabled,
		responseEnabled: config.ResponseEnabled,
		jsonIsArray:     config.JSONIsArray,
		deDotEnabled:    config.DeDotEnabled,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	response, err := m.http.FetchResponse()
	if err != nil {
		return err
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			m.Logger().Debug("error closing http body")
		}
	}()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	if m.jsonIsArray {
		var jsonBodyArr []common.MapStr
		if err = json.Unmarshal(body, &jsonBodyArr); err != nil {
			return err
		}

		for _, obj := range jsonBodyArr {
			event := m.processBody(response, obj)

			if reported := reporter.Event(event); !reported {
				m.Logger().Debug(errors.Errorf("error reporting event: %#v", event))
				return nil
			}
		}
	} else {
		var jsonBody common.MapStr
		if err = json.Unmarshal(body, &jsonBody); err != nil {
			return err
		}

		event := m.processBody(response, jsonBody)

		if reported := reporter.Event(event); !reported {
			m.Logger().Debug(errors.Errorf("error reporting event: %#v", event))
			return nil
		}
	}

	return nil
}
