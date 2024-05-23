// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/mito/lib/xml"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httpmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/useragent"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
)

const (
	inputName = "httpjson"
)

var (
	userAgent = useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

	// for testing
	timeNow = time.Now
)

// retryLogger provides a shim for a *logp.Logger to be used by
// go-retryablehttp as a retryablehttp.LeveledLogger.
type retryLogger struct {
	log *logp.Logger
}

func newRetryLogger(log *logp.Logger) *retryLogger {
	return &retryLogger{
		log: log.Named("retryablehttp").WithOptions(zap.AddCallerSkip(1)),
	}
}

func (log *retryLogger) Error(msg string, keysAndValues ...interface{}) {
	log.log.Errorw(msg, keysAndValues...)
}

func (log *retryLogger) Info(msg string, keysAndValues ...interface{}) {
	log.log.Infow(msg, keysAndValues...)
}

func (log *retryLogger) Debug(msg string, keysAndValues ...interface{}) {
	log.log.Debugw(msg, keysAndValues...)
}

func (log *retryLogger) Warn(msg string, keysAndValues ...interface{}) {
	log.log.Warnw(msg, keysAndValues...)
}

func Plugin(log *logp.Logger, store inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Manager:    NewInputManager(log, store),
	}
}

func test(url *url.URL) error {
	port := func() string {
		if url.Port() != "" {
			return url.Port()
		}
		switch url.Scheme {
		case "https":
			return "443"
		}
		return "80"
	}()

	_, err := net.DialTimeout("tcp", net.JoinHostPort(url.Hostname(), port), time.Second)
	if err != nil {
		return fmt.Errorf("url %q is unreachable", url)
	}

	return nil
}

func runWithMetrics(ctx v2.Context, cfg config, pub inputcursor.Publisher, crsr *inputcursor.Cursor) error {
	reg, unreg := inputmon.NewInputRegistry("httpjson", ctx.ID, nil)
	defer unreg()
	return run(ctx, cfg, pub, crsr, reg)
}

func run(ctx v2.Context, cfg config, pub inputcursor.Publisher, crsr *inputcursor.Cursor, reg *monitoring.Registry) error {
	log := ctx.Logger.With("input_url", cfg.Request.URL)

	stdCtx := ctxtool.FromCanceller(ctx.Cancelation)

	if cfg.Request.Tracer != nil {
		id := sanitizeFileName(ctx.ID)
		cfg.Request.Tracer.Filename = strings.ReplaceAll(cfg.Request.Tracer.Filename, "*", id)

		// Propagate tracer behaviour to all chain children.
		for i, c := range cfg.Chain {
			if c.Step != nil { // Request is validated as required.
				cfg.Chain[i].Step.Request.Tracer = cfg.Request.Tracer
			}
			if c.While != nil { // Request is validated as required.
				cfg.Chain[i].While.Request.Tracer = cfg.Request.Tracer
			}
		}
	}

	metrics := newInputMetrics(reg)

	client, err := newHTTPClient(stdCtx, cfg, log, reg)
	if err != nil {
		return err
	}

	requestFactory, err := newRequestFactory(stdCtx, cfg, log, metrics, reg)
	if err != nil {
		log.Errorf("Error while creating requestFactory: %v", err)
		return err
	}
	var xmlDetails map[string]xml.Detail
	if cfg.Response.XSD != "" {
		xmlDetails, err = xml.Details([]byte(cfg.Response.XSD))
		if err != nil {
			log.Errorf("error while collecting xml decoder type hints: %v", err)
			return err
		}
	}
	pagination := newPagination(cfg, client, log)
	responseProcessor := newResponseProcessor(cfg, pagination, xmlDetails, metrics, log)
	requester := newRequester(client, requestFactory, responseProcessor, log)

	trCtx := emptyTransformContext()
	trCtx.cursor = newCursor(cfg.Cursor, log)
	trCtx.cursor.load(crsr)

	doFunc := func() error {
		defer func() {
			// Clear response bodies between evaluations.
			trCtx.firstResponse.body = nil
			trCtx.lastResponse.body = nil
		}()

		log.Info("Process another repeated request.")

		startTime := time.Now()

		var err error
		if err = requester.doRequest(stdCtx, trCtx, pub); err != nil {
			log.Errorf("Error while processing http request: %v", err)
		}

		metrics.updateIntervalMetrics(err, startTime)

		if err := stdCtx.Err(); err != nil {
			return err
		}

		return nil
	}

	// we trigger the first call immediately,
	// then we schedule it on the given interval using timed.Periodic
	if err = doFunc(); err == nil {
		err = timed.Periodic(stdCtx, cfg.Interval, doFunc)
	}

	log.Infof("Input stopped because context was cancelled with: %v", err)

	return nil
}

// sanitizeFileName returns name with ":" and "/" replaced with "_", removing repeated instances.
// The request.tracer.filename may have ":" when a httpjson input has cursor config and
// the macOS Finder will treat this as path-separator and causes to show up strange filepaths.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, ":", string(filepath.Separator))
	name = filepath.Clean(name)
	return strings.ReplaceAll(name, string(filepath.Separator), "_")
}

func newHTTPClient(ctx context.Context, config config, log *logp.Logger, reg *monitoring.Registry) (*httpClient, error) {
	client, err := newNetHTTPClient(ctx, config.Request, log, reg)
	if err != nil {
		return nil, err
	}

	if config.Request.Retry.getMaxAttempts() > 1 {
		// Make retryable HTTP client if needed.
		client = (&retryablehttp.Client{
			HTTPClient:   client,
			Logger:       newRetryLogger(log),
			RetryWaitMin: config.Request.Retry.getWaitMin(),
			RetryWaitMax: config.Request.Retry.getWaitMax(),
			RetryMax:     config.Request.Retry.getMaxAttempts(),
			CheckRetry:   retryablehttp.DefaultRetryPolicy,
			Backoff:      retryablehttp.DefaultBackoff,
		}).StandardClient()
	}

	limiter := newRateLimiterFromConfig(config.Request.RateLimit, log)

	if config.Auth.OAuth2.isEnabled() {
		authClient, err := config.Auth.OAuth2.client(ctx, client)
		if err != nil {
			return nil, err
		}
		return &httpClient{client: authClient, limiter: limiter}, nil
	}

	return &httpClient{client: client, limiter: limiter}, nil
}

func newNetHTTPClient(ctx context.Context, cfg *requestConfig, log *logp.Logger, reg *monitoring.Registry) (*http.Client, error) {
	netHTTPClient, err := cfg.Transport.Client(clientOptions(cfg.URL.URL, cfg.KeepAlive.settings())...)
	if err != nil {
		return nil, err
	}

	if cfg.Tracer != nil {
		w := zapcore.AddSync(cfg.Tracer)
		go func() {
			// Close the logger when we are done.
			<-ctx.Done()
			cfg.Tracer.Close()
		}()
		core := ecszap.NewCore(
			ecszap.NewDefaultEncoderConfig(),
			w,
			zap.DebugLevel,
		)
		traceLogger := zap.New(core)

		const margin = 1e3 // 1OkB ought to be enough room for all the remainder of the trace details.
		maxSize := cfg.Tracer.MaxSize*1e6 - margin
		if maxSize < 0 {
			maxSize = 0
		}
		netHTTPClient.Transport = httplog.NewLoggingRoundTripper(netHTTPClient.Transport, traceLogger, maxSize, log)
	}

	if reg != nil {
		netHTTPClient.Transport = httpmon.NewMetricsRoundTripper(netHTTPClient.Transport, reg)
	}

	netHTTPClient.CheckRedirect = checkRedirect(cfg, log)

	return netHTTPClient, nil
}

func newChainHTTPClient(ctx context.Context, authCfg *authConfig, requestCfg *requestConfig, log *logp.Logger, reg *monitoring.Registry, p ...*Policy) (*httpClient, error) {
	client, err := newNetHTTPClient(ctx, requestCfg, log, reg)
	if err != nil {
		return nil, err
	}

	var retryPolicyFunc retryablehttp.CheckRetry
	if len(p) != 0 {
		retryPolicyFunc = p[0].CustomRetryPolicy
	} else {
		retryPolicyFunc = retryablehttp.DefaultRetryPolicy
	}

	if requestCfg.Retry.getMaxAttempts() > 1 {
		// Make retryable HTTP client if needed.
		client = (&retryablehttp.Client{
			HTTPClient:   client,
			Logger:       newRetryLogger(log),
			RetryWaitMin: requestCfg.Retry.getWaitMin(),
			RetryWaitMax: requestCfg.Retry.getWaitMax(),
			RetryMax:     requestCfg.Retry.getMaxAttempts(),
			CheckRetry:   retryPolicyFunc,
			Backoff:      retryablehttp.DefaultBackoff,
		}).StandardClient()
	}

	limiter := newRateLimiterFromConfig(requestCfg.RateLimit, log)

	if authCfg != nil && authCfg.OAuth2.isEnabled() {
		authClient, err := authCfg.OAuth2.client(ctx, client)
		if err != nil {
			return nil, err
		}
		return &httpClient{client: authClient, limiter: limiter}, nil
	}

	return &httpClient{client: client, limiter: limiter}, nil
}

// clientOption returns constructed client configuration options, including
// setting up http+unix and http+npipe transports if requested.
func clientOptions(u *url.URL, keepalive httpcommon.WithKeepaliveSettings) []httpcommon.TransportOption {
	scheme, trans, ok := strings.Cut(u.Scheme, "+")
	var dialer transport.Dialer
	switch {
	default:
		fallthrough
	case !ok:
		return []httpcommon.TransportOption{
			httpcommon.WithAPMHTTPInstrumentation(),
			keepalive,
		}

	// We set the host for the unix socket and Windows named
	// pipes schemes because the http.Transport expects to
	// have a host and will error out if it is not present.
	// The values here are just non-zero with a helpful name.
	// They are not used in any logic.
	case trans == "unix":
		u.Host = "unix-socket"
		dialer = socketDialer{u.Path}
	case trans == "npipe":
		u.Host = "windows-npipe"
		dialer = npipeDialer{u.Path}
	}
	u.Scheme = scheme
	return []httpcommon.TransportOption{
		httpcommon.WithAPMHTTPInstrumentation(),
		keepalive,
		httpcommon.WithBaseDialer(dialer),
	}
}

// socketDialer implements transport.Dialer to a constant socket path.
type socketDialer struct {
	path string
}

func (d socketDialer) Dial(_, _ string) (net.Conn, error) {
	return net.Dial("unix", d.path)
}

func (d socketDialer) DialContext(ctx context.Context, _, _ string) (net.Conn, error) {
	var nd net.Dialer
	return nd.DialContext(ctx, "unix", d.path)
}

func checkRedirect(config *requestConfig, log *logp.Logger) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		log.Debug("http client: checking redirect")
		if len(via) >= config.RedirectMaxRedirects {
			log.Debug("http client: max redirects exceeded")
			return fmt.Errorf("stopped after %d redirects", config.RedirectMaxRedirects)
		}

		if !config.RedirectForwardHeaders || len(via) == 0 {
			log.Debugf("http client: nothing to do while checking redirects - forward_headers: %v, via: %#v", config.RedirectForwardHeaders, via)
			return nil
		}

		prev := via[len(via)-1] // previous request to get headers from

		log.Debugf("http client: forwarding headers from previous request: %#v", prev.Header)
		req.Header = prev.Header.Clone()

		for _, k := range config.RedirectHeadersBanList {
			log.Debugf("http client: ban header %v", k)
			req.Header.Del(k)
		}

		return nil
	}
}

func makeEvent(body mapstr.M) (beat.Event, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return beat.Event{}, err
	}
	now := timeNow()
	fields := mapstr.M{
		"event": mapstr.M{
			"created": now,
		},
		"message": string(bodyBytes),
	}

	return beat.Event{
		Timestamp: now,
		Fields:    fields,
	}, nil
}
