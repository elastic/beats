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
	"time"

	"go.elastic.co/apm/module/apmhttp"
	"golang.org/x/net/http2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// HTTPTransportSettings provides common HTTP settings for HTTP clients.
type HTTPTransportSettings struct {
	// TLS provides ssl/tls setup settings
	TLS *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty" json:"ssl,omitempty"`

	// Timeout configures the `(http.Transport).Timeout`.
	Timeout time.Duration `config:"timeout" yaml:"timeout,omitempty" json:"timeout,omitempty"`

	Proxy HTTPClientProxySettings `config:",inline" yaml:",inline"`

	// TODO: Add more settings:
	//  - DisableKeepAlive
	//  - MaxIdleConns
	//  - IdleConnTimeout
	//  - ResponseHeaderTimeout
	//  - ConnectionTimeout (currently 'Timeout' is used for both)
}

// WithKeepaliveSettings options can be used to modify the Keepalive
type WithKeepaliveSettings struct {
	Disable             bool
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
}

var _ httpTransportOption = WithKeepaliveSettings{}

const defaultHTTPTimeout = 90 * time.Second

type (
	// TransportOption are applied to the http.RoundTripper to be build
	// from HTTPTransportSettings.
	TransportOption interface{ sealTransportOption() }

	extraSettings struct {
		logger *logp.Logger
		http2  bool
	}

	dialerOption interface {
		TransportOption
		baseDialer() transport.Dialer
	}
	dialerModOption interface {
		TransportOption
		applyDialer(*HTTPTransportSettings, transport.Dialer) transport.Dialer
	}
	httpTransportOption interface {
		TransportOption
		applyTransport(*HTTPTransportSettings, *http.Transport)
	}
	roundTripperOption interface {
		TransportOption
		applyRoundTripper(*HTTPTransportSettings, http.RoundTripper) http.RoundTripper
	}
	extraOption interface {
		TransportOption
		applyExtra(*extraSettings)
	}
)

type baseDialerFunc func() transport.Dialer

var _ dialerOption = baseDialerFunc(nil)

func (baseDialerFunc) sealTransportOption() {}
func (fn baseDialerFunc) baseDialer() transport.Dialer {
	return fn()
}

type dialerOptFunc func(transport.Dialer) transport.Dialer

var _ dialerModOption = dialerOptFunc(nil)

func (dialerOptFunc) sealTransportOption() {}
func (fn dialerOptFunc) applyDialer(_ *HTTPTransportSettings, d transport.Dialer) transport.Dialer {
	return fn(d)

}

type transportOptFunc func(*HTTPTransportSettings, *http.Transport)

var _ httpTransportOption = transportOptFunc(nil)

func (transportOptFunc) sealTransportOption() {}
func (fn transportOptFunc) applyTransport(s *HTTPTransportSettings, t *http.Transport) {
	fn(s, t)
}

type rtOptFunc func(http.RoundTripper) http.RoundTripper

var _ roundTripperOption = rtOptFunc(nil)

func (rtOptFunc) sealTransportOption() {}
func (fn rtOptFunc) applyRoundTripper(_ *HTTPTransportSettings, rt http.RoundTripper) http.RoundTripper {
	return fn(rt)
}

type extraOptionFunc func(*extraSettings)

func (extraOptionFunc) sealTransportOption()           {}
func (fn extraOptionFunc) applyExtra(s *extraSettings) { fn(s) }

type headerRoundTripper struct {
	headers map[string]string
	rt      http.RoundTripper
}

func (rt *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range rt.headers {
		if len(req.Header.Get(k)) == 0 {
			req.Header.Set(k, v)
		}
	}
	return rt.rt.RoundTrip(req)
}

// DefaultHTTPTransportSettings returns the default HTTP transport setting.
func DefaultHTTPTransportSettings() HTTPTransportSettings {
	return HTTPTransportSettings{
		Proxy:   DefaultHTTPClientProxySettings(),
		Timeout: defaultHTTPTimeout,
	}
}

// Unpack reads a config object into the settings.
func (settings *HTTPTransportSettings) Unpack(cfg *common.Config) error {
	tmp := struct {
		TLS     *tlscommon.Config `config:"ssl"`
		Timeout time.Duration     `config:"timeout"`
	}{Timeout: settings.Timeout}

	if err := cfg.Unpack(&tmp); err != nil {
		return err
	}

	var proxy HTTPClientProxySettings
	if err := cfg.Unpack(&proxy); err != nil {
		return err
	}

	_, err := tlscommon.LoadTLSConfig(tmp.TLS)
	if err != nil {
		return err
	}

	*settings = HTTPTransportSettings{
		TLS:     tmp.TLS,
		Timeout: tmp.Timeout,
		Proxy:   proxy,
	}
	return nil
}

// RoundTripper creates a http.RoundTripper for use with http.Client.
//
// The dialers will registers with stats if given. Stats is used to collect metrics for io errors,
// bytes in, and bytes out.
func (settings *HTTPTransportSettings) RoundTripper(opts ...TransportOption) (http.RoundTripper, error) {
	var dialer transport.Dialer

	var extra extraSettings
	for _, opt := range opts {
		if opt, ok := opt.(extraOption); ok {
			opt.applyExtra(&extra)
		}
	}

	for _, opt := range opts {
		if dialOpt, ok := opt.(dialerOption); ok {
			dialer = dialOpt.baseDialer()
		}
	}

	if dialer == nil {
		dialer = transport.NetDialer(settings.Timeout)
	}

	tls, err := tlscommon.LoadTLSConfig(settings.TLS)
	if err != nil {
		return nil, err
	}

	tlsDialer := transport.TLSDialer(dialer, tls, settings.Timeout)
	for _, opt := range opts {
		if dialOpt, ok := opt.(dialerModOption); ok {
			dialer = dialOpt.applyDialer(settings, dialer)
			tlsDialer = dialOpt.applyDialer(settings, tlsDialer)
		}
	}

	if logger := extra.logger; logger != nil {
		dialer = transport.LoggingDialer(dialer, logger)
		tlsDialer = transport.LoggingDialer(tlsDialer, logger)
	}

	var rt http.RoundTripper
	if extra.http2 {
		rt, err = settings.http2RoundTripper(tls, dialer, tlsDialer, opts...)
	} else {
		rt, err = settings.httpRoundTripper(tls, dialer, tlsDialer, opts...)
	}

	for _, opt := range opts {
		if rtOpt, ok := opt.(roundTripperOption); ok {
			rt = rtOpt.applyRoundTripper(settings, rt)
		}
	}
	return rt, nil
}

func (settings *HTTPTransportSettings) httpRoundTripper(
	tls *tlscommon.TLSConfig,
	dialer, tlsDialer transport.Dialer,
	opts ...TransportOption,
) (*http.Transport, error) {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = nil
	t.DialTLSContext = nil
	t.Dial = dialer.Dial
	t.DialTLS = tlsDialer.Dial
	t.TLSClientConfig = tls.ToConfig()
	t.ForceAttemptHTTP2 = false
	t.Proxy = settings.Proxy.ProxyFunc()
	t.ProxyConnectHeader = settings.Proxy.Headers.Headers()

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

func (settings *HTTPTransportSettings) http2RoundTripper(
	tls *tlscommon.TLSConfig,
	dialer, tlsDialer transport.Dialer,
	opts ...TransportOption,
) (*http2.Transport, error) {
	t1, err := settings.httpRoundTripper(tls, dialer, tlsDialer, opts...)
	if err != nil {
		return nil, err
	}

	t2, err := http2.ConfigureTransports(t1)
	if err != nil {
		return nil, err
	}

	t2.AllowHTTP = true
	return t2, nil
}

// Client creates a new http.Client with configured Transport. The transport is
// instrumented using apmhttp.WrapRoundTripper.
func (settings HTTPTransportSettings) Client(opts ...TransportOption) (*http.Client, error) {
	rt, err := settings.RoundTripper(opts...)
	if err != nil {
		return nil, err
	}

	return &http.Client{Transport: rt, Timeout: settings.Timeout}, nil
}

func (opts WithKeepaliveSettings) sealTransportOption() {}
func (opts WithKeepaliveSettings) applyTransport(_ *HTTPTransportSettings, t *http.Transport) {
	t.DisableKeepAlives = opts.Disable
	if opts.IdleConnTimeout != 0 {
		t.IdleConnTimeout = opts.IdleConnTimeout
	}
	if opts.MaxIdleConns != 0 {
		t.MaxIdleConns = opts.MaxIdleConns
	}
	if opts.MaxIdleConnsPerHost != 0 {
		t.MaxIdleConnsPerHost = opts.MaxIdleConnsPerHost
	}
}

// WithBaseDialer configures the dialer used for TCP and TLS connections.
func WithBaseDialer(d transport.Dialer) TransportOption {
	return baseDialerFunc(func() transport.Dialer {
		return d
	})
}

// WithIOStats instruments the RoundTripper dialers with the given statser, such
// that bytes in, bytes out, and errors can be monitored.
func WithIOStats(stats transport.IOStatser) TransportOption {
	return dialerOptFunc(func(d transport.Dialer) transport.Dialer {
		if stats == nil {
			return d
		}
		return transport.StatsDialer(d, stats)
	})
}

// WithTransportFunc register a custom function that is used to apply
// custom changes to the net.Transport, when the Client is build.
func WithTransportFunc(fn func(*http.Transport)) TransportOption {
	return transportOptFunc(func(_ *HTTPTransportSettings, t *http.Transport) {
		fn(t)
	})
}

// WithHTTP2Only will ensure that a HTTP 2 only roundtripper is created.
func WithHTTP2Only(b bool) TransportOption {
	return extraOptionFunc(func(settings *extraSettings) {
		settings.http2 = b
	})
}

// WithForceAttemptHTTP2 sets the `http.Tansport.ForceAttemptHTTP2` field.
func WithForceAttemptHTTP2(b bool) TransportOption {
	return transportOptFunc(func(settings *HTTPTransportSettings, t *http.Transport) {
		t.ForceAttemptHTTP2 = b
	})
}

// WithNOProxy disables the configured proxy. Proxy environment variables
// like HTTP_PROXY and HTTPS_PROXY will have no affect.
func WithNOProxy() TransportOption {
	return transportOptFunc(func(s *HTTPTransportSettings, t *http.Transport) {
		t.Proxy = nil
	})
}

// WithoutProxyEnvironmentVariables disables support for the HTTP_PROXY, HTTPS_PROXY and
// NO_PROXY envionrment variables. Explicitely configured proxy URLs will still applied.
func WithoutProxyEnvironmentVariables() TransportOption {
	return transportOptFunc(func(settings *HTTPTransportSettings, t *http.Transport) {
		if settings.Proxy.Disable || settings.Proxy.URL == nil {
			t.Proxy = nil
		}
	})
}

// WithModRoundtripper allows customization of the roundtipper.
func WithModRoundtripper(w func(http.RoundTripper) http.RoundTripper) TransportOption {
	return rtOptFunc(w)
}

var withAPMHTTPRountTripper = WithModRoundtripper(func(rt http.RoundTripper) http.RoundTripper {
	return apmhttp.WrapRoundTripper(rt)
})

// WithAPMHTTPInstrumentation insruments the HTTP client via apmhttp.WrapRoundTripper.
// Custom APM round tripper wrappers can be configured via WithModRoundtripper.
func WithAPMHTTPInstrumentation() TransportOption {
	return withAPMHTTPRountTripper
}

// HeaderRoundTripper will return a RoundTripper that sets header KVs if the key is not present.
func HeaderRoundTripper(rt http.RoundTripper, headers map[string]string) http.RoundTripper {
	return &headerRoundTripper{headers, rt}
}

// WithHeaderRoundTripper instuments the HTTP client via a custom http.RoundTripper.
// This RoundTripper will add headers to each request if the key is not present.
func WithHeaderRoundTripper(headers map[string]string) TransportOption {
	return WithModRoundtripper(func(rt http.RoundTripper) http.RoundTripper {
		return HeaderRoundTripper(rt, headers)
	})
}

// WithLogger sets the internal logger that will be used to log dial or TCP level errors.
// Logging at the connection level will only happen if the logger has been set.
func WithLogger(logger *logp.Logger) TransportOption {
	return extraOptionFunc(func(s *extraSettings) {
		s.logger = logger
	})
}
