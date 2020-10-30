// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const requestNamespace = "request"

func registerRequestTransforms() {
	registerTransform(requestNamespace, appendName, newAppendRequest)
	registerTransform(requestNamespace, deleteName, newDeleteRequest)
	registerTransform(requestNamespace, setName, newSetRequest)
}

type request struct {
	body   common.MapStr
	header http.Header
	url    *url.URL
}

func newRequest(ctx transformContext, body *common.MapStr, url url.URL, trs []requestTransform) (*request, error) {
	req := &request{
		body:   common.MapStr{},
		header: http.Header{},
	}

	clonedURL, err := url.Parse(url.String())
	if err != nil {
		return nil, err
	}
	req.url = clonedURL

	if body != nil {
		req.body.DeepUpdate(*body)
	}

	for _, t := range trs {
		req, err = t.run(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

type requestFactory struct {
	url        url.URL
	method     string
	body       *common.MapStr
	transforms []requestTransform
	user       string
	password   string
	log        *logp.Logger
}

func newRequestFactory(config *requestConfig, authConfig *authConfig, log *logp.Logger) *requestFactory {
	// config validation already checked for errors here
	ts, _ := newRequestTransformsFromConfig(config.Transforms)
	rf := &requestFactory{
		url:        *config.URL.URL,
		method:     config.Method,
		body:       config.Body,
		transforms: ts,
		log:        log,
	}
	if authConfig != nil && authConfig.Basic.isEnabled() {
		rf.user = authConfig.Basic.User
		rf.password = authConfig.Basic.Password
	}
	return rf
}

func (rf *requestFactory) newHTTPRequest(stdCtx context.Context, trCtx transformContext) (*http.Request, error) {
	trReq, err := newRequest(trCtx, rf.body, rf.url, rf.transforms)
	if err != nil {
		return nil, err
	}

	var body []byte
	if len(trReq.body) > 0 {
		switch rf.method {
		case "POST":
			body, err = json.Marshal(trReq.body)
			if err != nil {
				return nil, err
			}
		default:
			rf.log.Errorf("A body is set, but method is not POST. The body will be ignored.")
		}
	}

	req, err := http.NewRequest(rf.method, trReq.url.String(), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(stdCtx)

	req.Header = trReq.header
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if rf.method == "POST" {
		req.Header.Set("Content-Type", "application/json")
	}

	if rf.user != "" || rf.password != "" {
		req.SetBasicAuth(rf.user, rf.password)
	}

	return req, nil
}

type requester struct {
	log               *logp.Logger
	client            *http.Client
	requestFactory    *requestFactory
	responseProcessor *responseProcessor
}

func newRequester(
	client *http.Client,
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

func (r *requester) doRequest(stdCtx context.Context, trCtx transformContext, publisher cursor.Publisher) error {
	req, err := r.requestFactory.newHTTPRequest(stdCtx, trCtx)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	httpResp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute http client.Do: %w", err)
	}
	defer httpResp.Body.Close()

	eventsCh, err := r.responseProcessor.startProcessing(stdCtx, trCtx)
	if err != nil {
		return err
	}

	for maybeEvent := range eventsCh {
		if maybeEvent.failed() {
			r.log.Errorf("error processing response: %v", maybeEvent)
			continue
		}
		if err := publisher.Publish(maybeEvent.event, trCtx.cursor.Clone()); err != nil {
			r.log.Errorf("error publishing event: %v", err)
		}
	}

	return nil
}
