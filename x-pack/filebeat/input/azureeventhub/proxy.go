// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"net/http"

	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

// proxyHTTPClient returns an HTTP client configured with the given proxy
// settings. It returns nil when the settings match the default behaviour
// (use environment variables), allowing callers to fall back to their
// own defaults.
func proxyHTTPClient(proxy httpcommon.HTTPClientProxySettings) *http.Client {
	if !proxy.Disable && proxy.URL == nil {
		return nil
	}
	dt, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{
			Transport: &http.Transport{Proxy: proxy.ProxyFunc()},
		}
	}
	t := dt.Clone()
	t.Proxy = proxy.ProxyFunc()
	return &http.Client{Transport: t}
}
