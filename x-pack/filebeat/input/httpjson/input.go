// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	inputName = "httpjson"
)

var (
	userAgent = useragent.UserAgent("Filebeat")

	// for testing
	timeNow = time.Now
)

// httpJSONInput struct has the HttpJsonInput configuration and other userful info.
type httpJSONInput struct{}

// Plugin create a stateful input Plugin collecting logs from HTTPJSONInput.
func Plugin(log *logp.Logger, store cursor.StateStore) input.Plugin {
	return input.Plugin{
		Name:       inputName,
		Stability:  feature.Beta,
		Deprecated: false,
		Info:       "HTTP JSON Input",
		Manager: &cursor.InputManager{
			Logger:     log.Named(inputName),
			StateStore: store,
			Type:       inputName,
			Configure:  configure,
		},
	}
}

func configure(cfg *common.Config) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	httpClient, err := newHTTPClient(config)
	if err != nil {
		return nil, nil, err
	}

	r := &requester{config: config, client: httpClient}

	in := &httpJSONInput{}

	return []cursor.Source{r}, in, nil
}

func (*httpJSONInput) Name() string { return inputName }

func (*httpJSONInput) Test(source cursor.Source, ctx input.TestContext) error {
	requester := source.(*requester)
	url, err := url.Parse(requester.config.URL)
	if err != nil {
		return err
	}

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

	_, err = net.DialTimeout("tcp", fmt.Sprintf("%s:%s", url.Hostname(), port), time.Second)
	if err != nil {
		return fmt.Errorf("url %q is unreachable", requester.config.URL)
	}

	return nil
}

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (in *httpJSONInput) Run(
	ctx input.Context,
	source cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	requester := source.(*requester)

	log := ctx.Logger.With("url", requester.config.URL)
	requester.log = log

	requester.loadCheckpoint(cursor)

	stdCtx := ctxtool.FromCanceller(ctx.Cancelation)

	ri := &requestInfo{
		contentMap: common.MapStr{},
		headers:    requester.config.HTTPHeaders,
	}

	if requester.config.HTTPMethod == "POST" &&
		requester.config.HTTPRequestBody != nil {
		ri.contentMap.Update(common.MapStr(requester.config.HTTPRequestBody))
	}

	err := requester.processHTTPRequest(stdCtx, publisher, ri)
	if err == nil && requester.config.Interval > 0 {
		ticker := time.NewTicker(requester.config.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-stdCtx.Done():
				log.Info("Context done.")
				return nil
			case <-ticker.C:
				log.Info("Process another repeated request.")
				err = requester.processHTTPRequest(stdCtx, publisher, ri)
				if err != nil {
					return err
				}
			}
		}
	}

	return err
}

func newHTTPClient(config config) (*http.Client, error) {
	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	// Make retryable HTTP client
	var client *retryablehttp.Client = &retryablehttp.Client{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: config.HTTPClientTimeout,
				}).DialContext,
				TLSClientConfig:   tlsConfig.ToConfig(),
				DisableKeepAlives: true,
			},
			Timeout: config.HTTPClientTimeout,
		},
		Logger:       newRetryLogger(),
		RetryWaitMin: config.RetryWaitMin,
		RetryWaitMax: config.RetryWaitMax,
		RetryMax:     config.RetryMax,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
	}

	if config.OAuth2.IsEnabled() {
		return config.OAuth2.Client(client.StandardClient())
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
