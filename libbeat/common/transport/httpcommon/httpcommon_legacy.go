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

// +build !go1.15

package httpcommon

import (
	"net/http"

	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

func (settings *HTTPTransportSettings) httpRoundTripper(
	tls *tlscommon.TLSConfig,
	dialer, tlsDialer transport.Dialer,
	opts ...TransportOption,
) (*http.Transport, error) {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = nil
	t.Dial = dialer.Dial
	t.DialTLS = tlsDialer.Dial
	t.TLSClientConfig = tls.ToConfig()
	t.ForceAttemptHTTP2 = false
	t.Proxy = settings.Proxy.ProxyFunc()
	t.ProxyConnectHeader = settings.Proxy.Headers

	//  reset some internal timeouts to not change old Beats defaults
	t.TLSHandshakeTimeout = 0
	t.ExpectContinueTimeout = 0

	for _, opt := range opts {
		if transportOpt, ok := opt.(httpTransportOption); ok {
			transportOpt.applyTransport(settings, t)
		}
	}

	return t, nil
}
