// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PaesslerAG/jsonpath"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	isChain                bool
	until                  *valueTpl
	chainHTTPClient        *httpClient
	chainResponseProcessor *responseProcessor
}

func newRequestFactory(ctx context.Context, config config, log *logp.Logger) ([]*requestFactory, error) {
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
		var rf *requestFactory
		// chain calls requestFactory object
		if ch.Step != nil {
			ts, _ := newBasicTransformsFromConfig(ch.Step.Request.Transforms, requestNamespace, log)
			ch.Step.Auth = tryAssignAuth(config.Auth, ch.Step.Auth)
			httpClient, err := newChainHTTPClient(ctx, ch.Step.Auth, ch.Step.Request, log)
			if err != nil {
				return nil, fmt.Errorf("failed in creating chain http client with error : %w", err)
			}
			if ch.Step.Auth != nil && ch.Step.Auth.Basic.isEnabled() {
				rf.user = ch.Step.Auth.Basic.User
				rf.password = ch.Step.Auth.Basic.Password
			}
			pagination := newChainPagination(ch, httpClient, log)
			responseProcessor := newChainResponseProcessor(ch, pagination, log)
			rf = &requestFactory{
				url:                    *ch.Step.Request.URL.URL,
				method:                 ch.Step.Request.Method,
				body:                   ch.Step.Request.Body,
				transforms:             ts,
				log:                    log,
				encoder:                registeredEncoders[config.Request.EncodeAs],
				replace:                ch.Step.Replace,
				isChain:                true,
				chainHTTPClient:        httpClient,
				chainResponseProcessor: responseProcessor,
			}
		} else if ch.While != nil {
			ts, _ := newBasicTransformsFromConfig(ch.While.Request.Transforms, requestNamespace, log)
			policy := newHTTPPolicy(evaluateResponse, ch.While.Until, log)
			ch.While.Auth = tryAssignAuth(config.Auth, ch.While.Auth)
			httpClient, err := newChainHTTPClient(ctx, ch.While.Auth, ch.While.Request, log, policy)
			if err != nil {
				return nil, fmt.Errorf("failed in creating chain http client with error : %w", err)
			}
			if ch.While.Auth != nil && ch.While.Auth.Basic.isEnabled() {
				rf.user = ch.While.Auth.Basic.User
				rf.password = ch.While.Auth.Basic.Password
			}
			pagination := newChainPagination(ch, httpClient, log)
			responseProcessor := newChainResponseProcessor(ch, pagination, log)
			rf = &requestFactory{
				url:                    *ch.While.Request.URL.URL,
				method:                 ch.While.Request.Method,
				body:                   ch.While.Request.Body,
				transforms:             ts,
				log:                    log,
				encoder:                registeredEncoders[config.Request.EncodeAs],
				replace:                ch.While.Replace,
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
		initialResponse   []*http.Response
		intermediateResps []*http.Response
		finalResps        []*http.Response
		isChainExpected   bool
		chainIndex        int
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
			if r.requestFactories[i+1].isChain {
				isChainExpected = true
				chainIndex = i + 1
				resp, err := cloneResponse(httpResp) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
				if err != nil {
					return err
				}
				initialResponse = append(initialResponse, resp) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
			}
			intermediateResps = append(intermediateResps, httpResp) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
			ids, err = r.getIdsFromResponses(intermediateResps, r.requestFactories[i+1].replace)
			if err != nil {
				return err
			}
			if !isChainExpected {
				n = r.processAndPublishEvents(stdCtx, trCtx, publisher, intermediateResps, false, i)
			}
		} else {
			if len(ids) == 0 {
				n = 0
				continue
			}
			urlCopy = rf.url
			urlString = rf.url.String()

			// new transform context for every chain step , derived from parent transform context
			var chainTrCtx *transformContext
			if rf.isChain {
				chainTrCtx = emptyTransformContext()
				chainTrCtx.cursor = trCtx.cursor
			}

			// perform request over collected ids
			for _, id := range ids {
				// reformat urls of requestFactory using ids
				rf.url, err = generateNewUrl(rf.replace, urlString, id)
				if err != nil {
					return fmt.Errorf("failed to generate new URL: %w", err)
				}

				// collect data from new urls
				httpResp, err = rf.collectResponse(stdCtx, chainTrCtx, r) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
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
			if rf.isChain {
				n += processAndPublishChainEvents(stdCtx, chainTrCtx, rf.chainResponseProcessor, publisher, resps, i < len(r.requestFactories), r.log)
			} else {
				n += r.processAndPublishEvents(stdCtx, trCtx, publisher, resps, i < len(r.requestFactories), i)
			}
		}
	}

	defer httpResp.Body.Close()

	if isChainExpected {
		n += r.processRemainingChainEvents(stdCtx, trCtx, publisher, initialResponse, chainIndex)
	}
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
		if resp.Body != nil {
			b, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("error while reading response body: %w", err)
			}
		}
		// gracefully close response and retain response data
		err = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error closing response body : %w", err)
		}
		// resp.Body = io.NopCloser(bytes.NewBuffer(b))

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

	var n int
	for maybeMsg := range events {
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

// processAndPublishEvents processes and publishes chain events based on response type
func processAndPublishChainEvents(stdCtx context.Context, trCtx *transformContext, chainRsp *responseProcessor, publisher inputcursor.Publisher, finalResps []*http.Response, publish bool, log *logp.Logger) int {
	events := chainRsp.startProcessing(stdCtx, trCtx, finalResps)

	var n int
	for maybeMsg := range events {
		if maybeMsg.failed() {
			log.Errorf("error processing response: %v", maybeMsg)
			continue
		}

		if publish {
			event, err := makeEvent(maybeMsg.msg)
			if err != nil {
				log.Errorf("error creating event: %v", maybeMsg)
				continue
			}

			if err := publisher.Publish(event, trCtx.cursorMap()); err != nil {
				log.Errorf("error publishing event: %v", err)
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

func (r *requester) processRemainingChainEvents(stdCtx context.Context, trCtx *transformContext, publisher inputcursor.Publisher, initialResp []*http.Response, chainIndex int) int {
	events := r.responseProcessors[0].startProcessing(stdCtx, trCtx, initialResp)

	var n int
	var eventCount int
	for maybeMsg := range events {
		if maybeMsg.failed() {
			r.log.Errorf("error processing response: %v", maybeMsg)
			continue
		}

		if n >= 1 { // skip 1st event as it has already ben processed before
			var response http.Response
			response.StatusCode = 200
			body := new(bytes.Buffer)

			err := json.NewEncoder(body).Encode(maybeMsg.msg)
			if err != nil {
				r.log.Errorf("error processing chain event: %w", err)
				continue
			}
			response.Body = io.NopCloser(body)

			count, err := r.processChainPaginationEvents(stdCtx, trCtx, publisher, &response, chainIndex, r.log)
			if err != nil {
				r.log.Errorf("error processing chain event: %w", err)
				continue
			}
			eventCount += count

			if len(*trCtx.firstEventClone()) == 0 {
				trCtx.updateFirstEvent(maybeMsg.msg)
			}
			trCtx.updateLastEvent(maybeMsg.msg)
			trCtx.updateCursor()

			err = response.Body.Close()
			if err != nil {
				r.log.Errorf("error closing http response body : %w", err)
			}
		}

		n++
	}
	return eventCount
}

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

	intermediateResps = append(intermediateResps, response) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
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

		// new transform context for every chain step , derived from parent transform context
		var chainTrCtx *transformContext
		if rf.isChain {
			chainTrCtx = emptyTransformContext()
			chainTrCtx.cursor = trCtx.cursor
		}

		// perform request over collected ids
		for _, id := range ids {
			// reformat urls of requestFactory using ids
			rf.url, err = generateNewUrl(rf.replace, urlString, id)
			if err != nil {
				return -1, fmt.Errorf("failed to generate new URL: %w", err)
			}

			// collect data from new urls
			httpResp, err = rf.collectResponse(stdCtx, chainTrCtx, r) //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.
			if err != nil {
				return -1, fmt.Errorf("failed to execute rf.collectResponse: %w", err)
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
				return -1, err
			}
			resps = intermediateResps
		}
		n += processAndPublishChainEvents(stdCtx, chainTrCtx, rf.chainResponseProcessor, publisher, resps, i < len(r.requestFactories), r.log)
	}

	defer httpResp.Body.Close()
	return n, nil
}

func cloneResponse(source *http.Response) (*http.Response, error) {
	var resp http.Response //nolint:bodyclose // Bad linter! The response body will always be closed by drainBody function.

	body, err := io.ReadAll(source.Body)
	if err != nil {
		return nil, fmt.Errorf("failed ro read http response body: %w", err)
	}

	// clones response
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
