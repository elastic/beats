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

package http

import (
	"errors"
	"fmt"
	"net/url"

	conf "github.com/elastic/elastic-agent-libs/config"
)

// destinations returns the hosts of the URLs and of the proxy of the monitor,
// parsed as create does. Redirects are refused: they can send the request to
// any host. Kerberos is refused.
func destinations(cfg *conf.C) ([]string, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}
	if config.MaxRedirects > 0 {
		return nil, errors.New("max_redirects must be 0")
	}
	if config.Kerberos.IsEnabled() {
		return nil, errors.New("kerberos is not supported")
	}

	// Validate, run by Unpack, merged urls into hosts.
	hosts := make([]string, 0, len(config.Hosts)+1)
	for i, raw := range config.Hosts {
		// Values are left out of errors: they can come from the keystore.
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != urlSchemaHTTP && u.Scheme != urlSchemaHTTPS) || u.Hostname() == "" {
			return nil, fmt.Errorf("cannot parse URL %d", i)
		}
		hosts = append(hosts, u.Hostname())
	}

	if proxy := config.Transport.Proxy; proxy.URL != nil && !proxy.Disable {
		host := proxy.URL.URI().Hostname()
		if host == "" {
			return nil, errors.New("cannot parse proxy_url")
		}
		hosts = append(hosts, host)
	}
	return hosts, nil
}
