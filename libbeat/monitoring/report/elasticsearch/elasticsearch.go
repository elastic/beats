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
	"strconv"
	"time"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/menderesk/beats/v7/libbeat/esleg/eslegclient"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/libbeat/monitoring"
	"github.com/menderesk/beats/v7/libbeat/monitoring/report"
	"github.com/menderesk/beats/v7/libbeat/outputs"
	"github.com/menderesk/beats/v7/libbeat/publisher/pipeline"
	"github.com/menderesk/beats/v7/libbeat/publisher/processing"
	"github.com/menderesk/beats/v7/libbeat/publisher/queue"
	"github.com/menderesk/beats/v7/libbeat/publisher/queue/memqueue"
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

const logSelector = "monitoring"

var errNoMonitoring = errors.New("xpack monitoring not available")

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
		APIKey:           "",
		ProxyURL:         "",
		CompressionLevel: 0,
		MaxRetries:       3,
		MetricsPeriod:    10 * time.Second,
		StatePeriod:      1 * time.Minute,
		BulkMaxSize:      50,
		BufferSize:       50,
		Tags:             nil,
		Backoff: backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		ClusterUUID: settings.ClusterUUID,
		Transport:   httpcommon.DefaultHTTPTransportSettings(),
	}

	if settings.DefaultUsername != "" {
		c.Username = settings.DefaultUsername
	}

	return c
}

func makeReporter(beat beat.Info, settings report.Settings, cfg *common.Config) (report.Reporter, error) {
	log := logp.NewLogger(logSelector)
	config := defaultConfig(settings)
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	// Unset username which is set by default, even if no password is set
	if config.APIKey != "" {
		config.Username = ""
		config.Password = ""
	}

	// check endpoint availability on startup only every 30 seconds
	checkRetry := 30 * time.Second
	windowSize := config.BulkMaxSize - 1
	if windowSize <= 0 {
		windowSize = 1
	}

	params := makeClientParams(config)

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.New("empty hosts list")
	}

	var clients []outputs.NetworkClient
	for _, host := range hosts {
		client, err := makeClient(host, params, &config, beat.Beat)
		if err != nil {
			return nil, err
		}
		clients = append(clients, client)
	}

	queueFactory := func(ackListener queue.ACKListener) (queue.Queue, error) {
		return memqueue.NewQueue(log,
			memqueue.Settings{
				ACKListener: ackListener,
				Events:      20,
			}), nil
	}

	monitoring := monitoring.Default.GetRegistry("monitoring")

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
	r.logger.Debug("Start monitoring endpoint init loop.")
	defer r.logger.Debug("Finish monitoring endpoint init loop.")

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
				log.Info("Failed to connect to Elastic X-Pack Monitoring. Either Elasticsearch X-Pack monitoring is not enabled or Elasticsearch is not available. Will keep retrying. Error: ", err)
				logged = true
			}
			r.logger.Debugf("Monitoring could not connect to Elasticsearch, failed with %+v", err)
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
			log.Debug("Empty snapshot.")
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

func makeClient(host string, params map[string]string, config *config, beatname string) (outputs.NetworkClient, error) {
	url, err := common.MakeURL(config.Protocol, "", host, 9200)
	if err != nil {
		return nil, err
	}

	esClient, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              url,
		Beatname:         beatname,
		Username:         config.Username,
		Password:         config.Password,
		APIKey:           config.APIKey,
		Parameters:       params,
		Headers:          config.Headers,
		CompressionLevel: config.CompressionLevel,
		Transport:        config.Transport,
	})
	if err != nil {
		return nil, err
	}

	return newPublishClient(esClient, params)
}

func closing(log *logp.Logger, c io.Closer) {
	if err := c.Close(); err != nil {
		log.Warnf("Closed failed with: %v", err)
	}
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

func makeClientParams(config config) map[string]string {
	params := map[string]string{}

	for k, v := range config.Params {
		params[k] = v
	}

	return params
}
