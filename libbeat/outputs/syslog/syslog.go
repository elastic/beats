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

package syslog

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

func init() {
	outputs.RegisterType("syslog", makeSyslog)
}

func makeSyslog(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	// Not Support Proxy, if Proxy is not nil, Network is udp will error.
	transp := &transport.Config{
		Timeout: config.Timeout,
		Proxy:   nil,
		TLS:     tls,
		Stats:   observer,
	}

	clients := make([]outputs.NetworkClient, len(hosts))

	for i, host := range hosts {
		var client outputs.NetworkClient

		conn, err := transport.NewClient(transp, config.Network, host, config.Port)
		if err != nil {
			return outputs.Fail(err)
		}

		client = newClient(conn, observer, config.SyslogProgram, config.SyslogPriority, config.SyslogSeverity, config.Timeout)

		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)

		clients[i] = client
	}
	return outputs.SuccessNet(false, -1, config.MaxRetries, clients)
}
