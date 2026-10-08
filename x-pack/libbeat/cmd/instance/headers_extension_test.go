// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/filebeat/cmd"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// fakeHeadersSetter models the contrib headers_setter extension: a component
// implementing extensionauth.HTTPClient whose RoundTripper upserts a fixed
// set of headers on every request.
type fakeHeadersSetter struct {
	headers map[string]string
	err     error
}

func (f *fakeHeadersSetter) Start(context.Context, component.Host) error { return nil }
func (f *fakeHeadersSetter) Shutdown(context.Context) error              { return nil }

func (f *fakeHeadersSetter) RoundTripper(base http.RoundTripper) (http.RoundTripper, error) {
	if f.err != nil {
		return nil, f.err
	}
	return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		for k, v := range f.headers {
			r.Header.Set(k, v)
		}
		return base.RoundTrip(r)
	}), nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// plainExtension does not implement extensionauth.HTTPClient.
type plainExtension struct{}

func (plainExtension) Start(context.Context, component.Host) error { return nil }
func (plainExtension) Shutdown(context.Context) error              { return nil }

func headersSetterID(name string) component.ID {
	return component.NewIDWithName(component.MustNewType(headersSetterType), name)
}

// sendThrough wraps a stub transport with wrap and sends one request, returning
// the headers the stub transport saw.
func sendThrough(t *testing.T, wrap func(http.RoundTripper) http.RoundTripper) http.Header {
	t.Helper()
	var seen http.Header
	base := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		seen = r.Header.Clone()
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: r}, nil
	})
	req, err := http.NewRequest(http.MethodGet, "http://example.invalid/", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "generated-user-agent")
	resp, err := wrap(base).RoundTrip(req)
	require.NoError(t, err)
	resp.Body.Close()
	return seen
}

func TestHTTPTransportWrapperFromExtensions(t *testing.T) {
	const ua = "Elastic-Filebeat/9.4.0 (linux; amd64; Managed; Privileged)"

	t.Run("no extensions", func(t *testing.T) {
		wrap := httpTransportWrapperFromExtensions(map[component.ID]component.Component{}, logptest.NewTestingLogger(t, ""))
		assert.Nil(t, wrap, "no headers_setter means no wrapper")
	})

	t.Run("other extension types are ignored even if they implement HTTPClient", func(t *testing.T) {
		exts := map[component.ID]component.Component{
			component.MustNewID("beatsauth"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": ua}},
		}
		wrap := httpTransportWrapperFromExtensions(exts, logptest.NewTestingLogger(t, ""))
		assert.Nil(t, wrap, "only headers_setter should be picked up")
	})

	t.Run("headers_setter without HTTPClient is skipped", func(t *testing.T) {
		exts := map[component.ID]component.Component{headersSetterID(""): plainExtension{}}
		wrap := httpTransportWrapperFromExtensions(exts, logptest.NewTestingLogger(t, ""))
		assert.Nil(t, wrap, "an extension without RoundTripper cannot be used")
	})

	t.Run("headers_setter headers are applied to requests", func(t *testing.T) {
		exts := map[component.ID]component.Component{
			headersSetterID(""): &fakeHeadersSetter{headers: map[string]string{"User-Agent": ua, "X-Elastic-Team": "obs"}},
		}
		wrap := httpTransportWrapperFromExtensions(exts, logptest.NewTestingLogger(t, ""))
		require.NotNil(t, wrap, "a headers_setter implementing HTTPClient should yield a wrapper")

		seen := sendThrough(t, wrap)
		assert.Equal(t, ua, seen.Get("User-Agent"), "extension User-Agent should replace the generated one")
		assert.Equal(t, "obs", seen.Get("X-Elastic-Team"), "other extension headers should be added too")
	})

	t.Run("nil base falls back to http.DefaultTransport", func(t *testing.T) {
		exts := map[component.ID]component.Component{
			headersSetterID(""): &fakeHeadersSetter{headers: map[string]string{"User-Agent": ua}},
		}
		wrap := httpTransportWrapperFromExtensions(exts, logptest.NewTestingLogger(t, ""))
		require.NotNil(t, wrap)

		var gotUA string
		srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotUA = r.Header.Get("User-Agent")
		}))
		defer srv.Close()

		client := &http.Client{Transport: wrap(nil)}
		resp, err := client.Get(srv.URL)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, ua, gotUA, "request through the wrapped default transport should carry the extension User-Agent")
	})

	t.Run("RoundTripper error falls back to the base transport", func(t *testing.T) {
		exts := map[component.ID]component.Component{
			headersSetterID(""): &fakeHeadersSetter{err: errors.New("additional_auth missing")},
		}
		wrap := httpTransportWrapperFromExtensions(exts, logptest.NewTestingLogger(t, ""))
		require.NotNil(t, wrap, "the wrapper is still returned; the failure happens per client")

		seen := sendThrough(t, wrap)
		assert.Equal(t, "generated-user-agent", seen.Get("User-Agent"), "the request should go through untouched")
	})

	t.Run("multiple headers_setter: first by ID is used", func(t *testing.T) {
		exts := map[component.ID]component.Component{
			headersSetterID("b"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": "from-b"}},
			headersSetterID("a"): &fakeHeadersSetter{headers: map[string]string{"User-Agent": "from-a"}},
		}
		wrap := httpTransportWrapperFromExtensions(exts, logptest.NewTestingLogger(t, ""))
		require.NotNil(t, wrap)

		seen := sendThrough(t, wrap)
		assert.Equal(t, "from-a", seen.Get("User-Agent"), "selection must be deterministic by ID order")
	})
}

// TestBeatReceiverStart_WiresHeadersSetter checks that Start finds the
// headers_setter extension on the host and exposes its RoundTripper through
// beat.Info.HTTPTransportWrapper before the beater runs.
func TestBeatReceiverStart_WiresHeadersSetter(t *testing.T) {
	const ua = "Elastic-Filebeat/9.4.0 (linux; amd64; Managed; Privileged)"

	mb := &mockReceiverBeater{
		acked:    &atomic.Int64{},
		initDone: make(chan struct{}),
		done:     make(chan struct{}),
	}
	// Capture what the beater sees in Run, which is where inputs copy
	// beat.Info from.
	var seenWrapper atomic.Pointer[func(http.RoundTripper) http.RoundTripper]
	creator := func(*beat.Beat, *conf.C) (beat.Beater, error) {
		return beaterFunc{
			run: func(b *beat.Beat) error {
				if b.Info.HTTPTransportWrapper != nil {
					w := b.Info.HTTPTransportWrapper
					seenWrapper.Store(&w)
				}
				return mb.Run(b)
			},
			stop: mb.Stop,
		}, nil
	}

	b, err := NewBeatForReceiver(
		cmd.FilebeatSettings("filebeat"),
		map[string]any{"path.home": t.TempDir()},
		consumertest.NewNop(),
		"test-receiver",
		zapcore.NewNopCore(),
	)
	require.NoError(t, err, "building the receiver beat should succeed")
	require.Nil(t, b.Info.HTTPTransportWrapper, "no wrapper should be set before Start")

	var rs receiver.Settings
	rs.Logger = zap.NewNop()
	rs.ID = component.NewIDWithName(component.MustNewType("mockbeatreceiver"), "r1")

	br, err := NewBeatReceiver(t.Context(), b, creator, rs)
	require.NoError(t, err, "creating the beat receiver should succeed")

	host := &fakeExtensionHost{extensions: map[component.ID]component.Component{
		headersSetterID(""): &fakeHeadersSetter{headers: map[string]string{"User-Agent": ua}},
	}}

	startErr := make(chan error, 1)
	go func() { startErr <- br.Start(host) }()

	select {
	case <-mb.initDone:
	case <-time.After(30 * time.Second):
		t.Fatal("beater did not start")
	}

	wrap := seenWrapper.Load()
	require.NotNil(t, wrap, "the beater should see the wrapper in Run")
	seen := sendThrough(t, *wrap)
	assert.Equal(t, ua, seen.Get("User-Agent"), "the wrapper should apply the extension's headers")

	shutdownDone := make(chan error, 1)
	go func() { shutdownDone <- br.Shutdown(t.Context()) }()
	select {
	case err := <-shutdownDone:
		require.NoError(t, err, "Shutdown should not error")
	case <-time.After(30 * time.Second):
		t.Fatal("Shutdown hung")
	}
	select {
	case err := <-startErr:
		require.NoError(t, err, "beater.Run should return cleanly")
	case <-time.After(10 * time.Second):
		t.Fatal("beater.Run did not return after Stop")
	}
}

// beaterFunc adapts a pair of functions to beat.Beater.
type beaterFunc struct {
	run  func(*beat.Beat) error
	stop func()
}

func (f beaterFunc) Run(b *beat.Beat) error { return f.run(b) }
func (f beaterFunc) Stop()                  { f.stop() }
