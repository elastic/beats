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
	"errors"
	"io"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"strconv"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
	"github.com/elastic/beats/libbeat/monitoring/report"
	"github.com/elastic/beats/libbeat/outputs"
	esout "github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/outil"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/elastic/beats/libbeat/publisher/pipeline"
	"github.com/elastic/beats/libbeat/publisher/queue"
	"github.com/elastic/beats/libbeat/publisher/queue/memqueue"
)

type reporter struct {
	done *stopper

	checkRetry time.Duration

	// event metadata
	beatMeta common.MapStr
	tags     []string

	// pipeline
	pipeline *pipeline.Pipeline
	client   beat.Client
	out      outputs.Group
}

var debugf = logp.MakeDebug("monitoring")

var errNoMonitoring = errors.New("xpack monitoring not available")

// default monitoring api parameters
var defaultParams = map[string]string{
	"system_id":          "beats",
	"system_api_version": "6",
}

func init() {
	report.RegisterReporterFactory("elasticsearch", makeReporter)
}

func makeReporter(beat beat.Info, cfg *common.Config) (report.Reporter, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	// check endpoint availability on startup only every 30 seconds
	checkRetry := 30 * time.Second
	windowSize := config.BulkMaxSize - 1
	if windowSize <= 0 {
		windowSize = 1
	}

	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		logp.Info("Using proxy URL: %s", proxyURL)
	}
	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	params := map[string]string{}
	for k, v := range defaultParams {
		params[k] = v
	}
	for k, v := range config.Params {
		params[k] = v
	}

	out := outputs.Group{
		Clients:   nil,
		BatchSize: windowSize,
		Retry:     0, // no retry. on error drop events
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		client, err := makeClient(host, params, proxyURL, tlsConfig, &config)
		if err != nil {
			return nil, err
		}
		out.Clients = append(out.Clients, client)
	}

	queueFactory := func(e queue.Eventer) (queue.Queue, error) {
		return memqueue.NewBroker(memqueue.Settings{
			Eventer: e,
			Events:  20,
		}), nil
	}

	monitoring := monitoring.Default.GetRegistry("xpack.monitoring")

	pipeline, err := pipeline.New(
		beat,
		monitoring,
		queueFactory, out, pipeline.Settings{
			WaitClose:     0,
			WaitCloseMode: pipeline.NoWaitOnClose,
		})
	if err != nil {
		return nil, err
	}

	client, err := pipeline.Connect()
	if err != nil {
		pipeline.Close()
		return nil, err
	}

	r := &reporter{
		done:       newStopper(),
		beatMeta:   makeMeta(beat),
		tags:       config.Tags,
		checkRetry: checkRetry,
		pipeline:   pipeline,
		client:     client,
		out:        out,
	}
	go r.initLoop(config)
	return r, nil
}

func (r *reporter) Stop() {
	r.done.Stop()
	r.client.Close()
	r.pipeline.Close()
}

func (r *reporter) initLoop(c config) {
	debugf("Start monitoring endpoint init loop.")
	defer debugf("Finish monitoring endpoint init loop.")

	logged := false

	for {
		// Select one configured endpoint by random and check if xpack is available
		client := r.out.Clients[rand.Intn(len(r.out.Clients))].(outputs.NetworkClient)
		err := client.Connect()
		if err == nil {
			closing(client)
			break
		} else {
			if !logged {
				logp.Info("Failed to connect to Elastic X-Pack Monitoring. Either Elasticsearch X-Pack monitoring is not enabled or Elasticsearch is not available. Will keep retrying.")
				logged = true
			}
			debugf("Monitoring could not connect to elasticsearch, failed with %v", err)
		}

		select {
		case <-r.done.C():
			return
		case <-time.After(r.checkRetry):
		}
	}

	logp.Info("Successfully connected to X-Pack Monitoring endpoint.")

	// Start collector and send loop if monitoring endpoint has been found.
	go r.snapshotLoop("state", c.StatePeriod)
	go r.snapshotLoop("stats", c.MetricsPeriod)
}

func (r *reporter) snapshotLoop(namespace string, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	logp.Info("Start monitoring %s metrics snapshot loop with period %s.", namespace, period)
	defer logp.Info("Stop monitoring %s metrics snapshot loop.", namespace)

	for {
		var ts time.Time

		select {
		case <-r.done.C():
			return
		case ts = <-ticker.C:
		}

		snapshot := makeSnapshot(monitoring.GetNamespace(namespace).GetRegistry())
		if snapshot == nil {
			debugf("Empty snapshot.")
			continue
		}

		fields := common.MapStr{
			"beat":    r.beatMeta,
			namespace: snapshot,
		}
		if len(r.tags) > 0 {
			fields["tags"] = r.tags
		}
		r.client.Publish(beat.Event{
			Timestamp: ts,
			Fields:    fields,
			Meta: common.MapStr{
				"type":        "beats_" + namespace,
				"interval_ms": int64(period / time.Millisecond),
				// Converting to seconds as interval only accepts `s` as unit
				"params": map[string]string{"interval": strconv.Itoa(int(period/time.Second)) + "s"},
			},
		})
	}
}

func makeClient(
	host string,
	params map[string]string,
	proxyURL *url.URL,
	tlsConfig *transport.TLSConfig,
	config *config,
) (outputs.NetworkClient, error) {
	url, err := common.MakeURL(config.Protocol, "", host, 9200)
	if err != nil {
		return nil, err
	}

	esClient, err := esout.NewClient(esout.ClientSettings{
		URL:              url,
		Proxy:            proxyURL,
		TLS:              tlsConfig,
		Username:         config.Username,
		Password:         config.Password,
		Parameters:       params,
		Headers:          config.Headers,
		Index:            outil.MakeSelector(outil.ConstSelectorExpr("_xpack")),
		Pipeline:         nil,
		Timeout:          config.Timeout,
		CompressionLevel: config.CompressionLevel,
	}, nil)
	if err != nil {
		return nil, err
	}

	return newPublishClient(esClient, params), nil
}

func closing(c io.Closer) {
	if err := c.Close(); err != nil {
		logp.Warn("Closed failed with: %v", err)
	}
}

// TODO: make this reusable. Same definition in elasticsearch monitoring module
func parseProxyURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, nil
	}

	url, err := url.Parse(raw)
	if err == nil && strings.HasPrefix(url.Scheme, "http") {
		return url, err
	}

	// Proxy was bogus. Try prepending "http://" to it and
	// see if that parses correctly.
	return url.Parse("http://" + raw)
}

func makeMeta(beat beat.Info) common.MapStr {
	return common.MapStr{
		"type":    beat.Beat,
		"version": beat.Version,
		"name":    beat.Name,
		"host":    beat.Hostname,
		"uuid":    beat.UUID,
	}
}
