// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"bytes"
	"context"
	"encoding/json"
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

func newRequest(ctx transformContext, body *common.MapStr, url url.URL, trs []basicTransform) (*transformable, error) {
	req := emptyTransformable()
	req.url = url

	if body != nil {
		req.body.DeepUpdate(*body)
	}

	var err error
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
	transforms []basicTransform
	user       string
	password   string
	log        *logp.Logger
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

func (r *requester) doRequest(stdCtx context.Context, trCtx transformContext, publisher inputcursor.Publisher) error {
	req, err := r.requestFactory.newHTTPRequest(stdCtx, trCtx)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	httpResp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute http client.Do: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode > 399 {
		body, _ := ioutil.ReadAll(httpResp.Body)
		return fmt.Errorf("server responded with status code %d: %s", httpResp.StatusCode, string(body))
	}

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

		if err := publisher.Publish(event, trCtx.cursor.clone()); err != nil {
			r.log.Errorf("error publishing event: %v", err)
			continue
		}

		*trCtx.lastEvent = maybeMsg.msg
		trCtx.cursor.update(trCtx)
		n += 1
	}

	r.log.Infof("request finished: %d events published", n)
	return nil
}
