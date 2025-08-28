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
	log := beat.Logger.Named("logstash")
	return MakeLogstashClients(beat.Version, log, observer, cfg, beat.IndexPrefix)
}

func MakeLogstashClients(
	beatVersion string,
	logger *logp.Logger,
	observer outputs.Observer,
	rawCfg *conf.C,
	beatIndexPrefix string,
) (outputs.Group, error) {
	config, err := readConfig(rawCfg, beatIndexPrefix)
	if err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(rawCfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tls, err := tlscommon.LoadTLSConfig(config.TLS, logger)
	if err != nil {
		return outputs.Group{}, err
	}

	transp := transport.Config{
		Timeout: config.Timeout,
		Proxy:   &config.Proxy,
		TLS:     tls,
		Stats:   observer,
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		var client outputs.NetworkClient

		conn, err := transport.NewClient(transp, "tcp", host, defaultPort)
		if err != nil {
			return outputs.Fail(err)
		}

		if config.Pipelining > 0 {
			client, err = newAsyncClient(logger, beatVersion, conn, observer, config)
		} else {
			client, err = newSyncClient(logger, beatVersion, conn, observer, config)
		}
		if err != nil {
			return outputs.Fail(err)
		}

		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}

<<<<<<< HEAD
	return outputs.SuccessNet(lsConfig.Queue, lsConfig.LoadBalance, lsConfig.BulkMaxSize, lsConfig.MaxRetries, nil, clients)
=======
	return outputs.SuccessNet(config.Queue, config.LoadBalance, config.BulkMaxSize, config.MaxRetries, nil, logger, clients)
>>>>>>> aed42964b (Export the Logstash client to enable reuse across packages. (#46040))
}
