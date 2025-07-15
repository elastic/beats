// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httpmon"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/private"
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

func Plugin(log *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Manager:    NewInputManager(log, store),
	}
}

type redact struct {
	value  mapstrM
	fields []string
}

func (r redact) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	v, err := private.Redact(r.value, "", r.fields)
	if err != nil {
		return fmt.Errorf("could not redact value: %v", err)
	}
	return v.MarshalLogObject(enc)
}

// mapstrM is a non-mutating version of mapstr.M.
// See https://github.com/elastic/elastic-agent-libs/issues/232.
type mapstrM mapstr.M

// MarshalLogObject implements the zapcore.ObjectMarshaler interface and allows
// for more efficient marshaling of mapstrM in structured logging.
func (m mapstrM) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if len(m) == 0 {
		return nil
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		if inner, ok := tryToMapStr(v); ok {
			err := enc.AddObject(k, inner)
			if err != nil {
				return fmt.Errorf("failed to add object: %w", err)
			}
			continue
		}
		zap.Any(k, v).AddTo(enc)
	}
	return nil
}

func tryToMapStr(v interface{}) (mapstrM, bool) {
	switch m := v.(type) {
	case mapstrM:
		return m, true
	case map[string]interface{}:
		return mapstrM(m), true
	default:
		return nil, false
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
	stat := ctx.StatusReporter
	if stat == nil {
		stat = noopReporter{}
	}
	stat.UpdateStatus(status.Starting, "")
	stat.UpdateStatus(status.Configuring, "")

	log := ctx.Logger.With("input_url", cfg.Request.URL)
	stdCtx := ctxtool.FromCanceller(ctx.Cancelation)

	if cfg.Request.Tracer != nil {
		id := sanitizeFileName(ctx.IDWithoutName)
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

	client, err := newHTTPClient(stdCtx, cfg.Auth, cfg.Request, stat, log, reg, nil)
	if err != nil {
		stat.UpdateStatus(status.Failed, "failed to create HTTP client: "+err.Error())
		return err
	}

	requestFactory, err := newRequestFactory(stdCtx, cfg, stat, log, metrics, reg)
	if err != nil {
		log.Errorf("Error while creating requestFactory: %v", err)
		stat.UpdateStatus(status.Failed, "failed to create request factory: "+err.Error())
		return err
	}
	var xmlDetails map[string]xml.Detail
	if cfg.Response.XSD != "" {
		xmlDetails, err = xml.Details([]byte(cfg.Response.XSD))
		if err != nil {
			log.Errorf("error while collecting xml decoder type hints: %v", err)
			stat.UpdateStatus(status.Failed, "error while collecting xml decoder type hints: "+err.Error())
			return err
		}
	}
	pagination := newPagination(cfg, client, stat, log)
	responseProcessor := newResponseProcessor(cfg, pagination, xmlDetails, metrics, stat, log)
	requester := newRequester(client, requestFactory, responseProcessor, metrics, stat, log)

	trCtx := emptyTransformContext()
	trCtx.cursor = newCursor(cfg.Cursor, stat, log)
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
			stat.UpdateStatus(status.Stopping, "")
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
	stat.UpdateStatus(status.Stopped, "")
	return nil
}

type noopReporter struct{}

func (noopReporter) UpdateStatus(status.Status, string) {}

// sanitizeFileName returns name with ":" and "/" replaced with "_", removing repeated instances.
// The request.tracer.filename may have ":" when a httpjson input has cursor config and
// the macOS Finder will treat this as path-separator and causes to show up strange filepaths.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, ":", string(filepath.Separator))
	name = filepath.Clean(name)
	return strings.ReplaceAll(name, string(filepath.Separator), "_")
}

// newHTTPClient returns a new httpClient based on the provided configuration values and
// sharing common OAuth2 client if it is configured. If authCfg.OAuth2.isEnabled() is true
// and there is no prepared OAuth2 client, one will be constructed and cached in the
// authCfg.OAuth2.prepared field, otherwise the existing cached client will be used.
func newHTTPClient(ctx context.Context, authCfg *authConfig, requestCfg *requestConfig, stat status.StatusReporter, log *logp.Logger, reg *monitoring.Registry, p *Policy) (*httpClient, error) {
	var (
		client *http.Client
		err    error
	)
	if authCfg.OAuth2.isEnabled() {
		client = authCfg.OAuth2.prepared
		if client == nil {
			client, err = newNetHTTPClient(ctx, requestCfg, log, reg)
			if err != nil {
				return nil, err
			}
			client, err = authCfg.OAuth2.client(ctx, client)
			if err != nil {
				return nil, err
			}
			authCfg.OAuth2.prepared = client
		}
	} else {
		client, err = newNetHTTPClient(ctx, requestCfg, log, reg)
		if err != nil {
			return nil, err
		}
	}

	if requestCfg.Retry.getMaxAttempts() > 1 {
		retryPolicy := retryablehttp.DefaultRetryPolicy
		if p != nil {
			retryPolicy = p.CustomRetryPolicy
		}
		// Make retryable HTTP client if needed.
		client = (&retryablehttp.Client{
			HTTPClient:   client,
			Logger:       newRetryLogger(log),
			RetryWaitMin: requestCfg.Retry.getWaitMin(),
			RetryWaitMax: requestCfg.Retry.getWaitMax(),
			RetryMax:     requestCfg.Retry.getMaxAttempts(),
			CheckRetry:   retryPolicy,
			Backoff:      retryablehttp.DefaultBackoff,
		}).StandardClient()
	}

	limiter := newRateLimiterFromConfig(requestCfg.RateLimit, stat, log)

	return &httpClient{client: client, limiter: limiter}, nil
}

// lumberjackTimestamp is a glob expression matching the time format string used
// by lumberjack when rolling over logs, "2006-01-02T15-04-05.000".
// https://github.com/natefinch/lumberjack/blob/4cb27fcfbb0f35cb48c542c5ea80b7c1d18933d0/lumberjack.go#L39
const lumberjackTimestamp = "[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]-[0-9][0-9]-[0-9][0-9].[0-9][0-9][0-9]"

func newNetHTTPClient(ctx context.Context, cfg *requestConfig, log *logp.Logger, reg *monitoring.Registry) (*http.Client, error) {
	netHTTPClient, err := cfg.Transport.Client(clientOptions(cfg.URL.URL, cfg.KeepAlive.settings())...)
	if err != nil {
		return nil, err
	}

	if cfg.Tracer.enabled() {
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

		maxBodyLen := cfg.Tracer.MaxSize * 1e6 / 10 // 10% of file max
		netHTTPClient.Transport = httplog.NewLoggingRoundTripper(netHTTPClient.Transport, traceLogger, maxBodyLen, log)
	} else if cfg.Tracer != nil {
		// We have a trace log name, but we are not enabled,
		// so remove all trace logs we own.
		err = os.Remove(cfg.Tracer.Filename)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Errorw("failed to remove request trace log", "path", cfg.Tracer.Filename, "error", err)
		}
		ext := filepath.Ext(cfg.Tracer.Filename)
		base := strings.TrimSuffix(cfg.Tracer.Filename, ext)
		paths, err := filepath.Glob(base + "-" + lumberjackTimestamp + ext)
		if err != nil {
			log.Errorw("failed to collect request trace log path names", "error", err)
		}
		for _, p := range paths {
			err = os.Remove(p)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				log.Errorw("failed to remove request trace log", "path", p, "error", err)
			}
		}
	}

	if reg != nil {
		netHTTPClient.Transport = httpmon.NewMetricsRoundTripper(netHTTPClient.Transport, reg)
	}

	netHTTPClient.CheckRedirect = checkRedirect(cfg, log)

	return netHTTPClient, nil
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
