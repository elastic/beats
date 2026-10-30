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

package tcp

import (
	"fmt"
	"net/url"

	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
)

// destinations returns the hosts of the endpoints and of the SOCKS5 proxy of
// the monitor, parsed as create does.
func destinations(cfg *conf.C) ([]string, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	hosts := make([]string, 0, len(config.Hosts)+1)
	for i, h := range config.Hosts {
		// The default scheme doesn't change the host name. Values are left
		// out of errors: they can come from the keystore.
		eps, err := makeEndpoints([]string{h}, config.Ports, "tcp")
		if err != nil || len(eps) != 1 {
			return nil, plugin.UnknownDestinations(fmt.Sprintf("cannot parse host %d", i))
		}
		hosts = append(hosts, eps[0].Hostname)
	}

	if config.Socks5.URL != "" {
		u, err := url.Parse(config.Socks5.URL)
		if err != nil || u.Hostname() == "" {
			return nil, plugin.UnknownDestinations("cannot parse proxy_url")
		}
		hosts = append(hosts, u.Hostname())
	}
	return hosts, nil
}
