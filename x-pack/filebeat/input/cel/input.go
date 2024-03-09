// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package cel implements an input that uses the Common Expression Language to
// perform requests and do endpoint processing of events. The cel package exposes
// the github.com/elastic/mito/lib CEL extension library.
package cel

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/icholy/digest"
	"github.com/rcrowley/go-metrics"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"google.golang.org/protobuf/types/known/structpb"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/elastic-agent-libs/transport"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	"github.com/elastic/elastic-agent-libs/useragent"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
	"github.com/elastic/mito/lib"
)

const (
	// inputName is the name of the input processor.
	inputName = "cel"

	// root is the label of the object through which the input state is
	// exposed to the CEL program.
	root = "state"
)

// The Filebeat user-agent is provided to the program as useragent.
var userAgent = useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String())

func Plugin(log *logp.Logger, store inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:      inputName,
		Stability: feature.Stable,
		Manager:   NewInputManager(log, store),
	}
}

type input struct {
	time func() time.Time
}

// now is time.Now with a modifiable time source.
func (i input) now() time.Time {
	if i.time == nil {
		return time.Now()
	}
	return i.time()
}

func (input) Name() string { return inputName }

func (input) Test(src inputcursor.Source, _ v2.TestContext) error {
	cfg := src.(*source).cfg
	if !wantClient(cfg) {
		return nil
	}
	return test(cfg.Resource.URL.URL)
}

// Run starts the input and blocks until it ends completes. It will return on
// context cancellation or type invalidity errors, any other error will be retried.
func (input) Run(env v2.Context, src inputcursor.Source, crsr inputcursor.Cursor, pub inputcursor.Publisher) error {
	var cursor map[string]interface{}
	if !crsr.IsNew() { // Allow the user to bootstrap the program if needed.
		err := crsr.Unpack(&cursor)
		if err != nil {
			return err
		}
	}
	return input{}.run(env, src.(*source), cursor, pub)
}

// sanitizeFileName returns name with ":" and "/" replaced with "_", removing repeated instances.
// The request.tracer.filename may have ":" when a httpjson input has cursor config and
// the macOS Finder will treat this as path-separator and causes to show up strange filepaths.
func sanitizeFileName(name string) string {
	name = strings.ReplaceAll(name, ":", string(filepath.Separator))
	name = filepath.Clean(name)
	return strings.ReplaceAll(name, string(filepath.Separator), "_")
}

func (i input) run(env v2.Context, src *source, cursor map[string]interface{}, pub inputcursor.Publisher) error {
	cfg := src.cfg
	log := env.Logger.With("input_url", cfg.Resource.URL)

	metrics := newInputMetrics(env.ID)
	defer metrics.Close()

	ctx := ctxtool.FromCanceller(env.Cancelation)

	if cfg.Resource.Tracer != nil {
		id := sanitizeFileName(env.ID)
		cfg.Resource.Tracer.Filename = strings.ReplaceAll(cfg.Resource.Tracer.Filename, "*", id)
	}

	client, trace, err := newClient(ctx, cfg, log)
	if err != nil {
		return err
	}

	limiter := newRateLimiterFromConfig(cfg.Resource)

	patterns, err := regexpsFromConfig(cfg)
	if err != nil {
		return err
	}

	var auth *lib.BasicAuth
	if cfg.Auth.Basic.isEnabled() {
		auth = &lib.BasicAuth{
			Username: cfg.Auth.Basic.User,
			Password: cfg.Auth.Basic.Password,
		}
	}
	prg, ast, err := newProgram(ctx, cfg.Program, root, client, limiter, auth, patterns, cfg.XSDs, log, trace)
	if err != nil {
		return err
	}

	var state map[string]interface{}
	if cfg.State == nil {
		state = make(map[string]interface{})
	} else {
		state = cfg.State
	}
	if cursor != nil {
		state["cursor"] = cursor
	}
	goodCursor := cursor
	goodURL := cfg.Resource.URL.String()
	state["url"] = goodURL
	metrics.resource.Set(goodURL)
	// On entry, state is expected to be in the shape:
	//
	// {
	//     "url": <resource address>,
	//     "cursor": { ... },
	//     ...
	// }
	//
	// The url field must be present and can be an HTTP end-point
	// or a file path. It is currently the responsibility of the
	// program to handle removing the scheme from a file url if it
	// is present. The url may be mutated during execution of the
	// program but the mutated state will not be persisted between
	// restarts and the url must be present in the returned value
	// to ensure that it is available in the next evaluation unless
	// the program has the resource address hard-coded in or it is
	// available from the cursor.
	//
	// Additional fields may be present at the root of the object
	// and if the program tolerates it, the cursor value may be
	// absent. Only the cursor is persisted over restarts, but
	// all fields in state are retained between iterations of
	// the processing loop except for the produced events array,
	// see discussion below.
	//
	// If the cursor is present the program should perform and
	// process requests based on its value. If cursor is not
	// present the program must have alternative logic to
	// determine what requests to make.
	//
	// In addition to this and the functions and globals available
	// from mito/lib, a global, useragent, is available to use
	// in requests.
	err = periodically(ctx, cfg.Interval, func() error {
		log.Info("process repeated request")
		var (
			budget    = *cfg.MaxExecutions
			waitUntil time.Time
		)
		for {
			if wait := time.Until(waitUntil); wait > 0 {
				// We have a special-case wait for when we have a zero limit.
				// x/time/rate allow a burst through even when the limit is zero
				// so in order to ensure that we don't try until we are out of
				// purgatory we calculate how long we should wait according to
				// the retry after for a 429 and rate limit headers if we have
				// a zero rate quota. See handleResponse below.
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(wait):
				}
			} else if err = ctx.Err(); err != nil {
				// Otherwise exit if we have been cancelled.
				return err
			}

			// Process a set of event requests.
			if trace != nil {
				log.Debugw("previous transaction", "transaction.id", trace.TxID())
			}
			log.Debugw("request state", logp.Namespace("cel"), "state", redactor{state: state, cfg: cfg.Redact})
			metrics.executions.Add(1)
			start := i.now().In(time.UTC)
			state, err = evalWith(ctx, prg, ast, state, start)
			log.Debugw("response state", logp.Namespace("cel"), "state", redactor{state: state, cfg: cfg.Redact})
			if err != nil {
				switch {
				case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
					return err
				}
				log.Errorw("failed evaluation", "error", err)
			}
			metrics.celProcessingTime.Update(time.Since(start).Nanoseconds())
			if trace != nil {
				log.Debugw("final transaction", "transaction.id", trace.TxID())
			}

			// On exit, state is expected to be in the shape:
			//
			// {
			//     "cursor": [
			//         {...},
			//         ...
			//     ],
			//     "events": [
			//         {...},
			//         ...
			//     ],
			//     "url": <resource address>,
			//     "status_code": <HTTP request status code if a network request>,
			//     "header": <HTTP response headers if a network request>,
			//     "rate_limit": <HTTP rate limit map if required by API>,
			//     "want_more": bool
			// }
			//
			// The "events" array must be present, but may be empty or null.
			// In the case of an error condition in the CEL program it is
			// acceptable to return a single object which will be wrapped as
			// an array below. It is the responsibility of the downstream
			// processor to handle this object correctly (which may be to drop
			// the event). The error event will also be logged.
			// If it is not empty, it must only have objects as elements.
			// Additional fields may be present at the root of the object.
			// The evaluation is repeated with the new state, after removing
			// the events field, if the "want_more" field is present and true
			// and a non-zero events array is returned.
			//
			// If cursor is present it must be either a single object or an
			// array with the same length as events; each element i of the
			// cursor array will be the details for obtaining the events at or
			// beyond event i in events. If the cursor is a single object it
			// is will be the details for obtaining events after the last
			// event in the events array and will only be retained on
			// successful publication of all the events in the array.
			//
			// If rate_limit is present it should be a map with numeric fields
			// rate and burst. It may also have a string error field and
			// other fields which will be logged. If it has an error field
			// the rate and burst will not be used to set rate limit behaviour.
			//
			// The status code and rate_limit values may be omitted if they do
			// not contribute to control.
			//
			// The following details how a cursor array works:
			//
			// Result after request resulting in 5 events. Each c obtained with
			// an e points to the ~next e.
			//
			//    +----+   +----+        +----+
			//    | e1 |   | c1 |        | e1 |
			//    +----+   +----+        +----+   +----+
			//    | e2 |   | c2 |        | e2 | < | c1 |
			//    +----+   +----+        +----+   +----+
			//    | e3 |   | c3 |        | e3 | < | c2 |
			//    +----+   +----+   =>   +----+   +----+
			//    | e4 |   | c4 |        | e4 | < | c3 |
			//    +----+   +----+        +----+   +----+
			//    | e5 |   | c5 |        | e5 | < | c4 |
			//    +----+   +----+        +----+   +----+
			//                           |next| < | c5 |
			//                           +----+   +----+
			//
			// After a successful publication this will leave a single c and
			// and empty events array. So the next evaluation has a boot.
			//
			// If the publication fails or execution is terminated at some
			// point during the events array, we may end up with, e.g.
			//
			//    +----+  +----+        +----+   +----+
			//    | e3 |  | c3 |        |next| < | c3 |
			//    +----+  +----+        +----+   +----+
			//    | e4 |  | c4 |   =>
			//    +----+  +----+          lost events
			//    | e5 |  | c5 |
			//    +----+  +----+
			//
			// At this point, the c3 cursor (or at worst the c2 cursor) has
			// been stored and we can continue from that point, recovering
			// the lost events and potentially re-requesting e3.

			var ok bool
			ok, waitUntil, err = handleResponse(log, state, limiter)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}

			_, ok = state["url"]
			if !ok && goodURL != "" {
				state["url"] = goodURL
				log.Debugw("adding missing url from last valid value: state did not contain a url", "last_valid_url", goodURL)
			}

			e, ok := state["events"]
			if !ok {
				log.Error("unexpected missing events array from evaluation")
			}
			var events []interface{}
			switch e := e.(type) {
			case []interface{}:
				if len(e) == 0 {
					return nil
				}
				events = e
			case map[string]interface{}:
				if e == nil {
					return nil
				}
				log.Errorw("single event object returned by evaluation", "event", e)
				events = []interface{}{e}
				// Make sure the cursor is not updated.
				delete(state, "cursor")
			default:
				return fmt.Errorf("unexpected type returned for evaluation events: %T", e)
			}

			// We have a non-empty batch of events to process.
			metrics.batchesReceived.Add(1)
			metrics.eventsReceived.Add(uint64(len(events)))

			// Drop events from state. If we fail during the publication,
			// we will re-request these events.
			delete(state, "events")

			// Get cursors if they exist.
			var (
				cursors      []interface{}
				singleCursor bool
			)
			if c, ok := state["cursor"]; ok {
				cursors, ok = c.([]interface{})
				if ok {
					if len(cursors) != len(events) {
						log.Errorw("unexpected cursor list length", "cursors", len(cursors), "events", len(events))
						// But try to continue.
						if len(cursors) < len(events) {
							cursors = nil
						}
					}
				} else {
					cursors = []interface{}{c}
					singleCursor = true
				}
			}
			// Drop old cursor from state. This will be replaced with
			// the current cursor object below; it is an array now.
			delete(state, "cursor")

			start = time.Now()
			var hadPublicationError bool
			for i, e := range events {
				event, ok := e.(map[string]interface{})
				if !ok {
					return fmt.Errorf("unexpected type returned for evaluation events: %T", e)
				}
				var pubCursor interface{}
				if cursors != nil {
					if singleCursor {
						// Only set the cursor for publication at the last event
						// when a single cursor object has been provided.
						if i == len(events)-1 {
							goodCursor = cursor
							cursor, ok = cursors[0].(map[string]interface{})
							if !ok {
								return fmt.Errorf("unexpected type returned for evaluation cursor element: %T", cursors[0])
							}
							pubCursor = cursor
						}
					} else {
						goodCursor = cursor
						cursor, ok = cursors[i].(map[string]interface{})
						if !ok {
							return fmt.Errorf("unexpected type returned for evaluation cursor element: %T", cursors[i])
						}
						pubCursor = cursor
					}
				}
				err = pub.Publish(beat.Event{
					Timestamp: time.Now(),
					Fields:    event,
				}, pubCursor)
				if err != nil {
					hadPublicationError = true
					log.Errorw("error publishing event", "error", err)
					cursors = nil // We are lost, so retry with this event's cursor,
					continue      // but continue with the events that we have without
					// advancing the cursor. This allows us to potentially publish the
					// events we have now, with a fallback to the last guaranteed
					// correctly published cursor.
				}
				if i == 0 {
					metrics.batchesPublished.Add(1)
				}
				metrics.eventsPublished.Add(1)

				err = ctx.Err()
				if err != nil {
					return err
				}
			}
			metrics.batchProcessingTime.Update(time.Since(start).Nanoseconds())

			// Advance the cursor to the final state if there was no error during
			// publications. This is needed to transition to the next set of events.
			if !hadPublicationError {
				goodCursor = cursor
			}

			// Replace the last known good cursor.
			state["cursor"] = goodCursor

			if more, _ := state["want_more"].(bool); !more {
				return nil
			}

			// Check we have a remaining execution budget.
			budget--
			if budget <= 0 {
				log.Warnw("exceeding maximum number of CEL executions", "limit", *cfg.MaxExecutions)
				return nil
			}
		}
	})
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		log.Infof("input stopped because context was cancelled with: %v", err)
		err = nil
	}
	return err
}

func periodically(ctx context.Context, each time.Duration, fn func() error) error {
	err := fn()
	if err != nil {
		return err
	}
	return timed.Periodic(ctx, each, fn)
}

// handleResponse checks the response status code and handles rate limit changes.
// It returns ok=true if the response is valid, otherwise false for a retry.
func handleResponse(log *logp.Logger, state map[string]interface{}, limiter *rate.Limiter) (ok bool, waitUntil time.Time, err error) {
	var header http.Header
	h, ok := state["header"]
	if ok {
		delete(state, "header")
		switch h := h.(type) {
		case http.Header:
			header = h
		case map[string][]string:
			header = h
		case map[string]interface{}:
			header = make(http.Header)
			for k, v := range h {
				switch v := v.(type) {
				case []string:
					header[k] = v
				case []interface{}:
					vals := make([]string, len(v))
					for i, e := range v {
						vals[i], ok = e.(string)
						if !ok {
							return false, time.Time{}, fmt.Errorf("unexpected type returned for response header value: %T", v)
						}
					}
					header[k] = vals
				default:
					return false, waitUntil, fmt.Errorf("unexpected type returned for response header value set: %T", v)
				}
			}
		default:
			return false, waitUntil, fmt.Errorf("unexpected type returned for response header: %T", h)
		}
	}

	r, ok := state["rate_limit"]
	if ok {
		delete(state, "rate_limit")
		switch r := r.(type) {
		case map[string]interface{}:
			// The state of rate-limit headers is a disaster. This needs to be
			// more robust, but there is no real consensus and the RFC is not
			// past draft yet. The draft is more sane than what we have now, but
			// still has a lot of complexity. Note that the RFC draft says that
			// this behaviour should be in the common path, not just in the 429
			// path.
			waitUntil = handleRateLimit(log, r, header, limiter)
		default:
			return false, waitUntil, fmt.Errorf("unexpected type returned for response header: %T", h)
		}
	}

	sc, ok := state["status_code"]
	if ok {
		delete(state, "status_code")
		var statusCode int
		switch sc := sc.(type) {
		case int:
			statusCode = sc
		case int64:
			statusCode = int(sc)
		case float64:
			statusCode = int(sc)
		default:
			return false, waitUntil, fmt.Errorf("unexpected type returned for request status code: %T", sc)
		}
		switch statusCode {
		case http.StatusOK:
			return true, time.Time{}, nil
		case http.StatusTooManyRequests:
			// https://datatracker.ietf.org/doc/html/rfc6585#page-3
			retry := header.Get("Retry-After")
			if d, err := strconv.Atoi(retry); err == nil {
				t := time.Now().Add(time.Duration(d) * time.Second)
				if t.After(waitUntil) {
					waitUntil = t
				}
			} else if t, err := time.Parse(http.TimeFormat, retry); err == nil {
				if t.After(waitUntil) {
					waitUntil = t
				}
			}
			return false, waitUntil, nil
		default:
			status := http.StatusText(statusCode)
			if status == "" {
				status = "unknown status code"
			}
			state["events"] = errorMessage(fmt.Sprintf("failed http request with %s: %d", status, statusCode))
			return true, time.Time{}, nil
		}
	}
	return true, waitUntil, nil
}

func handleRateLimit(log *logp.Logger, rateLimit map[string]interface{}, header http.Header, limiter *rate.Limiter) (waitUntil time.Time) {
	if e, ok := rateLimit["error"]; ok {
		// The error field should be a string, but we won't quibble here.
		log.Errorw("rate limit error", "error", e, "rate_limit", mapstr.M(rateLimit), "header", header)
		return waitUntil
	}

	limit, ok := getLimit("rate", rateLimit, log)
	if !ok {
		return waitUntil
	}

	var burst int
	b, ok := rateLimit["burst"]
	if !ok {
		log.Warnw("rate limit missing burst", "rate_limit", mapstr.M(rateLimit))
	}
	switch b := b.(type) {
	case int:
		burst = b
	case int64:
		burst = int(b)
	case float64:
		burst = int(b)
	default:
		log.Errorw("unexpected type returned for rate limit burst", "type", fmt.Sprintf("%T", b), "rate_limit", mapstr.M(rateLimit))
	}
	if burst < 1 {
		// Make sure we can make at least one new request, even if we fail
		// to get a non-zero rate.Limit. We could set to zero for the case
		// that limit=rate.Inf, but that detail is not important.
		burst = 1
	}

	// Process reset if we need to wait until reset to avoid a request against a zero quota.
	if limit == 0 {
		w, ok := rateLimit["reset"]
		if ok {
			switch w := w.(type) {
			case time.Time:
				waitUntil = w
				next, ok := getLimit("next", rateLimit, log)
				if !ok {
					return waitUntil
				}
				limiter.SetLimitAt(waitUntil, next)
				limiter.SetBurstAt(waitUntil, burst)
			case string:
				t, err := time.Parse(time.RFC3339, w)
				if err != nil {
					log.Errorw("unexpected value returned for rate limit reset", "value", w, "rate_limit", mapstr.M(rateLimit))
					return waitUntil
				}
				waitUntil = t
				next, ok := getLimit("next", rateLimit, log)
				if !ok {
					return waitUntil
				}
				limiter.SetLimitAt(waitUntil, next)
				limiter.SetBurstAt(waitUntil, burst)
			default:
				log.Errorw("unexpected type returned for rate limit reset", "type", reflect.TypeOf(w).String(), "rate_limit", mapstr.M(rateLimit))
			}
		}
		return waitUntil
	}

	limiter.SetLimit(limit)
	limiter.SetBurst(burst)
	return waitUntil
}

func getLimit(which string, rateLimit map[string]interface{}, log *logp.Logger) (limit rate.Limit, ok bool) {
	r, ok := rateLimit[which]
	if !ok {
		log.Errorw("rate limit missing "+which, "rate_limit", mapstr.M(rateLimit))
		return limit, false
	}
	switch r := r.(type) {
	case rate.Limit:
		limit = r
	case int:
		limit = rate.Limit(r)
	case int64:
		limit = rate.Limit(r)
	case float64:
		limit = rate.Limit(r)
	case string:
		if !strings.EqualFold(r, "inf") {
			log.Errorw("unexpected value returned for rate limit "+which, "value", r, "rate_limit", mapstr.M(rateLimit))
			return limit, false
		}
		limit = rate.Inf
	default:
		log.Errorw("unexpected type returned for rate limit "+which, "type", reflect.TypeOf(r).String(), "rate_limit", mapstr.M(rateLimit))
	}
	return limit, true
}

func newClient(ctx context.Context, cfg config, log *logp.Logger) (*http.Client, *httplog.LoggingRoundTripper, error) {
	if !wantClient(cfg) {
		return nil, nil, nil
	}
	c, err := cfg.Resource.Transport.Client(clientOptions(cfg.Resource.URL.URL, cfg.Resource.KeepAlive.settings())...)
	if err != nil {
		return nil, nil, err
	}

	if cfg.Auth.Digest.isEnabled() {
		var noReuse bool
		if cfg.Auth.Digest.NoReuse != nil {
			noReuse = *cfg.Auth.Digest.NoReuse
		}
		c.Transport = &digest.Transport{
			Transport: c.Transport,
			Username:  cfg.Auth.Digest.User,
			Password:  cfg.Auth.Digest.Password,
			NoReuse:   noReuse,
		}
	}

	var trace *httplog.LoggingRoundTripper
	if cfg.Resource.Tracer != nil {
		w := zapcore.AddSync(cfg.Resource.Tracer)
		go func() {
			// Close the logger when we are done.
			<-ctx.Done()
			cfg.Resource.Tracer.Close()
		}()
		core := ecszap.NewCore(
			ecszap.NewDefaultEncoderConfig(),
			w,
			zap.DebugLevel,
		)
		traceLogger := zap.New(core)

		const margin = 1e3 // 1OkB ought to be enough room for all the remainder of the trace details.
		maxSize := cfg.Resource.Tracer.MaxSize * 1e6
		trace = httplog.NewLoggingRoundTripper(c.Transport, traceLogger, max(0, maxSize-margin))
		c.Transport = trace
	}

	c.CheckRedirect = checkRedirect(cfg.Resource, log)

	if cfg.Resource.Retry.getMaxAttempts() > 1 {
		maxAttempts := cfg.Resource.Retry.getMaxAttempts()
		c = (&retryablehttp.Client{
			HTTPClient:   c,
			Logger:       newRetryLog(log),
			RetryWaitMin: cfg.Resource.Retry.getWaitMin(),
			RetryWaitMax: cfg.Resource.Retry.getWaitMax(),
			RetryMax:     maxAttempts,
			CheckRetry:   retryablehttp.DefaultRetryPolicy,
			Backoff:      retryablehttp.DefaultBackoff,
			ErrorHandler: retryErrorHandler(maxAttempts, log),
		}).StandardClient()
	}

	if cfg.Auth.OAuth2.isEnabled() {
		authClient, err := cfg.Auth.OAuth2.client(ctx, c)
		if err != nil {
			return nil, nil, err
		}
		return authClient, trace, nil
	}

	return c, trace, nil
}

func wantClient(cfg config) bool {
	switch scheme, _, _ := strings.Cut(cfg.Resource.URL.Scheme, "+"); scheme {
	case "http", "https":
		return true
	default:
		return false
	}
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

func checkRedirect(cfg *ResourceConfig, log *logp.Logger) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		log.Debug("http client: checking redirect")
		if len(via) >= cfg.RedirectMaxRedirects {
			log.Debug("http client: max redirects exceeded")
			return fmt.Errorf("stopped after %d redirects", cfg.RedirectMaxRedirects)
		}

		if !cfg.RedirectForwardHeaders || len(via) == 0 {
			log.Debugf("http client: nothing to do while checking redirects - forward_headers: %v, via: %#v", cfg.RedirectForwardHeaders, via)
			return nil
		}

		prev := via[len(via)-1] // previous request to get headers from

		log.Debugf("http client: forwarding headers from previous request: %#v", prev.Header)
		req.Header = prev.Header.Clone()

		for _, k := range cfg.RedirectHeadersBanList {
			log.Debugf("http client: ban header %v", k)
			req.Header.Del(k)
		}

		return nil
	}
}

// retryErrorHandler returns a retryablehttp.ErrorHandler that will log retry resignation
// but return the last retry attempt's response and a nil error so that the CEL code
// can evaluate the response status itself. Any error passed to the retryablehttp.ErrorHandler
// is returned unaltered.
func retryErrorHandler(max int, log *logp.Logger) retryablehttp.ErrorHandler {
	return func(resp *http.Response, err error, numTries int) (*http.Response, error) {
		log.Warnw("giving up retries", "method", resp.Request.Method, "url", resp.Request.URL, "retries", max+1)
		return resp, err
	}
}

func newRateLimiterFromConfig(cfg *ResourceConfig) *rate.Limiter {
	r := rate.Inf
	b := 1
	if cfg != nil && cfg.RateLimit != nil {
		if cfg.RateLimit.Limit != nil {
			r = rate.Limit(*cfg.RateLimit.Limit)
		}
		if cfg.RateLimit.Burst != nil {
			b = *cfg.RateLimit.Burst
		}
	}
	return rate.NewLimiter(r, b)
}

func regexpsFromConfig(cfg config) (map[string]*regexp.Regexp, error) {
	if len(cfg.Regexps) == 0 {
		return nil, nil
	}
	patterns := make(map[string]*regexp.Regexp)
	for name, expr := range cfg.Regexps {
		var err error
		patterns[name], err = regexp.Compile(expr)
		if err != nil {
			return nil, err
		}
	}
	return patterns, nil
}

var (
	// mimetypes holds supported MIME type mappings.
	mimetypes = map[string]interface{}{
		"application/gzip":         func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) },
		"application/x-ndjson":     lib.NDJSON,
		"application/zip":          lib.Zip,
		"text/csv; header=absent":  lib.CSVNoHeader,
		"text/csv; header=present": lib.CSVHeader,

		// Include the undocumented space-less syntax to head off typo-related
		// user issues.
		//
		// TODO: Consider changing the MIME type look-ups to a formal parser
		// rather than a simple map look-up.
		"text/csv;header=absent":  lib.CSVNoHeader,
		"text/csv;header=present": lib.CSVHeader,
	}

	// limitPolicies are the provided rate limit policy helpers.
	limitPolicies = map[string]lib.LimitPolicy{
		"okta":  lib.OktaRateLimit,
		"draft": lib.DraftRateLimit,
	}
)

func newProgram(ctx context.Context, src, root string, client *http.Client, limiter *rate.Limiter, auth *lib.BasicAuth, patterns map[string]*regexp.Regexp, xsd map[string]string, log *logp.Logger, trace *httplog.LoggingRoundTripper) (cel.Program, *cel.Ast, error) {
	xml, err := lib.XML(nil, xsd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build xml type hints: %w", err)
	}
	opts := []cel.EnvOption{
		cel.Declarations(decls.NewVar(root, decls.Dyn)),
		cel.OptionalTypes(cel.OptionalTypesVersion(lib.OptionalTypesVersion)),
		lib.Collections(),
		lib.Crypto(),
		lib.JSON(nil),
		xml,
		lib.Strings(),
		lib.Time(),
		lib.Try(),
		lib.Debug(debug(log, trace)),
		lib.File(mimetypes),
		lib.MIME(mimetypes),
		lib.Limit(limitPolicies),
		lib.Globals(map[string]interface{}{
			"useragent": userAgent,
		}),
	}
	if client != nil {
		opts = append(opts, lib.HTTPWithContext(ctx, client, limiter, auth))
	}
	if len(patterns) != 0 {
		opts = append(opts, lib.Regexp(patterns))
	}
	env, err := cel.NewEnv(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create env: %w", err)
	}

	ast, iss := env.Compile(src)
	if iss.Err() != nil {
		return nil, nil, fmt.Errorf("failed compilation: %w", iss.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, nil, fmt.Errorf("failed program instantiation: %w", err)
	}
	return prg, ast, nil
}

func debug(log *logp.Logger, trace *httplog.LoggingRoundTripper) func(string, any) {
	log = log.Named("cel_debug")
	return func(tag string, value any) {
		level := "DEBUG"
		if _, ok := value.(error); ok {
			level = "ERROR"
		}
		if trace == nil {
			log.Debugw(level, "tag", tag, "value", value)
		} else {
			log.Debugw(level, "tag", tag, "value", value, "transaction.id", trace.TxID())
		}
	}
}

func evalWith(ctx context.Context, prg cel.Program, ast *cel.Ast, state map[string]interface{}, now time.Time) (map[string]interface{}, error) {
	out, _, err := prg.ContextEval(ctx, map[string]interface{}{
		// Replace global program "now" with current time. This is necessary
		// as the lib.Time now global is static at program instantiation time
		// which will persist over multiple evaluations. The lib.Time behaviour
		// is correct for mito where CEL program instances live for only a
		// single evaluation. Rather than incurring the cost of creating a new
		// cel.Program for each evaluation, shadow lib.Time's now with a new
		// value for each eval. We retain the lib.Time now global for
		// compatibility between CEL programs developed in mito with programs
		// run in the input.
		"now": now,
		root:  state,
	})
	if err != nil {
		err = lib.DecoratedError{AST: ast, Err: err}
	}
	if e := ctx.Err(); e != nil {
		err = e
	}
	if err != nil {
		state["events"] = errorMessage(fmt.Sprintf("failed eval: %v", err))
		clearWantMore(state)
		return state, fmt.Errorf("failed eval: %w", err)
	}

	v, err := out.ConvertToNative(reflect.TypeOf((*structpb.Struct)(nil)))
	if err != nil {
		state["events"] = errorMessage(fmt.Sprintf("failed proto conversion: %v", err))
		clearWantMore(state)
		return state, fmt.Errorf("failed proto conversion: %w", err)
	}
	switch v := v.(type) {
	case *structpb.Struct:
		return v.AsMap(), nil
	default:
		// This should never happen.
		errMsg := fmt.Sprintf("unexpected native conversion type: %T", v)
		state["events"] = errorMessage(errMsg)
		clearWantMore(state)
		return state, errors.New(errMsg)
	}
}

// clearWantMore sets the state to not request additional work in a periodic evaluation.
// It leaves state intact if there is no "want_more" element, and sets the element to false
// if there is. This is necessary instead of just doing delete(state, "want_more") as
// client CEL code may expect the want_more field to be present.
func clearWantMore(state map[string]interface{}) {
	if _, ok := state["want_more"]; ok {
		state["want_more"] = false
	}
}

func errorMessage(msg string) map[string]interface{} {
	return map[string]interface{}{"error": map[string]interface{}{"message": msg}}
}

// retryLog is a shim for the retryablehttp.Client.Logger.
type retryLog struct{ log *logp.Logger }

func newRetryLog(log *logp.Logger) *retryLog {
	return &retryLog{log: log.Named("retryablehttp").WithOptions(zap.AddCallerSkip(1))}
}

func (l *retryLog) Error(msg string, kv ...interface{}) { l.log.Errorw(msg, kv...) }
func (l *retryLog) Info(msg string, kv ...interface{})  { l.log.Infow(msg, kv...) }
func (l *retryLog) Debug(msg string, kv ...interface{}) { l.log.Debugw(msg, kv...) }
func (l *retryLog) Warn(msg string, kv ...interface{})  { l.log.Warnw(msg, kv...) }

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
		return fmt.Errorf("url %q is unreachable: %w", url, err)
	}

	return nil
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()

	resource            *monitoring.String // URL-ish of input resource
	executions          *monitoring.Uint   // times the CEL program has been executed
	batchesReceived     *monitoring.Uint   // number of event arrays received
	eventsReceived      *monitoring.Uint   // number of events received
	batchesPublished    *monitoring.Uint   // number of event arrays published
	eventsPublished     *monitoring.Uint   // number of events published
	celProcessingTime   metrics.Sample     // histogram of the elapsed successful cel program processing times in nanoseconds
	batchProcessingTime metrics.Sample     // histogram of the elapsed successful batch processing times in nanoseconds (time of receipt to time of ACK for non-empty batches).
}

func newInputMetrics(id string) *inputMetrics {
	reg, unreg := inputmon.NewInputRegistry(inputName, id, nil)
	out := &inputMetrics{
		unregister:          unreg,
		resource:            monitoring.NewString(reg, "resource"),
		executions:          monitoring.NewUint(reg, "cel_executions"),
		batchesReceived:     monitoring.NewUint(reg, "batches_received_total"),
		eventsReceived:      monitoring.NewUint(reg, "events_received_total"),
		batchesPublished:    monitoring.NewUint(reg, "batches_published_total"),
		eventsPublished:     monitoring.NewUint(reg, "events_published_total"),
		celProcessingTime:   metrics.NewUniformSample(1024),
		batchProcessingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "cel_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.celProcessingTime))
	_ = adapter.NewGoMetrics(reg, "batch_processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.batchProcessingTime))

	return out
}

func (m *inputMetrics) Close() {
	m.unregister()
}

// redactor implements lazy field redaction of sets of a mapstr.M.
type redactor struct {
	state mapstr.M
	cfg   *redact
}

// String renders the JSON corresponding to r.state after applying redaction
// operations.
func (r redactor) String() string {
	if r.cfg == nil || len(r.cfg.Fields) == 0 {
		return r.state.String()
	}
	c := make(mapstr.M, len(r.state))
	cloneMap(c, r.state)
	for _, mask := range r.cfg.Fields {
		if r.cfg.Delete {
			walkMap(c, mask, func(parent mapstr.M, key string) {
				delete(parent, key)
			})
			continue
		}
		walkMap(c, mask, func(parent mapstr.M, key string) {
			parent[key] = "*"
		})
	}
	return c.String()
}

// cloneMap is an enhanced version of mapstr.M.Clone that handles cloning arrays
// within objects. Nested arrays are not handled.
func cloneMap(dst, src mapstr.M) {
	for k, v := range src {
		switch v := v.(type) {
		case mapstr.M:
			d := make(mapstr.M, len(v))
			dst[k] = d
			cloneMap(d, v)
		case map[string]interface{}:
			d := make(map[string]interface{}, len(v))
			dst[k] = d
			cloneMap(d, v)
		case []mapstr.M:
			a := make([]mapstr.M, 0, len(v))
			for _, m := range v {
				d := make(mapstr.M, len(m))
				cloneMap(d, m)
				a = append(a, d)
			}
			dst[k] = a
		case []map[string]interface{}:
			a := make([]map[string]interface{}, 0, len(v))
			for _, m := range v {
				d := make(map[string]interface{}, len(m))
				cloneMap(d, m)
				a = append(a, d)
			}
			dst[k] = a
		default:
			dst[k] = v
		}
	}
}

// walkMap walks to all ends of the provided path in m and applies fn to the
// final element of each walk. Nested arrays are not handled.
func walkMap(m mapstr.M, path string, fn func(parent mapstr.M, key string)) {
	key, rest, more := strings.Cut(path, ".")
	v, ok := m[key]
	if !ok {
		return
	}
	if !more {
		fn(m, key)
		return
	}
	switch v := v.(type) {
	case mapstr.M:
		walkMap(v, rest, fn)
	case map[string]interface{}:
		walkMap(v, rest, fn)
	case []mapstr.M:
		for _, m := range v {
			walkMap(m, rest, fn)
		}
	case []map[string]interface{}:
		for _, m := range v {
			walkMap(m, rest, fn)
		}
	case []interface{}:
		for _, v := range v {
			switch m := v.(type) {
			case mapstr.M:
				walkMap(m, rest, fn)
			case map[string]interface{}:
				walkMap(m, rest, fn)
			}
		}
	}
}
