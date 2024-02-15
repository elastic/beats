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
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.elastic.co/apm/module/apmhttp/v2"
	"golang.org/x/net/http2"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/transport"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

var (
	ErrResponseLimit = errors.New("HTTP response length limit was reached")
)

// HTTPTransportSettings provides common HTTP settings for HTTP clients.
type HTTPTransportSettings struct {
	// TLS provides ssl/tls setup settings
	TLS *tlscommon.Config `config:"ssl" yaml:"ssl,omitempty" json:"ssl,omitempty"`

	// Timeout configures the `(http.Transport).Timeout`.
	Timeout time.Duration `config:"timeout" yaml:"timeout,omitempty" json:"timeout,omitempty"`

	Proxy HTTPClientProxySettings `config:",inline" yaml:",inline"`

	IdleConnTimeout time.Duration `config:"idle_connection_timeout" yaml:"idle_connection_timeout,omitempty" json:"idle_connection_timeout,omitempty"`

	// Add more settings:
	//  - DisableKeepAlive
	//  - MaxIdleConns
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
func (settings *HTTPTransportSettings) Unpack(cfg *config.C) error {
	tmp := struct {
		TLS             *tlscommon.Config `config:"ssl"`
		Timeout         time.Duration     `config:"timeout"`
		IdleConnTimeout time.Duration     `config:"idle_connection_timeout"`
	}{
		Timeout:         settings.Timeout,
		IdleConnTimeout: settings.IdleConnTimeout,
	}

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
		TLS:             tmp.TLS,
		Timeout:         tmp.Timeout,
		Proxy:           proxy,
		IdleConnTimeout: tmp.IdleConnTimeout,
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
		if err != nil {
			return nil, err
		}
	} else {
		rt = settings.httpRoundTripper(tls, dialer, tlsDialer, opts...)
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
) *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = dialer.DialContext
	t.DialTLSContext = tlsDialer.DialContext
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

	return t
}

func (settings *HTTPTransportSettings) http2RoundTripper(
	tls *tlscommon.TLSConfig,
	dialer, tlsDialer transport.Dialer,
	opts ...TransportOption,
) (*http2.Transport, error) {
	t1 := settings.httpRoundTripper(tls, dialer, tlsDialer, opts...)
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
// NO_PROXY envionrment variables. Explicitly configured proxy URLs will still applied.
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

// WithHeaderRoundTripper instruments the HTTP client via a custom http.RoundTripper.
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

// ReadAll returns the whole response body as bytes.
// This is an optimized version of `io.ReadAll`.
//
// Use `ReadAllWithLimit` with a reasonable limit when possible! Avoid reading HTTP responses without a limit!
// A malicious server might serve a `Content-Length` header with a value too high to handle
// or the server might serve a response body that is too long and can crash the client with OOM.
func ReadAll(resp *http.Response) ([]byte, error) {
	return ReadAllWithLimit(resp, -1)
}

// ReadAllWithLimit returns the whole response body as bytes respecting the given limit.
// This is an optimized version of `io.ReadAll`.
//
// If the `limit` is 0, an empty byte slice is returned.
// If the `limit` is a negative value, e.g `-1`, the limit is ignored and the entire response body is returned.
// If the `Content-Length` header was served and its value exceeds the limit, the `ErrResponseLimit` error is returned.
// If the body length is not known in advance, it reads from the body up to the set limit and returns a partial response without an error.
//
// Avoid reading HTTP responses without a limit and use a reasonable limit instead!
// A malicious server might serve a `Content-Length` header with a value too high to handle
// or the server might serve a response body that is too long and can crash the client with OOM.
func ReadAllWithLimit(resp *http.Response, limit int64) ([]byte, error) {
	if resp == nil {
		return nil, errors.New("response cannot be nil")
	}
	switch {
	// nothing to read according to the server or limit
	case resp.ContentLength == 0 || limit == 0 || resp.StatusCode == http.StatusNoContent:
		return []byte{}, nil

	// here if the limit is negative, e.g. `-1` it's ignored,
	// limit == 0 is handled above
	case limit > 0 && resp.ContentLength > limit:
		return nil, fmt.Errorf("received Content-Length %d exceeds the set limit %d: %w", resp.ContentLength, limit, ErrResponseLimit)

	// if we know the body length, we can allocate the buffer only once for the most efficient read
	case resp.ContentLength >= 0:
		body := make([]byte, resp.ContentLength)
		_, err := io.ReadFull(resp.Body, body)
		if err != nil {
			return nil, fmt.Errorf("failed to read the response body with a known length %d: %w", resp.ContentLength, err)
		}
		return body, nil

	default:
		// using `bytes.NewBuffer` + `io.Copy` is much faster than `io.ReadAll`
		// see https://github.com/elastic/beats/issues/36151#issuecomment-1931696767
		buf := bytes.NewBuffer(nil)
		var err error
		if limit > 0 {
			_, err = io.Copy(buf, io.LimitReader(resp.Body, limit))
		} else {
			_, err = io.Copy(buf, resp.Body)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read the response body with unknown length: %w", err)
		}
		body := buf.Bytes()
		if body == nil {
			body = []byte{}
		}
		return body, nil
	}
}
