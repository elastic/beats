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
	"io/ioutil"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"

	serverhelper "github.com/elastic/beats/v7/metricbeat/helper/server"
	httpserver "github.com/elastic/beats/v7/metricbeat/helper/server/http"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "remote_write", New)
}

type MetricSet struct {
	mb.BaseMetricSet
	server serverhelper.Server
	events chan *mb.Event
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
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
	return m, nil
}

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
		}
	}
}

func (m *MetricSet) handleFunc(writer http.ResponseWriter, req *http.Request) {
	compressed, err := ioutil.ReadAll(req.Body)
	if err != nil {
		m.Logger().Errorf("Read error %v", err)
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	reqBuf, err := snappy.Decode(nil, compressed)
	if err != nil {
		m.Logger().Errorf("Decode error %v", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	var protoReq prompb.WriteRequest
	if err := proto.Unmarshal(reqBuf, &protoReq); err != nil {
		m.Logger().Errorf("Unmarshal error %v", err)
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// refactor, optimize
	samples := protoToSamples(&protoReq)
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
