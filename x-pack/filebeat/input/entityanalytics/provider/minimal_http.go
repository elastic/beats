// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/paths"
	"github.com/elastic/beats/v7/pkg/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/lumberjack"
)

// TracerConfig configures request trace logging for a minimal-state
// provider's HTTP client. It accepts the same options as the legacy
// providers' tracer.
type TracerConfig struct {
	Enabled           *bool `config:"enabled"`
	lumberjack.Logger `config:",inline"`
}

// Validate checks that an enabled tracer has a file name, and defaults its
// maximum file size to 1MB.
func (c *TracerConfig) Validate() error {
	if !c.enabled() {
		return nil
	}
	if c.Filename == "" {
		return errors.New("request tracer must have a filename if used")
	}
	if c.MaxSize == 0 {
		// By default Lumberjack caps file sizes at 100MB which
		// is excessive for a debugging logger, so default to 1MB
		// which is the minimum.
		c.MaxSize = 1
	}
	return nil
}

func (c *TracerConfig) enabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}

// TraceRequests returns next wrapped with request trace logging if cfg
// enables it, and next otherwise. The trace file name is resolved within
// the logs directory for input, with any * replaced by id. If cfg names a
// file but is disabled, existing trace files for that name are removed.
// The trace file is closed when ctx is done. cfg is not modified.
func TraceRequests(ctx context.Context, next http.RoundTripper, cfg *TracerConfig, p *paths.Path, input, id string, log *logp.Logger) (http.RoundTripper, error) {
	if cfg == nil || cfg.Filename == "" {
		return next, nil
	}
	filename, err := httplog.ResolveTraceFilename(p, input, id, cfg.Filename)
	if err != nil {
		return nil, err
	}
	if !cfg.enabled() {
		httplog.CleanTraceFiles(filename, log)
		return next, nil
	}

	file := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
		MaxBackups: cfg.MaxBackups,
		LocalTime:  cfg.LocalTime,
		Compress:   cfg.Compress,
	}
	w := zapcore.AddSync(file)
	go func() {
		<-ctx.Done()
		if err := file.Close(); err != nil {
			log.Errorw("failed to close request trace log", "path", filename, "error", err)
		}
	}()
	core := ecszap.NewCore(
		ecszap.NewDefaultEncoderConfig(),
		w,
		zap.DebugLevel,
	)
	maxBodyLen := max(1, cfg.MaxSize) * 1e6 / 10 // 10% of file max
	return httplog.NewLoggingRoundTripper(next, zap.New(core), maxBodyLen, []string{"Authorization"}, log), nil
}

// SkipTracing returns a RoundTripper that sends requests under one of
// prefixes through direct, and all other requests through traced. A
// request is under a prefix URL if it has the same scheme and host,
// ignoring case, and its path starts with the prefix's path. It is used to
// keep requests that carry credentials in their bodies, such as token
// requests, out of the request trace.
func SkipTracing(traced, direct http.RoundTripper, prefixes []string) (http.RoundTripper, error) {
	if len(prefixes) == 0 {
		return traced, nil
	}
	urls := make([]*url.URL, 0, len(prefixes))
	for _, p := range prefixes {
		u, err := url.Parse(p)
		if err != nil {
			return nil, fmt.Errorf("untraced URL prefix %q: %w", p, err)
		}
		urls = append(urls, u)
	}
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		for _, u := range urls {
			if under(req.URL, u) {
				return direct.RoundTrip(req)
			}
		}
		return traced.RoundTrip(req)
	}), nil
}

// under reports whether u is under the prefix URL p.
func under(u, p *url.URL) bool {
	return strings.EqualFold(u.Scheme, p.Scheme) &&
		strings.EqualFold(u.Host, p.Host) &&
		strings.HasPrefix(u.Path, p.Path)
}

// roundTripperFunc is an http.RoundTripper implemented by a function.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// RequestConfig is the request block of a minimal-state provider that
// accepts the legacy request.* options.
type RequestConfig struct {
	KeepAlive              KeepAlive `config:"keep_alive"`
	RedirectForwardHeaders bool      `config:"redirect.forward_headers"`
	RedirectHeadersBanList []string  `config:"redirect.headers_ban_list"`
	RedirectMaxRedirects   int       `config:"redirect.max_redirects"`

	// RedirectSensitiveHeaders lists the headers that are not forwarded
	// on a redirect to another host or from https to http. If it is nil,
	// defaultSensitiveHeaders are used. An empty list forwards them all.
	// The default is applied after unpacking, not set in
	// DefaultRequestConfig, because go-ucfg merges a configured list into
	// a default list element by element, so a configured list could not
	// replace it.
	RedirectSensitiveHeaders []string `config:"redirect.sensitive_headers"`

	// Retry holds the legacy retry options. They are accepted so that
	// configurations written for the legacy providers still load, but
	// they are not used: minimal-state providers retry requests
	// themselves.
	Retry *config.C `config:"retry"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

// DefaultRequestConfig returns a RequestConfig with the given client
// timeout and the legacy providers' default of ten redirects.
func DefaultRequestConfig(timeout time.Duration) RequestConfig {
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = timeout
	return RequestConfig{
		RedirectMaxRedirects: 10,
		Transport:            transport,
	}
}

// Validate checks the redirect limit.
func (c RequestConfig) Validate() error {
	if c.RedirectMaxRedirects < 0 {
		return errors.New("redirect.max_redirects must not be negative")
	}
	return nil
}

// Client returns an HTTP client configured by c. It logs a warning if
// request.retry options are set, since they are not used.
func (c RequestConfig) Client(log *logp.Logger) (*http.Client, error) {
	if c.Retry != nil {
		log.Warn("request.retry options are not used with use_minimal_state; the provider retries failed requests itself")
	}
	opts := []httpcommon.TransportOption{
		httpcommon.WithLogger(log),
		httpcommon.WithAPMHTTPInstrumentation(),
	}
	if c.KeepAlive.set() {
		opts = append(opts, c.KeepAlive.settings())
	}
	cli, err := c.Transport.Client(opts...)
	if err != nil {
		return nil, err
	}
	cli.CheckRedirect = checkRedirect(c.RedirectMaxRedirects, c.RedirectForwardHeaders, sensitiveHeaders(c.RedirectSensitiveHeaders), c.RedirectHeadersBanList)
	return cli, nil
}

// KeepAlive holds the request.keep_alive options. Unlike the legacy
// providers, keep-alives stay enabled unless they are disabled
// explicitly, as minimal-state providers have always used them.
type KeepAlive struct {
	Disable             *bool         `config:"disable"`
	MaxIdleConns        int           `config:"max_idle_connections"`
	MaxIdleConnsPerHost int           `config:"max_idle_connections_per_host"`
	IdleConnTimeout     time.Duration `config:"idle_connection_timeout"`
}

// Validate checks that no keep-alive limit is negative. As in the legacy
// providers, the limits are not checked when keep-alives are disabled.
func (c KeepAlive) Validate() error {
	if c.Disable != nil && *c.Disable {
		return nil
	}
	switch {
	case c.MaxIdleConns < 0:
		return errors.New("max_idle_connections must not be negative")
	case c.MaxIdleConnsPerHost < 0:
		return errors.New("max_idle_connections_per_host must not be negative")
	case c.IdleConnTimeout < 0:
		return errors.New("idle_connection_timeout must not be negative")
	}
	return nil
}

func (c KeepAlive) set() bool {
	return c != KeepAlive{}
}

func (c KeepAlive) settings() httpcommon.WithKeepaliveSettings {
	return httpcommon.WithKeepaliveSettings{
		Disable:             c.Disable != nil && *c.Disable,
		MaxIdleConns:        c.MaxIdleConns,
		MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
		IdleConnTimeout:     c.IdleConnTimeout,
	}
}

// checkRedirect returns an http.Client.CheckRedirect function that stops
// after maxRedirects redirects. If forward is true, the previous request's
// headers are copied to the redirected request, without the headers in
// sensitive when the redirect goes to another host or from https to http,
// and without the headers in ban. Headers are never logged.
func checkRedirect(maxRedirects int, forward bool, sensitive, ban []string) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("stopped after %d redirects", maxRedirects)
		}
		if !forward || len(via) == 0 {
			return nil
		}
		prev := via[len(via)-1]
		req.Header = prev.Header.Clone() //nolint:gosec // G119: sensitive headers are removed below for cross-origin redirects
		if req.URL.Host != prev.URL.Host || (prev.URL.Scheme == "https" && req.URL.Scheme == "http") {
			for _, k := range sensitive {
				req.Header.Del(k)
			}
		}
		for _, k := range ban {
			req.Header.Del(k)
		}
		return nil
	}
}

// sensitiveHeaders returns the configured headers, or
// defaultSensitiveHeaders if the option is unset. An empty, non-nil list
// disables stripping.
func sensitiveHeaders(configured []string) []string {
	if configured == nil {
		return defaultSensitiveHeaders
	}
	return configured
}

// defaultSensitiveHeaders are removed from forwarded headers when a
// redirect leaves the previous request's host or downgrades from https to
// http, unless redirect.sensitive_headers is set.
var defaultSensitiveHeaders = []string{"Authorization", "Proxy-Authorization", "Cookie"}
