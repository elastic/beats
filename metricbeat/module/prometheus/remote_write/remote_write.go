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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
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
			case "/read":
				m.handleRead(w, r)
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

func (m *MetricSet) handleRead(w http.ResponseWriter, r *http.Request) {
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

	var req prompb.ReadRequest
	if err := proto.Unmarshal(reqBuf, &req); err != nil {
		m.Logger().Errorf("Unmarshal error %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := m.handleReadRequest(&req)
	if err != nil {
		m.Logger().Errorf("Request error %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("Content-Encoding", "snappy")

	compressed = snappy.Encode(nil, data)
	if _, err := w.Write(compressed); err != nil {
		m.Logger().Errorf("Error writing response %v", err)
	}
}

func (m *MetricSet) handleReadRequest(req *prompb.ReadRequest) (*prompb.ReadResponse, error) {
	client, err := elasticsearch.NewClient(elasticsearch.ClientSettings{
		URL:              "http://localhost:9200",
		Timeout:          60 * time.Second,
		CompressionLevel: 3,
	}, nil)
	if err != nil {
		return nil, err
	}

	timeseries := map[string]*prompb.TimeSeries{}
	for _, q := range req.Queries {
		var metricName string
		for _, m := range q.Matchers {
			if m.Name == model.MetricNameLabel {
				metricName = m.Value
			}
		}
		if metricName == "" {
			return nil, errors.New("query had no metric")
		}

		query, err := m.buildQuery(q)
		if err != nil {
			return nil, err
		}
		m.Logger().Debug("using query: ", query)

		params := map[string]string{
			"q":       query,
			"size":    "10000",
			"_source": "prometheus.*",
		}
		_, resp, err := client.SearchURI("metricbeat-*", "_doc", params)
		if err != nil {
			return nil, err
		}

		m.Logger().Debugf("Query took %d ms", resp.Took)

		for _, jsonDoc := range resp.Hits.Hits {
			var doc PromDoc
			if err := json.Unmarshal(jsonDoc, &doc); err != nil {
				return nil, err
			}

			tsid := doc.tsid()
			ts, ok := timeseries[tsid]
			if !ok {
				ts = &prompb.TimeSeries{
					Labels: doc.Labels(metricName),
				}
				timeseries[tsid] = ts
			}

			ts.Samples = append(ts.Samples, prompb.Sample{
				// convert ts to miliseconds
				Timestamp: doc.Source.Timestamp.UTC().UnixNano() / 1e6,
				Value:     doc.Source.Prometheus.Metrics[metricName],
			})
		}
	}

	resp := prompb.ReadResponse{
		Results: []*prompb.QueryResult{
			{Timeseries: make([]*prompb.TimeSeries, 0, len(timeseries))},
		},
	}
	for _, ts := range timeseries {
		m.Logger().Debug(ts)
		m.Logger().Debugf("%d samples", len(ts.Samples))
		resp.Results[0].Timeseries = append(resp.Results[0].Timeseries, ts)
	}
	return &resp, nil
}

type PromDoc struct {
	Source struct {
		Timestamp  time.Time `json:"@timestamp"`
		Prometheus struct {
			Metrics map[string]float64 `json:"metrics"`
			Labels  map[string]string  `json:"labels"`
		} `json:"prometheus"`
	} `json:"_source"`
}

func (p *PromDoc) tsid() string {
	bytes, err := json.Marshal(p.Source.Prometheus.Labels)
	if err != nil {
		return fmt.Sprintf("Not valid json: %v", err)
	}
	return string(bytes)
}

func (p *PromDoc) Labels(metricName string) []prompb.Label {
	labels := make([]prompb.Label, 0, len(p.Source.Prometheus.Labels))
	for k, v := range p.Source.Prometheus.Labels {
		labels = append(labels, prompb.Label{
			Name:  k,
			Value: v,
		})
	}

	labels = append(labels, prompb.Label{
		Name:  model.MetricNameLabel,
		Value: metricName,
	})

	return labels
}

func (m *MetricSet) buildQuery(q *prompb.Query) (string, error) {
	matchers := make([]string, 0, len(q.Matchers)+1)

	for _, m := range q.Matchers {
		if m.Name == model.MetricNameLabel {
			switch m.Type {
			case prompb.LabelMatcher_EQ:
				matchers = append(matchers, fmt.Sprintf("_exists_:prometheus.metrics.%s", m.Value))
			/* TODO
			case prompb.LabelMatcher_RE:
				from = fmt.Sprintf("FROM %q./^%s$/", c.retentionPolicy, escapeSlashes(m.Value))
			*/
			default:
				// TODO: Figure out how to support these efficiently.
				return "", errors.New("regex, non-equal or regex-non-equal matchers are not supported on the metric name yet")
			}
			continue
		}

		switch m.Type {
		case prompb.LabelMatcher_EQ:
			// TODO proper escaping is needed, check DSL docs
			matchers = append(matchers, fmt.Sprintf("prometheus.labels.%s='%s'", m.Name, escapeSingleQuotes(m.Value)))
		/* TODO
		case prompb.LabelMatcher_NEQ:
			matchers = append(matchers, fmt.Sprintf("%q != '%s'", m.Name, escapeSingleQuotes(m.Value)))
		case prompb.LabelMatcher_RE:
			matchers = append(matchers, fmt.Sprintf("%q =~ /^%s$/", m.Name, escapeSlashes(m.Value)))
		case prompb.LabelMatcher_NRE:
			matchers = append(matchers, fmt.Sprintf("%q !~ /^%s$/", m.Name, escapeSlashes(m.Value)))
		*/
		default:
			return "", errors.Errorf("unknown match type %v", m.Type)
		}
	}

	matchers = append(matchers, fmt.Sprintf("@timestamp:[%s TO %s]", formatTime(q.StartTimestampMs), formatTime(q.EndTimestampMs)))

	return strings.Join(matchers, " AND "), nil
}

func escapeSingleQuotes(str string) string {
	return strings.Replace(str, `'`, `\'`, -1)
}

func formatTime(timestamp int64) string {
	t := time.Unix(0, int64(time.Millisecond)*timestamp).UTC()
	return t.Format(time.RFC3339)
}
