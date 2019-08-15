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
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	"github.com/elastic/beats/libbeat/publisher/processing"
	"github.com/elastic/beats/libbeat/publisher/queue"
	"github.com/elastic/beats/libbeat/publisher/queue/memqueue"
)

type reporter struct {
	done   *stopper
	logger *logp.Logger

	checkRetry time.Duration

	// event metadata
	beatMeta common.MapStr
	tags     []string

	// pipeline
	pipeline *pipeline.Pipeline
	client   beat.Client

	out []outputs.NetworkClient
}

const selector = "monitoring"

var debugf = logp.MakeDebug(selector)

var errNoMonitoring = errors.New("xpack monitoring not available")

// default monitoring api parameters
var defaultParams = map[string]string{
	"system_id":          "beats",
	"system_api_version": "7",
}

func init() {
	report.RegisterReporterFactory("elasticsearch", makeReporter)
}

func defaultConfig(settings report.Settings) config {
	c := config{
		Hosts:            nil,
		Protocol:         "http",
		Params:           nil,
		Headers:          nil,
		Username:         "beats_system",
		Password:         "",
		ProxyURL:         "",
		CompressionLevel: 0,
		TLS:              nil,
		MaxRetries:       3,
		Timeout:          60 * time.Second,
		MetricsPeriod:    10 * time.Second,
		StatePeriod:      1 * time.Minute,
		BulkMaxSize:      50,
		BufferSize:       50,
		Tags:             nil,
		Backoff: backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		Format:      report.FormatXPackMonitoringBulk,
		ClusterUUID: settings.ClusterUUID,
	}

	if settings.DefaultUsername != "" {
		c.Username = settings.DefaultUsername
	}

	if settings.Format != report.FormatUnknown {
		c.Format = settings.Format
	}

	return c
}

func makeReporter(beat beat.Info, settings report.Settings, cfg *common.Config) (report.Reporter, error) {
	log := logp.L().Named(selector)
	config := defaultConfig(settings)
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
		log.Infof("Using proxy URL: %s", proxyURL)
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

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.New("empty hosts list")
	}

	var clients []outputs.NetworkClient
	for _, host := range hosts {
		client, err := makeClient(host, params, proxyURL, tlsConfig, &config)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}

	queueFactory := func(e queue.Eventer) (queue.Queue, error) {
		return memqueue.NewBroker(log,
			memqueue.Settings{
				Eventer: e,
				Events:  20,
			}), nil
	}

	monitoring := monitoring.Default.GetRegistry("xpack.monitoring")

	outClient := outputs.NewFailoverClient(clients)
	outClient = outputs.WithBackoff(outClient, config.Backoff.Init, config.Backoff.Max)

	processing, err := processing.MakeDefaultSupport(true)(beat, log, common.NewConfig())
	if err != nil {
		return nil, err
	}

	pipeline, err := pipeline.New(
		beat,
		pipeline.Monitors{
			Metrics: monitoring,
			Logger:  log,
		},
		queueFactory,
		outputs.Group{
			Clients:   []outputs.Client{outClient},
			BatchSize: windowSize,
			Retry:     0, // no retry. Drop event on error.
		},
		pipeline.Settings{
			WaitClose:     0,
			WaitCloseMode: pipeline.NoWaitOnClose,
			Processors:    processing,
		})
	if err != nil {
		return nil, err
	}

	pipeConn, err := pipeline.Connect()
	if err != nil {
		pipeline.Close()
		return nil, err
	}

	r := &reporter{
		logger:     log,
		done:       newStopper(),
		beatMeta:   makeMeta(beat),
		tags:       config.Tags,
		checkRetry: checkRetry,
		pipeline:   pipeline,
		client:     pipeConn,
		out:        clients,
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

	log := r.logger

	logged := false

	for {
		// Select one configured endpoint by random and check if xpack is available
		client := r.out[rand.Intn(len(r.out))]
		err := client.Connect()
		if err == nil {
			closing(log, client)
			break
		} else {
			if !logged {
				log.Info("Failed to connect to Elastic X-Pack Monitoring. Either Elasticsearch X-Pack monitoring is not enabled or Elasticsearch is not available. Will keep retrying.")
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

	log.Info("Successfully connected to X-Pack Monitoring endpoint.")

	// Start collector and send loop if monitoring endpoint has been found.
	go r.snapshotLoop("state", "state", c.StatePeriod, c.ClusterUUID)
	// For backward compatibility stats is named to metrics.
	go r.snapshotLoop("stats", "metrics", c.MetricsPeriod, c.ClusterUUID)
}

func (r *reporter) snapshotLoop(namespace, prefix string, period time.Duration, clusterUUID string) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	log := r.logger

	log.Infof("Start monitoring %s metrics snapshot loop with period %s.", namespace, period)
	defer log.Infof("Stop monitoring %s metrics snapshot loop.", namespace)

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
			"beat": r.beatMeta,
			prefix: snapshot,
		}
		if len(r.tags) > 0 {
			fields["tags"] = r.tags
		}

		meta := common.MapStr{
			"type":        "beats_" + namespace,
			"interval_ms": int64(period / time.Millisecond),
			// Converting to seconds as interval only accepts `s` as unit
			"params": map[string]string{"interval": strconv.Itoa(int(period/time.Second)) + "s"},
		}

		if clusterUUID == "" {
			clusterUUID = getClusterUUID()
		}
		if clusterUUID != "" {
			meta.Put("cluster_uuid", clusterUUID)
		}

		r.client.Publish(beat.Event{
			Timestamp: ts,
			Fields:    fields,
			Meta:      meta,
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

	if config.Format != report.FormatXPackMonitoringBulk && config.Format != report.FormatBulk {
		return nil, fmt.Errorf("unknown reporting format: %v", config.Format)
	}

	return newPublishClient(esClient, params, config.Format)
}

func closing(log *logp.Logger, c io.Closer) {
	if err := c.Close(); err != nil {
		log.Warnf("Closed failed with: %v", err)
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
		"uuid":    beat.ID,
	}
}

func getClusterUUID() string {
	stateRegistry := monitoring.GetNamespace("state").GetRegistry()
	outputsRegistry := stateRegistry.GetRegistry("outputs")
	if outputsRegistry == nil {
		return ""
	}

	elasticsearchRegistry := outputsRegistry.GetRegistry("elasticsearch")
	if elasticsearchRegistry == nil {
		return ""
	}

	snapshot := monitoring.CollectFlatSnapshot(elasticsearchRegistry, monitoring.Full, false)
	return snapshot.Strings["cluster_uuid"]
}
