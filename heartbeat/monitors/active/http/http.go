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
	"bytes"
	"net/http"
	"net/url"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"

	"github.com/elastic/beats/heartbeat/monitors"
)

func init() {
	monitors.RegisterActive("http", create)
}

var debugf = logp.MakeDebug("http")

func create(
	name string,
	cfg *common.Config,
) (jobs []monitors.Job, endpoints int, err error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, 0, err
	}

	var body []byte
	var enc contentEncoder

	if config.Check.Request.SendBody != "" {
		var err error
		compression := config.Check.Request.Compression
		enc, err = getContentEncoder(compression.Type, compression.Level)
		if err != nil {
			return nil, 0, err
		}

		buf := bytes.NewBuffer(nil)
		err = enc.Encode(buf, bytes.NewBufferString(config.Check.Request.SendBody))
		if err != nil {
			return nil, 0, err
		}

		body = buf.Bytes()
	}

	validator, err := makeValidateResponse(&config.Check.Response)
	if err != nil {
		return nil, 0, err
	}

	jobs = make([]monitors.Job, len(config.URLs))

	if config.ProxyURL != "" {
		transport, err := newRoundTripper(&config, tls)
		if err != nil {
			return nil, 0, err
		}

		for i, url := range config.URLs {
			jobs[i], err = newHTTPMonitorHostJob(url, &config, transport, enc, body, validator)
			if err != nil {
				return nil, 0, err
			}
		}
	} else {
		for i, url := range config.URLs {
			jobs[i], err = newHTTPMonitorIPsJob(&config, url, tls, enc, body, validator)
			if err != nil {
				return nil, 0, err
			}
		}
	}

	return jobs, len(config.URLs), nil
}

func newRoundTripper(config *Config, tls *transport.TLSConfig) (*http.Transport, error) {
	var proxy func(*http.Request) (*url.URL, error)
	if config.ProxyURL != "" {
		url, err := url.Parse(config.ProxyURL)
		if err != nil {
			return nil, err
		}
		proxy = http.ProxyURL(url)
	}

	dialer := transport.NetDialer(config.Timeout)
	tlsDialer, err := transport.TLSDialer(dialer, tls, config.Timeout)
	if err != nil {
		return nil, err
	}

	return &http.Transport{
		Proxy:             proxy,
		Dial:              dialer.Dial,
		DialTLS:           tlsDialer.Dial,
		DisableKeepAlives: true,
	}, nil
}
