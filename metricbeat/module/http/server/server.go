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

package server

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper/server"

	serverhelper "github.com/elastic/beats/v7/metricbeat/helper/server"
	httpserver "github.com/elastic/beats/v7/metricbeat/helper/server/http"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("http", "server", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	server    serverhelper.Server
	processor *metricProcessor
	events    chan *mb.Event
	errors    chan error
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultHttpServerConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	m := &MetricSet{
		BaseMetricSet: base,
	}
	svc, err := httpserver.NewHttpServer(base, m.handleFunc)
	if err != nil {
		return nil, err
	}
	m.server = svc
	m.processor = NewMetricProcessor(config.Paths, config.DefaultPath)
	return m, nil
}

// Run method provides the module with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	// Start event watcher
	m.server.Start()

	for {
		select {
		case <-reporter.Done():
			m.server.Stop()
			close(m.events)
			return
		case e := <-m.events:
			reporter.Event(*e)
		case err := <-m.errors:
			reporter.Error(err)
		}
	}
}

func (m *MetricSet) handleFunc(writer http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		meta := common.MapStr{
			"path": req.URL.String(),
		}

		contentType := req.Header.Get("Content-Type")
		if contentType != "" {
			meta["Content-Type"] = contentType
		}

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			logp.Err("Error reading body: %v", err)
			http.Error(writer, "Unexpected error reading request payload", http.StatusBadRequest)
			return
		}

		payload := common.MapStr{
			server.EventDataKey: body,
		}

		fields, err := m.processor.Process(payload, meta)
		if err != nil {
			m.errors <- err
		} else {
			event := mb.Event{}
			ns, ok := fields[mb.NamespaceKey].(string)
			if ok {
				ns = fmt.Sprintf("http.%s", ns)
				delete(fields, mb.NamespaceKey)
			}
			event.MetricSetFields = fields
			event.Namespace = ns
			m.events <- &event
		}

		writer.WriteHeader(http.StatusAccepted)

	case "GET":
		writer.WriteHeader(http.StatusOK)
		if req.TLS != nil {
			writer.Write([]byte("HTTPS Server accepts data via POST"))
		} else {
			writer.Write([]byte("HTTP Server accepts data via POST"))
		}

	}
}
