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

	serverhelper "github.com/menderesk/beats/v7/metricbeat/helper/server"
	httpserver "github.com/menderesk/beats/v7/metricbeat/helper/server/http"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
)

func init() {
	mb.Registry.MustAddMetricSet("prometheus", "remote_write",
		MetricSetBuilder(DefaultRemoteWriteEventsGeneratorFactory),
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// RemoteWriteEventsGenerator converts Prometheus Samples to a map of mb.Event
type RemoteWriteEventsGenerator interface {
	// Start must be called before using the generator
	Start()

	// GenerateEvents converts Prometheus Samples to a map of mb.Event
	GenerateEvents(metrics model.Samples) map[string]mb.Event

	// Stop must be called when the generator won't be used anymore
	Stop()
}

// RemoteWriteEventsGeneratorFactory creates a RemoteWriteEventsGenerator when instanciating a metricset
type RemoteWriteEventsGeneratorFactory func(ms mb.BaseMetricSet) (RemoteWriteEventsGenerator, error)

type MetricSet struct {
	mb.BaseMetricSet
	server          serverhelper.Server
	events          chan mb.Event
	promEventsGen   RemoteWriteEventsGenerator
	eventGenStarted bool
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig()
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	promEventsGen, err := DefaultRemoteWriteEventsGeneratorFactory(base)
	if err != nil {
		return nil, err
	}

	m := &MetricSet{
		BaseMetricSet:   base,
		events:          make(chan mb.Event),
		promEventsGen:   promEventsGen,
		eventGenStarted: false,
	}

	svc, err := httpserver.NewHttpServerWithHandler(base, m.handleFunc)
	if err != nil {
		return nil, err
	}
	m.server = svc
	return m, nil
}

// MetricSetBuilder returns a builder function for a new Prometheus remote_write metricset using
// the given namespace and event generator
func MetricSetBuilder(genFactory RemoteWriteEventsGeneratorFactory) func(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		config := defaultConfig()
		err := base.Module().UnpackConfig(&config)
		if err != nil {
			return nil, err
		}

		promEventsGen, err := genFactory(base)
		if err != nil {
			return nil, err
		}

		m := &MetricSet{
			BaseMetricSet:   base,
			events:          make(chan mb.Event),
			promEventsGen:   promEventsGen,
			eventGenStarted: false,
		}
		svc, err := httpserver.NewHttpServerWithHandler(base, m.handleFunc)
		if err != nil {
			return nil, err
		}
		m.server = svc

		return m, nil
	}
}

func (m *MetricSet) Run(reporter mb.PushReporterV2) {
	// Start event watcher
	m.server.Start()

	for {
		select {
		case <-reporter.Done():
			m.server.Stop()
			return
		case e := <-m.events:
			reporter.Event(e)
		}
	}
}

// Close stops the metricset
func (m *MetricSet) Close() error {
	if m.eventGenStarted {
		m.promEventsGen.Stop()
	}
	return nil
}

func (m *MetricSet) handleFunc(writer http.ResponseWriter, req *http.Request) {
	if !m.eventGenStarted {
		m.promEventsGen.Start()
		m.eventGenStarted = true
	}

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

	samples := protoToSamples(&protoReq)
	events := m.promEventsGen.GenerateEvents(samples)

	for _, e := range events {
		select {
		case <-req.Context().Done():
			return
		case m.events <- e:
		}
	}
	writer.WriteHeader(http.StatusAccepted)
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
