// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjson/config"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/timed"
)

const (
	inputName = "httpjson"
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

type input struct {
	config    config.Config
	tlsConfig *tlscommon.TLSConfig
}

func Plugin(log *logp.Logger, store cursor.StateStore) v2.Plugin {
	sim := stateless.NewInputManager(statelessConfigure)
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Beta,
		Deprecated: false,
		Manager: inputManager{
			stateless: &sim,
			cursor: &cursor.InputManager{
				Logger:     log,
				StateStore: store,
				Type:       inputName,
				Configure:  cursorConfigure,
			},
		},
	}
}

func newInput(config config.Config) (*input, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	return &input{
		config:    config,
		tlsConfig: tlsConfig,
	}, nil
}

func (*input) Name() string { return inputName }

func (in *input) test() error {
	port := func() string {
		if in.config.URL.Port() != "" {
			return in.config.URL.Port()
		}
		switch in.config.URL.Scheme {
		case "https":
			return "443"
		}
		return "80"
	}()

	_, err := net.DialTimeout("tcp", net.JoinHostPort(in.config.URL.Hostname(), port), time.Second)
	if err != nil {
		return fmt.Errorf("url %q is unreachable", in.config.URL)
	}

	return nil
}

func (in *input) run(
	ctx v2.Context,
	publisher cursor.Publisher,
	cursor *cursor.Cursor,
) error {
	log := ctx.Logger.With("url", in.config.URL)

	stdCtx := ctxtool.FromCanceller(ctx.Cancelation)

	httpClient, err := in.newHTTPClient(stdCtx)
	if err != nil {
		return err
	}

	dateCursor := newDateCursorFromConfig(in.config, log)

	rateLimiter := newRateLimiterFromConfig(in.config, log)

	pagination := newPaginationFromConfig(in.config)

	requester := newRequester(
		in.config,
		rateLimiter,
		dateCursor,
		pagination,
		httpClient,
		log,
	)

	requester.loadCursor(cursor, log)

	// TODO: disallow passing interval = 0 as a mean to run once.
	if in.config.Interval == 0 {
		return requester.processHTTPRequest(stdCtx, publisher)
	}

	err = timed.Periodic(stdCtx, in.config.Interval, func() error {
		log.Info("Process another repeated request.")
		if err := requester.processHTTPRequest(stdCtx, publisher); err != nil {
			log.Error(err)
		}
		return nil
	})

	log.Infof("Context done: %v", err)

	return nil
}

func (in *input) newHTTPClient(ctx context.Context) (*http.Client, error) {
	// Make retryable HTTP client
	client := &retryablehttp.Client{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: in.config.HTTPClientTimeout,
				}).DialContext,
				TLSClientConfig:   in.tlsConfig.ToConfig(),
				DisableKeepAlives: true,
			},
			Timeout: in.config.HTTPClientTimeout,
		},
		Logger:       newRetryLogger(),
		RetryWaitMin: in.config.RetryWaitMin,
		RetryWaitMax: in.config.RetryWaitMax,
		RetryMax:     in.config.RetryMax,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}

	if in.config.OAuth2.IsEnabled() {
		return in.config.OAuth2.Client(ctx, client.StandardClient())
	}

	return client.StandardClient(), nil
}

func makeEvent(body string) beat.Event {
	fields := common.MapStr{
		"event": common.MapStr{
			"created": time.Now().UTC(),
		},
		"message": body,
	}

	return beat.Event{
		Timestamp: time.Now().UTC(),
		Fields:    fields,
	}
}
