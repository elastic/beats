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

package transport

import (
	"net"
	"net/url"

	"golang.org/x/net/proxy"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// ProxyConfig holds the configuration information required to proxy
// connections through a SOCKS5 proxy server.
type ProxyConfig struct {
	// URL of the SOCKS proxy. Scheme must be socks5. Username and password can be
	// embedded in the URL.
	URL string `config:"proxy_url"`

	// Resolve names locally instead of on the SOCKS server.
	LocalResolve bool `config:"proxy_use_local_resolver"`
}

func (c *ProxyConfig) Validate() error {
	if c.URL == "" {
		return nil
	}

	url, err := url.Parse(c.URL)
	if err != nil {
		return err
	}
	if _, err := proxy.FromURL(url, nil); err != nil {
		return err
	}

	return nil
}

func ProxyDialer(log *logp.Logger, config *ProxyConfig, forward Dialer) (Dialer, error) {
	if config == nil || config.URL == "" {
		return forward, nil
	}

	url, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}

	if _, err := proxy.FromURL(url, nil); err != nil {
		return nil, err
	}

	log.Debugf("breaking down proxy URL. Scheme: '%s', host[:port]: '%s', path: '%s'", url.Scheme, url.Host, url.Path)
	log.Infof("proxy host: '%s'", url.Host)
	return DialerFunc(func(network, address string) (net.Conn, error) {
		var err error
		var addresses []string

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		if config.LocalResolve {
			addresses, err = net.LookupHost(host)
			if err != nil {
				log.Warnf(`DNS lookup failure "%s": %+v`, host, err)
				return nil, err
			}
		} else {
			// Do not resolve the address locally. It will be resolved on the
			// SOCKS server. The beat will have no control over the randomization
			// of the IP used when multiple IPs are returned by DNS.
			addresses = []string{host}
		}

		dialer, err := proxy.FromURL(url, forward)
		if err != nil {
			return nil, err
		}
		return DialWith(dialer, network, host, addresses, port)
	}), nil
}
