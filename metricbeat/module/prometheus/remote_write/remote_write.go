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

package remote_write

import (
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "remote_write", New)
}

type MetricSet struct {
	mb.BaseMetricSet
	server *http.Server
	events chan *mb.Event
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	tlsConfig, err := tlscommon.LoadTLSServerConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	m := MetricSet{
		BaseMetricSet: base,
		events:        make(chan *mb.Event),
	}

	httpServer := &http.Server{
		Addr: net.JoinHostPort(config.Host, strconv.Itoa(int(config.Port))),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.Logger().Debug(r.URL.Path)
			switch r.URL.Path {
			case "/write":
				m.handleWrite(w, r)
			}
		}),
	}

	if tlsConfig != nil {
		httpServer.TLSConfig = tlsConfig.BuildModuleConfig(config.Host)
	}

	m.server = httpServer
	return &m, nil
}

func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	go func() {
		if m.server.TLSConfig != nil {
			logp.Info("Starting HTTPS server on %s", m.server.Addr)
			//certificate is already loaded. That's why the parameters are empty
			err := m.server.ListenAndServeTLS("", "")
			if err != nil && err != http.ErrServerClosed {
				m.Logger().Errorf("Unable to start HTTPS server due to error: %v", err)
				return
			}
		} else {
			logp.Info("Starting HTTP server on %s", m.server.Addr)
			err := m.server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				m.Logger().Errorf("Unable to start HTTP server due to error: %v", err)
				return
			}
		}
	}()

	go func() {
		<-reporter.Done()
		m.server.Close()
		close(m.events)
	}()

	for e := range m.events {
		reporter.Event(*e)
	}
}

func (m *MetricSet) handleWrite(w http.ResponseWriter, r *http.Request) {
	compressed, err := ioutil.ReadAll(r.Body)
	if err != nil {
		m.Logger().Errorf("Read error %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		m.Logger().Errorf("Decode error %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &req); err != nil {
		m.Logger().Errorf("Unmarshal error %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// refactor, optimize
	samples := protoToSamples(&req)
	events := samplesToEvents(samples)

	for _, e := range events {
		m.events <- &e
	}
}

func protoToSamples(req *prompb.WriteRequest) model.Samples {
	var samples model.Samples
	for _, ts := range req.Timeseries {
		metric := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}

		for _, s := range ts.Samples {
			samples = append(samples, &model.Sample{
				Metric:    metric,
				Value:     model.SampleValue(s.Value),
				Timestamp: model.Time(s.Timestamp),
			})
		}
	}
	return samples
}
