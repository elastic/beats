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
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/helper/server"
)

var errInvalidPacket = errors.New("invalid statsd packet")

type metricProcessor struct {
	registry      metrics.Registry
	reservoirSize int
}

type statsdMetric struct {
	name       string
	metricType string
	sampleRate string
	value      string
	tags       string
}

func parseSingle(b []byte) (statsdMetric, error) {
	// format: <metric name>:<value>|<type>[|@samplerate][|tags]

	s := statsdMetric{}

	parts := bytes.SplitN(b, []byte("|"), 4)
	if len(parts) < 2 {
		return s, errInvalidPacket
	}

	if len(parts) > 2 && len(parts[2]) > 0 && parts[2][0] == '@' {
		s.sampleRate = string(parts[2][1:])

		if len(parts) > 3 {
			parts = [][]byte{parts[0], parts[1], parts[3]}
		} else {
			parts = [][]byte{parts[0], parts[1]}
		}
	}

	nameSplit := bytes.SplitN(parts[0], []byte{':'}, 2)
	if len(nameSplit) != 2 {
		return s, errInvalidPacket
	}

	s.name = string(nameSplit[0])
	s.value = string(nameSplit[1])
	s.metricType = string(parts[1])

	if len(parts) > 2 {
		s.tags = string(parts[2])
	}
	return s, nil

}

// parse will parse a statsd metric into its components
func parse(b []byte) ([]statsdMetric, error) {
	metrics := []statsdMetric{}
	for _, rawMetric := range bytes.Split(b, []byte("\n")) {
		if len(rawMetric) > 0 {
			metric, err := parseSingle(rawMetric)
			if err != nil {
				return metrics, err
			}
			metrics = append(metrics, metric)
		}
	}
	return metrics, nil
}

func newMetricProcessor(reservoirSize int) *metricProcessor {
	return &metricProcessor{
		registry:      metrics.NewRegistry(),
		reservoirSize: reservoirSize,
	}
}

func (p *metricProcessor) processSingle(m statsdMetric) error {
	if len(m.value) < 1 {
		return nil
	}

	switch m.metricType {
	case "c":
		c := metrics.GetOrRegisterCounter(m.name, p.registry)
		v, err := strconv.ParseInt(m.value, 10, 64)
		if err != nil {
			return err
		}
		// inc/dec or set
		if m.value[0] == '+' || m.value[0] == '-' {
			c.Inc(v)
		} else {
			c.Clear()
			c.Inc(v)
		}
	case "g":
		c := metrics.GetOrRegisterGaugeFloat64(m.name, p.registry)
		v, err := strconv.ParseFloat(m.value, 64)
		if err != nil {
			return err
		}
		c.Update(v)
	case "ms":
		c := metrics.GetOrRegisterTimer(m.name, p.registry)
		v, err := strconv.ParseFloat(m.value, 64)
		if err != nil {
			return err
		}
		c.Update(time.Duration(v))
	case "h": // TODO: can these be floats?
		c := metrics.GetOrRegisterHistogram(m.name, p.registry, metrics.NewUniformSample(p.reservoirSize))
		v, err := strconv.ParseInt(m.value, 10, 64)
		if err != nil {
			return err
		}
		c.Update(v)
	default:
		logp.NewLogger("statsd").Debugf("metric type '%s' is not supported", m.metricType)
		// ignore others
		// no support for sets, for example
	}
	return nil
}

func (p *metricProcessor) Process(event server.Event) error {
	bytesRaw, ok := event.GetEvent()[server.EventDataKey]
	if !ok {
		return errors.New("Unable to retrieve response bytes")
	}

	b, _ := bytesRaw.([]byte)
	if len(b) == 0 {
		return errors.New("Request has no data")
	}

	metrics, err := parse(b)
	if err != nil {
		return err
	}

	for _, m := range metrics {
		if err := p.processSingle(m); err != nil {
			return err
		}
	}

	return nil
}

func (p *metricProcessor) GetAll() common.MapStr {
	fields := common.MapStr{}
	for k, v := range p.registry.GetAll() {
		metric := common.MapStr{}
		for mk, mv := range v {
			metric[mk] = mv
		}
		fields[k] = metric
	}

	flattened := common.MapStr{}
	for k, v := range fields.Flatten() {
		// replace . with _ and % with p
		flattened[strings.Replace(strings.Replace(k, ".", "_", -1), "%", "p", -1)] = v
	}
	return flattened
}
