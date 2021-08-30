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

package httpcommon

import (
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
)

// HTTPClientProxySettings provides common HTTP proxy setup support.
//
// Proxy usage will be disabled in general if Disable is set.
// If URL is not set, the proxy configuration will default
// to HTTP_PROXY, HTTPS_PROXY, and NO_PROXY.
//
// The default (and zero) value of HTTPClientProxySettings has Proxy support
// enabled, and will select the proxy per URL based on the environment variables.
type HTTPClientProxySettings struct {
	// Proxy URL to use for http connections. If the proxy url is configured,
	// it is used for all connection attempts. All proxy related environment
	// variables are ignored.
	URL *ProxyURI `config:"proxy_url" yaml:"proxy_url,omitempty"`

	// Headers configures additonal headers that are send to the proxy
	// during CONNECT requests.
	Headers ProxyHeaders `config:"proxy_headers" yaml:"proxy_headers,omitempty"`

	// Disable HTTP proxy support. Configured URLs and environment variables
	// are ignored.
	Disable bool `config:"proxy_disable" yaml:"proxy_disable,omitempty"`
}

// NewHTTPClientProxySettings creates a new proxy settings based on provided proxy information.
func NewHTTPClientProxySettings(url string, headers map[string]string, disable bool) (*HTTPClientProxySettings, error) {
	proxyURI, err := NewProxyURIFromString(url)
	if err != nil {
		return nil, err
	}

	return &HTTPClientProxySettings{
		URL:     proxyURI,
		Headers: ProxyHeaders(headers),
		Disable: disable,
	}, nil
}

// DefaultHTTPClientProxySettings returns the default HTTP proxy setting.
func DefaultHTTPClientProxySettings() HTTPClientProxySettings {
	return HTTPClientProxySettings{
		Headers: make(ProxyHeaders),
	}
}

// Unpack sets the proxy settings from a config object.
// Note: Unpack is automatically used by the configuration system if `cfg.Unpack(&x)` is and X contains
// a field of type HTTPClientProxySettings.
func (settings *HTTPClientProxySettings) Unpack(cfg *common.Config) error {
	tmp := struct {
		URL     string            `config:"proxy_url"`
		Disable bool              `config:"proxy_disable"`
		Headers map[string]string `config:"proxy_headers"`
	}{}

	if err := cfg.Unpack(&tmp); err != nil {
		return err
	}

	s, err := NewHTTPClientProxySettings(tmp.URL, tmp.Headers, tmp.Disable)
	if err != nil {
		return err
	}

	*settings = *s
	return nil
}

// ProxyFunc creates a function that can be used with http.Transport in order to
// configure the HTTP proxy functionality.
func (settings *HTTPClientProxySettings) ProxyFunc() func(*http.Request) (*url.URL, error) {
	if settings.Disable {
		return nil
	}

	if settings.URL == nil {
		return http.ProxyFromEnvironment
	}

	return http.ProxyURL(settings.URL.URI())
}
