// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/PaesslerAG/jsonpath"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const requestNamespace = "request"

func registerRequestTransforms() {
	registerTransform(requestNamespace, appendName, newAppendRequest)
	registerTransform(requestNamespace, deleteName, newDeleteRequest)
	registerTransform(requestNamespace, setName, newSetRequestPagination)
}

type httpClient struct {
	client  *http.Client
	limiter *rateLimiter
}

<<<<<<< HEAD
func (c *httpClient) do(stdCtx context.Context, trCtx *transformContext, req *http.Request) (*http.Response, error) {
=======
func (c *httpClient) do(stdCtx context.Context, req *http.Request) (*http.Response, error) {
>>>>>>> 2ec07f5eca ([Filebeat][httpjson] httpjson chain calls (#29816))
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
	replace    string
}

func newRequestFactory(config config, log *logp.Logger) []*requestFactory {
	// config validation already checked for errors here
	rfs := make([]*requestFactory, 0, len(config.Chain)+1)
	ts, _ := newBasicTransformsFromConfig(config.Request.Transforms, requestNamespace, log)
	// regular call requestFactory object
	rf := &requestFactory{
		url:        *config.Request.URL.URL,
		method:     config.Request.Method,
		body:       config.Request.Body,
		transforms: ts,
		log:        log,
		encoder:    registeredEncoders[config.Request.EncodeAs],
	}
	if config.Auth != nil && config.Auth.Basic.isEnabled() {
		rf.user = config.Auth.Basic.User
		rf.password = config.Auth.Basic.Password
	}
	rfs = append(rfs, rf)
	for _, ch := range config.Chain {
		// chain calls requestFactory object
		ts, _ := newBasicTransformsFromConfig(ch.Step.Request.Transforms, requestNamespace, log)
		rf := &requestFactory{
			url:        *ch.Step.Request.URL.URL,
			method:     ch.Step.Request.Method,
			body:       ch.Step.Request.Body,
			transforms: ts,
			log:        log,
			encoder:    registeredEncoders[config.Request.EncodeAs],
			replace:    ch.Step.Replace,
		}
		rfs = append(rfs, rf)
	}
	return rfs
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
	log                *logp.Logger
	client             *httpClient
	requestFactories   []*requestFactory
	responseProcessors []*responseProcessor
}

func newRequester(
	client *httpClient,
	requestFactory []*requestFactory,
	responseProcessor []*responseProcessor,
	log *logp.Logger) *requester {
	return &requester{
		log:                log,
		client:             client,
		requestFactories:   requestFactory,
		responseProcessors: responseProcessor,
	}
}

// collectResponse returns response from provided request
func (rf *requestFactory) collectResponse(stdCtx context.Context, trCtx *transformContext, r *requester) (*http.Response, error) {
	req, err := rf.newHTTPRequest(stdCtx, trCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpResp, err := r.client.do(stdCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute http client.Do: %w", err)
	}
	return httpResp, nil
}

// generateNewUrl returns new url value using replacement from oldUrl with ids
func generateNewUrl(replacement, oldUrl, id string) (url.URL, error) {
	newUrl, err := url.Parse(strings.Replace(oldUrl, replacement, id, 1))
	if err != nil {
		return url.URL{}, fmt.Errorf("failed to replace value in url: %w", err)
	}
	return *newUrl, nil
}

func (r *requester) doRequest(stdCtx context.Context, trCtx *transformContext, publisher inputcursor.Publisher) error {
	var (
		n                 int
		ids               []string
		err               error
		urlCopy           url.URL
		urlString         string
		httpResp          *http.Response
		intermediateResps []*http.Response
		finalResps        []*http.Response
	)
	for i, rf := range r.requestFactories {
		finalResps = nil
		intermediateResps = nil
		// iterate over collected ids from last response
		if i == 0 {
			// perform and store regular call responses
			httpResp, err = rf.collectResponse(stdCtx, trCtx, r) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
			if err != nil {
				return fmt.Errorf("failed to execute rf.collectResponse: %w", err)
			}
			if len(r.requestFactories) == 1 {
				finalResps = append(finalResps, httpResp) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
				n = r.processAndPublishEvents(stdCtx, trCtx, publisher, finalResps, true, i)
				continue
			}
			intermediateResps = append(intermediateResps, httpResp) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
			ids, err = r.getIdsFromResponses(intermediateResps, r.requestFactories[i+1].replace)
			if err != nil {
				return err
			}
			n = r.processAndPublishEvents(stdCtx, trCtx, publisher, intermediateResps, false, i)
		} else {
			if len(ids) == 0 {
				n = 0
				continue
			}
			urlCopy = rf.url
			urlString = rf.url.String()
			// perform request over collected ids
			for _, id := range ids {
				// reformat urls of requestFactory using ids
				rf.url, err = generateNewUrl(rf.replace, urlString, id)
				if err != nil {
					return fmt.Errorf("failed to generate new URL: %w", err)
				}

				// collect data from new urls
				httpResp, err = rf.collectResponse(stdCtx, trCtx, r) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
				if err != nil {
					return fmt.Errorf("failed to execute rf.collectResponse: %w", err)
				}
				// store data according to response type
				if i == len(r.requestFactories)-1 && len(ids) != 0 {
					finalResps = append(finalResps, httpResp) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
				} else {
					intermediateResps = append(intermediateResps, httpResp) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
				}
			}
			rf.url = urlCopy

			var resps []*http.Response
			if i == len(r.requestFactories)-1 {
				resps = finalResps
			} else {
				// The if comdition (i < len(r.requestFactories)) ensures this branch never runs to the last element
				// of r.requestFactories, therefore r.requestFactories[i+1] will never be out of bounds.
				ids, err = r.getIdsFromResponses(intermediateResps, r.requestFactories[i+1].replace)
				if err != nil {
					return err
				}
				resps = intermediateResps
			}
			n = r.processAndPublishEvents(stdCtx, trCtx, publisher, resps, i < len(r.requestFactories), i)
		}
	}

	defer httpResp.Body.Close()
	r.log.Infof("request finished: %d events published", n)

	return nil
}

// getIdsFromResponses returns ids from responses
func (r *requester) getIdsFromResponses(intermediateResps []*http.Response, replace string) ([]string, error) {
	var b []byte
	var ids []string
	var err error
	// collect ids from all responses
	for _, resp := range intermediateResps {
		defer resp.Body.Close()
		if resp.Body != nil {
			b, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("error while reading response body: %w", err)
			}
		}

		// get replace values from collected json
		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, fmt.Errorf("cannot unmarshal data: %w", err)
		}
		values, err := jsonpath.Get(replace, v)
		if err != nil {
			return nil, fmt.Errorf("error while getting keys: %w", err)
		}

		switch tresp := values.(type) {
		case []interface{}:
			for _, v := range tresp {
				switch v.(type) {
				case float64, string:
					ids = append(ids, fmt.Sprintf("%v", v))
				default:
					r.log.Errorf("events must a number or string, but got %T: skipping", v)
					continue
				}
			}
		case float64, string:
			ids = append(ids, fmt.Sprintf("%v", tresp))
		default:
			r.log.Errorf("cannot collect IDs from type '%T' : '%v'", values, values)
		}
	}
	return ids, nil
}

// processAndPublishEvents process and publish events based on response type
func (r *requester) processAndPublishEvents(stdCtx context.Context, trCtx *transformContext, publisher inputcursor.Publisher, finalResps []*http.Response, publish bool, i int) int {
	events := r.responseProcessors[i].startProcessing(stdCtx, trCtx, finalResps)

	eventsCh, err := r.responseProcessor.startProcessing(stdCtx, trCtx, httpResp)
	if err != nil {
		return err
	}

	trCtx.clearIntervalData()

	var n int
<<<<<<< HEAD
	for maybeMsg := range eventsCh {
=======
	for maybeMsg := range events {
>>>>>>> 2ec07f5eca ([Filebeat][httpjson] httpjson chain calls (#29816))
		if maybeMsg.failed() {
			r.log.Errorf("error processing response: %v", maybeMsg)
			continue
		}

		if publish {
			event, err := makeEvent(maybeMsg.msg)
			if err != nil {
				r.log.Errorf("error creating event: %v", maybeMsg)
				continue
			}

			if err := publisher.Publish(event, trCtx.cursorMap()); err != nil {
				r.log.Errorf("error publishing event: %v", err)
				continue
			}
		}
		if len(*trCtx.firstEventClone()) == 0 {
			trCtx.updateFirstEvent(maybeMsg.msg)
		}
		trCtx.updateLastEvent(maybeMsg.msg)
		trCtx.updateCursor()
		n++
	}
	return n
}
