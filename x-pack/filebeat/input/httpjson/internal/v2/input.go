// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
)

const (
	inputName = "httpjsonv2"
)

var (
	userAgent = useragent.UserAgent("Filebeat")

	// for testing
	timeNow = time.Now
)

type retryLogger struct {
	log *logp.Logger
}

func newRetryLogger() *retryLogger {
	return &retryLogger{
		log: logp.NewLogger("httpjson.retryablehttp", zap.AddCallerSkip(1)),
	}
}

func (log *retryLogger) Error(format string, args ...interface{}) {
	log.log.Errorf(format, args...)
}

func (log *retryLogger) Info(format string, args ...interface{}) {
	log.log.Infof(format, args...)
}

func (log *retryLogger) Debug(format string, args ...interface{}) {
	log.log.Debugf(format, args...)
}

func (log *retryLogger) Warn(format string, args ...interface{}) {
	log.log.Warnf(format, args...)
}

func newTLSConfig(config config) (*tlscommon.TLSConfig, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(config.Request.SSL)
	if err != nil {
		return nil, err
	}

	return tlsConfig, nil
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

func run(
	ctx v2.Context,
	config config,
	tlsConfig *tlscommon.TLSConfig,
	publisher inputcursor.Publisher,
	cursor *inputcursor.Cursor,
) error {
	log := ctx.Logger.With("url", config.Request.URL)

	stdCtx := ctxtool.FromCanceller(ctx.Cancelation)

	httpClient, err := newHTTPClient(stdCtx, config, tlsConfig, log)
	if err != nil {
		return err
	}

	requestFactory := newRequestFactory(config.Request, config.Auth, log)
	pagination := newPagination(config, httpClient, log)
	responseProcessor := newResponseProcessor(config.Response, pagination, log)
	requester := newRequester(httpClient, requestFactory, responseProcessor, log)

	trCtx := emptyTransformContext()
	trCtx.cursor = newCursor(config.Cursor, log)
	trCtx.cursor.load(cursor)

	err = timed.Periodic(stdCtx, config.Interval, func() error {
		log.Info("Process another repeated request.")

		if err := requester.doRequest(stdCtx, trCtx, publisher); err != nil {
			log.Errorf("Error while processing http request: %v", err)
		}

		if stdCtx.Err() != nil {
			return err
		}

		return nil
	})

	log.Infof("Context done: %v", err)

	return nil
}

func newHTTPClient(ctx context.Context, config config, tlsConfig *tlscommon.TLSConfig, log *logp.Logger) (*httpClient, error) {
	timeout := config.Request.getTimeout()

	// Make retryable HTTP client
	client := &retryablehttp.Client{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: timeout,
				}).DialContext,
				TLSClientConfig:   tlsConfig.ToConfig(),
				DisableKeepAlives: true,
			},
			Timeout:       timeout,
			CheckRedirect: checkRedirect(config.Request),
		},
		Logger:       newRetryLogger(),
		RetryWaitMin: config.Request.Retry.getWaitMin(),
		RetryWaitMax: config.Request.Retry.getWaitMax(),
		RetryMax:     config.Request.Retry.getMaxAttempts(),
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}

	limiter := newRateLimiterFromConfig(config.Request.RateLimit, log)

	if config.Auth.OAuth2.isEnabled() {
		authClient, err := config.Auth.OAuth2.client(ctx, client.StandardClient())
		if err != nil {
			return nil, err
		}
		return &httpClient{client: authClient, limiter: limiter}, nil
	}

	return &httpClient{client: client.StandardClient(), limiter: limiter}, nil
}

func checkRedirect(config *requestConfig) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= config.RedirectMaxRedirects {
			return fmt.Errorf("stopped after %d redirects", config.RedirectMaxRedirects)
		}

		if !config.RedirectHeadersForward || len(via) == 0 {
			return nil
		}

		prev := via[len(via)-1] // previous request to get headers from

		req.Header = prev.Header.Clone()

		if !config.RedirectLocationTrusted {
			for _, k := range config.RedirectHeadersBanList {
				req.Header.Del(k)
			}
		}

		return nil
	}
}

func makeEvent(body common.MapStr) (beat.Event, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return beat.Event{}, err
	}
	now := timeNow()
	fields := common.MapStr{
		"event": common.MapStr{
			"created": now,
		},
		"message": string(bodyBytes),
	}

	return beat.Event{
		Timestamp: now,
		Fields:    fields,
	}, nil
}
