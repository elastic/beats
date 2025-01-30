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

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers/wraputil"
	"github.com/elastic/beats/v7/libbeat/version"
	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/elastic/elastic-agent-libs/useragent"
)

func init() {
	plugin.Register("http", create, "synthetics/http")
}

var userAgent = useragent.UserAgent("Heartbeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

// Create makes a new HTTP monitor
func create(
	name string,
	cfg *conf.C,
) (p plugin.Plugin, err error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return plugin.Plugin{}, err
	}

	var body []byte
	var enc contentEncoder

	if config.Check.Request.SendBody != "" {
		var err error
		compression := config.Check.Request.Compression
		enc, err = getContentEncoder(compression.Type, compression.Level)
		if err != nil {
			return plugin.Plugin{}, err
		}

		buf := bytes.NewBuffer(nil)
		err = enc.Encode(buf, bytes.NewBufferString(config.Check.Request.SendBody))
		if err != nil {
			return plugin.Plugin{}, err
		}

		body = buf.Bytes()
	}

	validator, err := makeValidateResponse(&config.Check.Response)
	if err != nil {
		return plugin.Plugin{}, err
	}

	// Determine whether we're using a proxy or not and then use that to figure out how to
	// run the job
	var makeJob func(string) (jobs.Job, error)
	// In the event that a ProxyURL is present, or redirect support is enabled
	// we execute DNS resolution requests inline with the request, not running them as a separate job, and not returning
	// separate DNS rtt data.
	if (config.Transport.Proxy.URL != nil && !config.Transport.Proxy.Disable) || config.MaxRedirects > 0 {
		transport, err := newRoundTripper(&config)
		if err != nil {
			return plugin.Plugin{}, err
		}

		makeJob = func(urlStr string) (jobs.Job, error) {
			return newHTTPMonitorHostJob(urlStr, &config, transport, enc, body, validator)
		}
	} else {
		// preload TLS configuration
		tls, err := tlscommon.LoadTLSConfig(config.Transport.TLS)
		if err != nil {
			return plugin.Plugin{}, err
		}
		config.Transport.TLS = nil

		makeJob = func(urlStr string) (jobs.Job, error) {
			return newHTTPMonitorIPsJob(&config, urlStr, tls, enc, body, validator)
		}
	}

	js := make([]jobs.Job, len(config.Hosts))
	for i, urlStr := range config.Hosts {
		u, err := url.Parse(urlStr)
		if err != nil {
			return plugin.Plugin{}, err
		}

		job, err := makeJob(urlStr)
		if err != nil {
			return plugin.Plugin{}, err
		}

		// Assign any execution errors to the error field and
		// assign the url field
		js[i] = wraputil.WithURLField(u, job)
	}

	return plugin.Plugin{Jobs: js, Endpoints: len(config.Hosts)}, nil
}

func newRoundTripper(config *Config) (http.RoundTripper, error) {
	return config.Transport.RoundTripper(
		httpcommon.WithAPMHTTPInstrumentation(),
		httpcommon.WithoutProxyEnvironmentVariables(),
		httpcommon.WithKeepaliveSettings{
			Disable: true,
		},
		httpcommon.WithHeaderRoundTripper(map[string]string{"User-Agent": userAgent}),
	)
}
