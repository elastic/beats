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

package logstash

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	minWindowSize             int = 1
	defaultStartMaxWindowSize int = 10
	defaultPort                   = 5044
)

func init() {
	outputs.RegisterType("logstash", makeLogstash)
}

func makeLogstash(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *conf.C,
) (outputs.Group, error) {
	lsConfig, err := readConfig(cfg, beat)
	if err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tls, err := tlscommon.LoadTLSConfig(lsConfig.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	transp := transport.Config{
		Timeout: lsConfig.Timeout,
		Proxy:   &lsConfig.Proxy,
		TLS:     tls,
		Stats:   observer,
	}

	encoderFactory := newEventEncoderFactory(
		logp.NewLogger("logstash"),
		beat,
		lsConfig.EscapeHTML,
		lsConfig.Index)

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		var client outputs.NetworkClient

		conn, err := transport.NewClient(transp, "tcp", host, defaultPort)
		if err != nil {
			return outputs.Fail(err)
		}

		if lsConfig.Pipelining > 0 {
			client, err = newAsyncClient(beat, conn, observer, lsConfig)
		} else {
			client, err = newSyncClient(beat, conn, observer, lsConfig)
		}
		if err != nil {
			return outputs.Fail(err)
		}

		client = outputs.WithBackoff(client, lsConfig.Backoff.Init, lsConfig.Backoff.Max)
		clients[i] = client
	}

	return outputs.SuccessNet(lsConfig.Queue, lsConfig.LoadBalance, lsConfig.BulkMaxSize, lsConfig.MaxRetries, encoderFactory, clients)
}
