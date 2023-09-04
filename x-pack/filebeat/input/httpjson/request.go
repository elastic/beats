// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/PaesslerAG/jsonpath"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/mito/lib/xml"
)

const requestNamespace = "request"

type httpClient struct {
	client  *http.Client
	limiter *rateLimiter
}

func (r *requester) doRequest(stdCtx context.Context, trCtx *transformContext, publisher inputcursor.Publisher) error {
	var (
		n                       int
		ids                     []string
		err                     error
		urlCopy                 url.URL
		urlString               string
		httpResp                *http.Response
		initialResponse         []*http.Response
		intermediateResps       []*http.Response
		finalResps              []*http.Response
		isChainWithPageExpected bool
		chainIndex              int
	)

	//nolint:bodyclose // response body is closed through drainBody method
	for i, rf := range r.requestFactories {
		finalResps = nil
		intermediateResps = nil
		// iterate over collected ids from last response
		if i == 0 {
			// perform and store regular call responses
			httpResp, err = rf.collectResponse(stdCtx, trCtx, r)
			if err != nil {
				return fmt.Errorf("failed to execute rf.collectResponse: %w", err)
			}

			if rf.saveFirstResponse {
				// store first response in transform context
				var bodyMap map[string]interface{}
				body, err := io.ReadAll(httpResp.Body)
				if err != nil {
					return fmt.Errorf("failed to read http response body: %w", err)
				}
				httpResp.Body = io.NopCloser(bytes.NewReader(body))
				err = json.Unmarshal(body, &bodyMap)
				if err != nil {
					r.log.Errorf("unable to unmarshal first_response.body: %v", err)
				}
				firstResponse := response{
					url:    *httpResp.Request.URL,
					header: httpResp.Header.Clone(),
					body:   bodyMap,
				}
				trCtx.updateFirstResponse(firstResponse)
			}

			if len(r.requestFactories) == 1 {
				finalResps = append(finalResps, httpResp)
				p := newPublisher(trCtx, publisher, true, r.log)
				events := newStream()
				r.responseProcessors[i].startProcessing(stdCtx, trCtx, finalResps, true, events)
				p.processAndPublishEvents(events)
				n = p.eventCount()
				continue
			}

			// if flow of control reaches here, that means there are more than 1 request factories
			// if a chain step exists, only then we will initialize flags & variables here which are required for chaining
			if r.requestFactories[i+1].isChain {
				chainIndex = i + 1
				resp, err := cloneResponse(httpResp)
				if err != nil {
					return err
				}
				// the response is cloned and added to finalResps here, since the response of the 1st page (whether pagination exists or not), will
				// be sent for further processing to check if any response processors can be applied or not and at the same time update the last_response,
				// first_event & last_event cursor values.
				finalResps = append(finalResps, resp)

				// if a pagination request factory exists at the root level along with a chain step, only then we will initialize flags & variables here
				// which are required for chaining with root level pagination
				if r.responseProcessors[i].pagination.requestFactory != nil {
					isChainWithPageExpected = true
					resp, err := cloneResponse(httpResp)
					if err != nil {
						return err
					}
					initialResponse = append(initialResponse, resp)
				}
			}

			intermediateResps = append(intermediateResps, httpResp)
			ids, err = r.getIdsFromResponses(intermediateResps, r.requestFactories[i+1].replace)
			if err != nil {
				return err
			}
			// we avoid unnecessary pagination here since chaining is present, thus avoiding any unexpected updates to cursor values
			p := newPublisher(trCtx, publisher, false, r.log)
			events := newStream()
			r.responseProcessors[i].startProcessing(stdCtx, trCtx, finalResps, false, events)
			p.processAndPublishEvents(events)
			n = p.eventCount()
		} else {
			if len(ids) == 0 {
				n = 0
				continue
			}
			urlCopy = rf.url
			urlString = rf.url.String()

			// new transform context for every chain step, derived from parent transform context
			var chainTrCtx *transformContext
			if rf.isChain {
				chainTrCtx = trCtx.clone()
			}

			var val string
			var doReplaceWith bool
			var replaceArr []string
			if rf.replaceWith != "" {
				replaceArr = strings.Split(rf.replaceWith, ",")
				val, doReplaceWith, err = fetchValueFromContext(chainTrCtx, strings.TrimSpace(replaceArr[1]))
				if err != nil {
					return err
				}
			}

			// perform request over collected ids
			for _, id := range ids {
				// reformat urls of requestFactory using ids
				rf.url, err = generateNewUrl(rf.replace, urlString, id)
				if err != nil {
					return fmt.Errorf("failed to generate new URL: %w", err)
				}

				// reformat url accordingly if replaceWith clause exists
				if doReplaceWith {
					rf.url, err = generateNewUrl(strings.TrimSpace(replaceArr[0]), rf.url.String(), val)
					if err != nil {
						return fmt.Errorf("failed to generate new URL: %w", err)
					}
				}
				// collect data from new urls
				httpResp, err = rf.collectResponse(stdCtx, chainTrCtx, r)
				if err != nil {
					return fmt.Errorf("failed to execute rf.collectResponse: %w", err)
				}
				// store data according to response type
				if i == len(r.requestFactories)-1 && len(ids) != 0 {
					finalResps = append(finalResps, httpResp)
				} else {
					intermediateResps = append(intermediateResps, httpResp)
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

			p := newPublisher(chainTrCtx, publisher, i < len(r.requestFactories), r.log)
			events := newStream()
			if rf.isChain {
				rf.chainResponseProcessor.startProcessing(stdCtx, chainTrCtx, resps, true, events)
			} else {
				r.responseProcessors[i].startProcessing(stdCtx, trCtx, resps, true, events)
			}
			p.processAndPublishEvents(events)
			n += p.eventCount()
		}
	}

	defer httpResp.Body.Close()
	// if pagination exists for the parent request along with chaining, then for each page response the chain is processed
	if isChainWithPageExpected {
		n += r.processRemainingChainEvents(stdCtx, trCtx, publisher, initialResponse, chainIndex)
	}
	r.log.Infof("request finished: %d events published", n)

	return nil
}

// collectResponse returns response from provided request
func (rf *requestFactory) collectResponse(stdCtx context.Context, trCtx *transformContext, r *requester) (*http.Response, error) {
	var err error
	var httpResp *http.Response

	req, err := rf.newHTTPRequest(stdCtx, trCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	if rf.isChain && rf.chainHTTPClient != nil {
		httpResp, err = rf.chainHTTPClient.do(stdCtx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute chain http client.Do: %w", err)
		}
	} else {
		httpResp, err = r.client.do(stdCtx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to execute http client.Do: %w", err)
		}
	}

	return httpResp, nil
}

func (c *httpClient) do(stdCtx context.Context, req *http.Request) (*http.Response, error) {
	resp, err := c.limiter.execute(stdCtx, func() (*http.Response, error) {
		return c.client.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute http client.Do: %w", err)
	}
	defer resp.Body.Close()

	// Read the whole resp.Body so we can release the connection.
	// This implementation is inspired by httputil.DumpResponse
	resp.Body, err = drainBody(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode > 399 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server responded with status code %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}

type requestFactory struct {
	url                    url.URL
	method                 string
	body                   *mapstr.M
	transforms             []basicTransform
	user                   string
	password               string
	log                    *logp.Logger
	encoder                encoderFunc
	replace                string
	replaceWith            string
	isChain                bool
	until                  *valueTpl
	chainHTTPClient        *httpClient
	chainResponseProcessor *responseProcessor
	saveFirstResponse      bool
}

func newRequestFactory(ctx context.Context, config config, log *logp.Logger, metrics *inputMetrics, reg *monitoring.Registry) ([]*requestFactory, error) {
	// config validation already checked for errors here
	rfs := make([]*requestFactory, 0, len(config.Chain)+1)
	ts, _ := newBasicTransformsFromConfig(registeredTransforms, config.Request.Transforms, requestNamespace, log)
	// regular call requestFactory object
	rf := &requestFactory{
		url:               *config.Request.URL.URL,
		method:            config.Request.Method,
		body:              config.Request.Body,
		transforms:        ts,
		log:               log,
		encoder:           registeredEncoders[config.Request.EncodeAs],
		saveFirstResponse: config.Response.SaveFirstResponse,
	}
	if config.Auth != nil && config.Auth.Basic.isEnabled() {
		rf.user = config.Auth.Basic.User
		rf.password = config.Auth.Basic.Password
	}
	var xmlDetails map[string]xml.Detail
	if config.Response.XSD != "" {
		var err error
		xmlDetails, err = xml.Details([]byte(config.Response.XSD))
		if err != nil {
			log.Errorf("error while collecting xml decoder type hints: %v", err)
			return nil, err
		}
	}
	rfs = append(rfs, rf)
	for _, ch := range config.Chain {
		var rf *requestFactory
		// chain calls requestFactory object
		if ch.Step != nil {
			ts, _ := newBasicTransformsFromConfig(registeredTransforms, ch.Step.Request.Transforms, requestNamespace, log)
			ch.Step.Auth = tryAssignAuth(config.Auth, ch.Step.Auth)
			httpClient, err := newChainHTTPClient(ctx, ch.Step.Auth, ch.Step.Request, log, reg)
			if err != nil {
				return nil, fmt.Errorf("failed in creating chain http client with error : %w", err)
			}
			if ch.Step.Auth != nil && ch.Step.Auth.Basic.isEnabled() {
				rf.user = ch.Step.Auth.Basic.User
				rf.password = ch.Step.Auth.Basic.Password
			}

			responseProcessor := newChainResponseProcessor(ch, httpClient, xmlDetails, metrics, log)

			rf = &requestFactory{
				url:                    *ch.Step.Request.URL.URL,
				method:                 ch.Step.Request.Method,
				body:                   ch.Step.Request.Body,
				transforms:             ts,
				log:                    log,
				encoder:                registeredEncoders[config.Request.EncodeAs],
				replace:                ch.Step.Replace,
				replaceWith:            ch.Step.ReplaceWith,
				isChain:                true,
				chainHTTPClient:        httpClient,
				chainResponseProcessor: responseProcessor,
			}
		} else if ch.While != nil {
			ts, _ := newBasicTransformsFromConfig(registeredTransforms, ch.While.Request.Transforms, requestNamespace, log)
			policy := newHTTPPolicy(evaluateResponse, ch.While.Until, log)
			ch.While.Auth = tryAssignAuth(config.Auth, ch.While.Auth)
			httpClient, err := newChainHTTPClient(ctx, ch.While.Auth, ch.While.Request, log, reg, policy)
			if err != nil {
				return nil, fmt.Errorf("failed in creating chain http client with error : %w", err)
			}
			if ch.While.Auth != nil && ch.While.Auth.Basic.isEnabled() {
				rf.user = ch.While.Auth.Basic.User
				rf.password = ch.While.Auth.Basic.Password
			}

			responseProcessor := newChainResponseProcessor(ch, httpClient, xmlDetails, metrics, log)
			rf = &requestFactory{
				url:                    *ch.While.Request.URL.URL,
				method:                 ch.While.Request.Method,
				body:                   ch.While.Request.Body,
				transforms:             ts,
				log:                    log,
				encoder:                registeredEncoders[config.Request.EncodeAs],
				replace:                ch.While.Replace,
				replaceWith:            ch.While.ReplaceWith,
				until:                  ch.While.Until,
				isChain:                true,
				chainHTTPClient:        httpClient,
				chainResponseProcessor: responseProcessor,
			}
		}
		rfs = append(rfs, rf)
	}
	return rfs, nil
}

func evaluateResponse(expression *valueTpl, data []byte, log *logp.Logger) (bool, error) {
	var dataMap mapstr.M

	err := json.Unmarshal(data, &dataMap)
	if err != nil {
		return false, fmt.Errorf("error while unmarshalling data : %w", err)
	}
	tr := transformable{}
	paramCtx := &transformContext{
		firstEvent:    &mapstr.M{},
		lastEvent:     &mapstr.M{},
		firstResponse: &response{},
		lastResponse:  &response{body: dataMap},
	}

	val, err := expression.Execute(paramCtx, tr, "", nil, log)
	if err != nil {
		return false, fmt.Errorf("error while evaluating expression : %w", err)
	}
	result, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("error while parsing boolean value of string : %w", err)
	}

	return result, nil
}

func tryAssignAuth(parentConfig *authConfig, childConfig *authConfig) *authConfig {
	if parentConfig != nil && childConfig == nil {
		return parentConfig
	}
	return childConfig
}

func (rf *requestFactory) newHTTPRequest(stdCtx context.Context, trCtx *transformContext) (*http.Request, error) {
	trReq, err := rf.newRequest(trCtx)
	if err != nil {
		return nil, err
	}

	var body []byte
	if rf.method == http.MethodPost {
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

	if rf.method == http.MethodPost {
		header = req.header()
		if header.Get("Content-Type") == "" {
			header.Set("Content-Type", "application/json")
			req.setHeader(header)
		}
	}

	rf.log.Debugf("new request: %#v", req)

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
	log *logp.Logger,
) *requester {
	return &requester{
		log:                log,
		client:             client,
		requestFactories:   requestFactory,
		responseProcessors: responseProcessor,
	}
}

// getIdsFromResponses returns ids from responses
func (r *requester) getIdsFromResponses(intermediateResps []*http.Response, replace string) ([]string, error) {
	var b []byte
	var ids []string
	var err error
	// collect ids from all responses
	for _, resp := range intermediateResps {
		if resp.Body != nil {
			b, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("error while reading response body: %w", err)
			}
		}
		// gracefully close response
		err = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error closing response body: %w", err)
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

// processRemainingChainEvents, processes the remaining pagination events for chain blocks
func (r *requester) processRemainingChainEvents(stdCtx context.Context, trCtx *transformContext, publisher inputcursor.Publisher, initialResp []*http.Response, chainIndex int) int {
	// we start from 0, and skip the 1st event since we have already processed it
	p := newChainProcessor(r, trCtx, publisher, chainIndex)
	events := newStream()
	r.responseProcessors[0].startProcessing(stdCtx, trCtx, initialResp, true, events)
	p.processRemainingChainEvents(stdCtx, events)
	return p.eventCount()
}

type chainProcessor struct {
	req  *requester
	ctx  *transformContext
	pub  inputcursor.Publisher
	idx  int
	tail bool
	n    int
}

func newChainProcessor(req *requester, trCtx *transformContext, pub inputcursor.Publisher, idx int) *chainProcessor {
	return &chainProcessor{
		req: req,
		ctx: trCtx,
		pub: pub,
		idx: idx,
	}
}

func (p *chainProcessor) processRemainingChainEvents(ctx context.Context, events stream) {
	for maybeMsg := range events.ch {
		if maybeMsg.failed() {
			p.fail(maybeMsg.err)
			continue
		}

		p.event(ctx, maybeMsg.msg)
	}
}

func (p *chainProcessor) event(ctx context.Context, msg mapstr.M) {
	if !p.tail {
		// Skip first event as it has already been processed.
		p.tail = true
		return
	}

	var response http.Response
	response.StatusCode = 200
	body := new(bytes.Buffer)
	// we construct a new response here from each of the pagination events
	err := json.NewEncoder(body).Encode(msg)
	if err != nil {
		p.req.log.Errorf("error processing chain event: %w", err)
		return
	}
	response.Body = io.NopCloser(body)

	// updates the cursor for pagination last_event & last_response when chaining is present
	p.ctx.updateLastEvent(msg)
	p.ctx.updateCursor()

	// for each pagination response, we repeat all the chain steps / blocks
	n, err := p.req.processChainPaginationEvents(ctx, p.ctx, p.pub, &response, p.idx, p.req.log)
	if err != nil {
		p.req.log.Errorf("error processing chain event: %w", err)
		return
	}
	p.n += n

	err = response.Body.Close()
	if err != nil {
		p.req.log.Errorf("error closing http response body: %w", err)
	}
}

func (p *chainProcessor) fail(err error) {
	p.req.log.Errorf("error processing response: %v", err)
}

func (p *chainProcessor) eventCount() int {
	return p.n
}

func (*chainProcessor) close() {}

// processChainPaginationEvents takes a pagination response as input and runs all the chain blocks for the input
//
//nolint:bodyclose // response body is closed through drainBody method
func (r *requester) processChainPaginationEvents(stdCtx context.Context, trCtx *transformContext, publisher inputcursor.Publisher, response *http.Response, chainIndex int, log *logp.Logger) (int, error) {
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

	intermediateResps = append(intermediateResps, response)
	ids, err = r.getIdsFromResponses(intermediateResps, r.requestFactories[chainIndex].replace)
	if err != nil {
		return -1, err
	}

	for i := chainIndex; i < len(r.requestFactories); i++ {
		finalResps = nil
		intermediateResps = nil
		rf := r.requestFactories[i]

		if len(ids) == 0 {
			n = 0
			continue
		}
		urlCopy = rf.url
		urlString = rf.url.String()

		// new transform context for every chain step, derived from parent transform context
		chainTrCtx := trCtx.clone()

		var val string
		var doReplaceWith bool
		var replaceArr []string
		if rf.replaceWith != "" {
			replaceArr = strings.Split(rf.replaceWith, ",")
			val, doReplaceWith, err = fetchValueFromContext(chainTrCtx, strings.TrimSpace(replaceArr[1]))
			if err != nil {
				return n, err
			}
		}

		// perform request over collected ids
		for _, id := range ids {
			// reformat urls of requestFactory using ids
			rf.url, err = generateNewUrl(rf.replace, urlString, id)
			if err != nil {
				return -1, fmt.Errorf("failed to generate new URL: %w", err)
			}

			// reformat url accordingly if replaceWith clause exists
			if doReplaceWith {
				rf.url, err = generateNewUrl(strings.TrimSpace(replaceArr[0]), rf.url.String(), val)
				if err != nil {
					return n, fmt.Errorf("failed to generate new URL: %w", err)
				}
			}

			// collect data from new urls
			httpResp, err = rf.collectResponse(stdCtx, chainTrCtx, r)
			if err != nil {
				return -1, fmt.Errorf("failed to execute rf.collectResponse: %w", err)
			}
			// store data according to response type
			if i == len(r.requestFactories)-1 && len(ids) != 0 {
				finalResps = append(finalResps, httpResp)
			} else {
				intermediateResps = append(intermediateResps, httpResp)
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
				return -1, err
			}
			resps = intermediateResps
		}
		p := newPublisher(chainTrCtx, publisher, i < len(r.requestFactories), r.log)
		events := newStream()
		rf.chainResponseProcessor.startProcessing(stdCtx, chainTrCtx, resps, true, events)
		p.processAndPublishEvents(events)
		n += p.eventCount()
	}

	defer func() {
		if httpResp != nil && httpResp.Body != nil {
			httpResp.Body.Close()
		}
	}()

	return n, nil
}

// generateNewUrl returns new url value using replacement from oldUrl with ids
func generateNewUrl(replacement, oldUrl, id string) (url.URL, error) {
	newUrl, err := url.Parse(strings.Replace(oldUrl, replacement, id, 1))
	if err != nil {
		return url.URL{}, fmt.Errorf("failed to replace value in url: %w", err)
	}
	return *newUrl, nil
}

type publisher struct {
	ctx *transformContext
	pub inputcursor.Publisher
	n   int
	log *logp.Logger
}

func newPublisher(trCtx *transformContext, pub inputcursor.Publisher, publish bool, log *logp.Logger) *publisher {
	if !publish {
		pub = nil
	}
	return &publisher{
		ctx: trCtx,
		pub: pub,
		log: log,
	}
}

// processAndPublishEvents process and publish events based on event type
func (p *publisher) processAndPublishEvents(events stream) {
	for maybeMsg := range events.ch {
		p.processAndPublishEvent(maybeMsg)
	}
}

// processAndPublishEvent processes and publishes one events based on event type
func (p *publisher) processAndPublishEvent(evt maybeMsg) {
	if evt.failed() {
		p.fail(evt.err)
		return
	}
	p.event(nil, evt.msg)
}

func (p *publisher) event(_ context.Context, msg mapstr.M) {
	if p.pub != nil {
		event, err := makeEvent(msg)
		if err != nil {
			p.log.Errorf("error creating event: %v: %v", msg, err)
			return
		}

		if err := p.pub.Publish(event, p.ctx.cursorMap()); err != nil {
			p.log.Errorf("error publishing event: %v", err)
			return
		}
	}
	if len(*p.ctx.firstEventClone()) == 0 {
		p.ctx.updateFirstEvent(msg)
	}
	p.ctx.updateLastEvent(msg)
	p.ctx.updateCursor()

	p.n++
}

func (p *publisher) fail(err error) {
	p.log.Errorf("error processing response: %v", err)
}

func (p *publisher) eventCount() int {
	return p.n
}

func (p *publisher) close() {}

const (
	// This is generally updated with chain responses, if present, as they continue to occur
	// Otherwise this is always the last response of the root request w.r.t pagination
	lastResponse = "last_response"
	// This is always the first root response
	firstResponse = "first_response"
	// This is always the last response of the parent (root) request w.r.t pagination
	// This is only set if chaining is used
	parentLastResponse = "parent_last_response"
)

func fetchValueFromContext(trCtx *transformContext, expression string) (string, bool, error) {
	var val interface{}

	switch keys := processExpression(expression); keys[0] {
	case lastResponse:
		respMap, err := responseToMap(trCtx.lastResponse)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	case parentLastResponse:
		respMap, err := responseToMap(trCtx.parentTrCtx.lastResponse)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	case firstResponse:
		// since first response body is already a map, we do not need to transform it
		respMap, err := responseToMap(trCtx.firstResponse)
		if err != nil {
			return "", false, err
		}
		val, err = iterateRecursive(respMap, keys[1:], 0)
		if err != nil {
			return "", false, err
		}
	// In this scenario we treat the expression as a hardcoded value, with which we will replace the fixed-pattern
	case expression:
		return expression, true, nil
	default:
		return "", false, fmt.Errorf("context value not supported for key: %q in expression %q", keys[0], expression)
	}

	return fmt.Sprint(val), true, nil
}

// processExpression, splits the expression string based on the separator and looks for
// supported keywords. If present, returns an expression array containing separated elements.
// If no keywords are present, the expression is treated as a hardcoded value and returned
// as a merged string which is the only array element.
func processExpression(expression string) []string {
	if !strings.HasPrefix(expression, ".") {
		return []string{expression}
	}
	switch {
	case strings.HasPrefix(expression, "."+firstResponse+"."),
		strings.HasPrefix(expression, "."+lastResponse+"."),
		strings.HasPrefix(expression, "."+parentLastResponse+"."):
		return strings.Split(expression, ".")[1:]
	default:
		return []string{expression}
	}
}

func responseToMap(r *response) (mapstr.M, error) {
	if r.body == nil {
		return nil, fmt.Errorf("response body is empty for request url: %s", &r.url)
	}
	respMap := map[string]interface{}{
		"header": make(mapstr.M),
		"body":   make(mapstr.M),
	}

	for key, value := range r.header {
		respMap["header"] = mapstr.M{
			key: value,
		}
	}
	respMap["body"] = r.body

	return respMap, nil
}

func iterateRecursive(m mapstr.M, keys []string, depth int) (interface{}, error) {
	val := m[keys[depth]]

	if val == nil {
		return nil, fmt.Errorf("value of expression could not be determined for key %s", strings.Join(keys[:depth+1], "."))
	}

	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		return v.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return v.Float(), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Map:
		nextMap, ok := v.Interface().(map[string]interface{})
		if !ok {
			return nil, errors.New("unable to parse the value of the given expression")
		}
		depth++
		if depth >= len(keys) {
			return nil, errors.New("value of expression could not be determined")
		}
		return iterateRecursive(nextMap, keys, depth)
	default:
		return nil, fmt.Errorf("unable to parse the value of the expression %s: type %T is not handled", strings.Join(keys[:depth+1], "."), val)
	}
}

// cloneResponse clones required http response attributes
func cloneResponse(source *http.Response) (*http.Response, error) {
	var resp http.Response

	body, err := io.ReadAll(source.Body)
	if err != nil {
		return nil, fmt.Errorf("failed ro read http response body: %w", err)
	}

	source.Body = io.NopCloser(bytes.NewReader(body))
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = source.ContentLength
	resp.Header = source.Header
	resp.Trailer = source.Trailer
	resp.StatusCode = source.StatusCode
	resp.Request = source.Request.Clone(source.Request.Context())

	return &resp, nil
}

// drainBody reads all of b to memory and then returns a equivalent
// ReadCloser yielding the same bytes.
//
// It returns an error if the initial slurp of all bytes fails. It does not attempt
// to make the returned ReadCloser have identical error-matching behavior.
//
// This function is a modified version of drainBody from the http/httputil package.
func drainBody(b io.ReadCloser) (r1 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return http.NoBody, nil
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(b); err != nil {
		return b, err
	}
	if err = b.Close(); err != nil {
		return b, err
	}

	return io.NopCloser(&buf), nil
}
