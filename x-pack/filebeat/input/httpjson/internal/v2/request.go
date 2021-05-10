// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const requestNamespace = "request"

func registerRequestTransforms() {
	registerTransform(requestNamespace, appendName, newAppendRequest)
	registerTransform(requestNamespace, deleteName, newDeleteRequest)
	registerTransform(requestNamespace, setName, newSetRequest)
}

type httpClient struct {
	client  *http.Client
	limiter *rateLimiter
}

func (c *httpClient) do(stdCtx context.Context, trCtx *transformContext, req *http.Request) (*http.Response, error) {
	resp, err := c.limiter.execute(stdCtx, func() (*http.Response, error) {
		return c.client.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute http client.Do: %w", err)
	}
	if resp.StatusCode > 399 {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("server responded with status code %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}

func (rf *requestFactory) newRequest(ctx *transformContext) (transformable, error) {
	req := transformable{}
	req.setURL(rf.url)

	if rf.body != nil && len(*rf.body) > 0 {
		req.setBody(rf.body.Clone())
	}

	header := http.Header{}
	header.Set("Accept", "application/json")
	header.Set("User-Agent", userAgent)
	req.setHeader(header)

	var err error
	for _, t := range rf.transforms {
		req, err = t.run(ctx, req)
		if err != nil {
			return transformable{}, err
		}
	}

	if rf.method == "POST" {
		header = req.header()
		if header.Get("Content-Type") == "" {
			header.Set("Content-Type", "application/json")
			req.setHeader(header)
		}
	}

	rf.log.Debugf("new request: %#v", req)

	return req, nil
}

type requestFactory struct {
	url        url.URL
	method     string
	body       *common.MapStr
	transforms []basicTransform
	user       string
	password   string
	log        *logp.Logger
	encoder    encoderFunc
}

func newRequestFactory(config *requestConfig, authConfig *authConfig, log *logp.Logger) *requestFactory {
	// config validation already checked for errors here
	ts, _ := newBasicTransformsFromConfig(config.Transforms, requestNamespace, log)
	rf := &requestFactory{
		url:        *config.URL.URL,
		method:     config.Method,
		body:       config.Body,
		transforms: ts,
		log:        log,
		encoder:    registeredEncoders[config.EncodeAs],
	}
	if authConfig != nil && authConfig.Basic.isEnabled() {
		rf.user = authConfig.Basic.User
		rf.password = authConfig.Basic.Password
	}
	return rf
}

func (rf *requestFactory) newHTTPRequest(stdCtx context.Context, trCtx *transformContext) (*http.Request, error) {
	trReq, err := rf.newRequest(trCtx)
	if err != nil {
		return nil, err
	}

	var body []byte
	if rf.method == "POST" {
		if rf.encoder != nil {
			body, err = rf.encoder(trReq)
		} else {
			body, err = encode(trReq.header().Get("Content-Type"), trReq)
		}
		if err != nil {
			return nil, err
		}
	}

	url := trReq.url()
	req, err := http.NewRequest(rf.method, url.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(stdCtx)

	req.Header = trReq.header().Clone()

	if rf.user != "" || rf.password != "" {
		req.SetBasicAuth(rf.user, rf.password)
	}

	return req, nil
}

type requester struct {
	log               *logp.Logger
	client            *httpClient
	requestFactory    *requestFactory
	responseProcessor *responseProcessor
}

func newRequester(
	client *httpClient,
	requestFactory *requestFactory,
	responseProcessor *responseProcessor,
	log *logp.Logger) *requester {
	return &requester{
		log:               log,
		client:            client,
		requestFactory:    requestFactory,
		responseProcessor: responseProcessor,
	}
}

func (r *requester) doRequest(stdCtx context.Context, trCtx *transformContext, publisher inputcursor.Publisher) error {
	req, err := r.requestFactory.newHTTPRequest(stdCtx, trCtx)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	httpResp, err := r.client.do(stdCtx, trCtx, req)
	if err != nil {
		return fmt.Errorf("failed to execute http client.Do: %w", err)
	}
	defer httpResp.Body.Close()

	eventsCh, err := r.responseProcessor.startProcessing(stdCtx, trCtx, httpResp)
	if err != nil {
		return err
	}

	var n int
	for maybeMsg := range eventsCh {
		if maybeMsg.failed() {
			r.log.Errorf("error processing response: %v", maybeMsg)
			continue
		}

		event, err := makeEvent(maybeMsg.msg)
		if err != nil {
			r.log.Errorf("error creating event: %v", maybeMsg)
			continue
		}

		if err := publisher.Publish(event, trCtx.cursorMap()); err != nil {
			r.log.Errorf("error publishing event: %v", err)
			continue
		}
		if len(*trCtx.firstEventClone()) == 0 {
			trCtx.updateFirstEvent(maybeMsg.msg)
		}
		trCtx.updateLastEvent(maybeMsg.msg)
		trCtx.updateCursor()
		n++
	}

	r.log.Infof("request finished: %d events published", n)

	return nil
}
