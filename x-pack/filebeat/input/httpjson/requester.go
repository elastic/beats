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
	"io/ioutil"
	"net/http"
	"strings"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type requestInfo struct {
	url        string
	contentMap common.MapStr
	headers    common.MapStr
}

type requester struct {
	log         *logp.Logger
	client      *http.Client
	dateCursor  *dateCursor
	rateLimiter *rateLimiter
	pagination  *pagination

	method        string
	reqBody       common.MapStr
	headers       common.MapStr
	noHTTPBody    bool
	apiKey        string
	authScheme    string
	jsonObjects   string
	splitEventsBy string

	cursorState cursorState
}

func newRequester(
	config config,
	rateLimiter *rateLimiter,
	dateCursor *dateCursor,
	pagination *pagination,
	client *http.Client,
	log *logp.Logger) *requester {
	return &requester{
		log:           log,
		client:        client,
		rateLimiter:   rateLimiter,
		dateCursor:    dateCursor,
		pagination:    pagination,
		method:        config.HTTPMethod,
		reqBody:       config.HTTPRequestBody.Clone(),
		headers:       config.HTTPHeaders.Clone(),
		noHTTPBody:    config.NoHTTPBody,
		apiKey:        config.APIKey,
		authScheme:    config.AuthenticationScheme,
		splitEventsBy: config.SplitEventsBy,
		jsonObjects:   config.JSONObjects,
	}
}

type response struct {
	header http.Header
	body   common.MapStr
}

// processHTTPRequest processes HTTP request, and handles pagination if enabled
func (r *requester) processHTTPRequest(ctx context.Context, publisher cursor.Publisher) error {
	ri := &requestInfo{
		url:        r.dateCursor.getURL(r.cursorState.LastDateCursorValue),
		contentMap: common.MapStr{},
		headers:    r.headers,
	}

	if r.method == "POST" && r.reqBody != nil {
		ri.contentMap.Update(r.reqBody)
	}

	var (
		m, v     interface{}
		response response
		lastObj  common.MapStr
	)

	// always request at least once
	hasNext := true

	for hasNext {
		resp, err := r.rateLimiter.execute(
			ctx,
			func(ctx context.Context) (*http.Response, error) {
				req, err := r.createHTTPRequest(ctx, ri)
				if err != nil {
					return nil, fmt.Errorf("failed to create http request: %w", err)
				}
				msg, err := r.client.Do(req)
				if err != nil {
					return nil, fmt.Errorf("failed to execute http client.Do: %w", err)
				}
				return msg, nil
			},
		)
		if err != nil {
			return err
		}

		response.header = resp.Header
		responseData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read http response: %w", err)
		}
		_ = resp.Body.Close()

		if err = json.Unmarshal(responseData, &m); err != nil {
			r.log.Debug("failed to unmarshal http.response.body", string(responseData))
			return fmt.Errorf("failed to unmarshal http.response.body: %w", err)
		}

		switch obj := m.(type) {
		// Top level Array
		case []interface{}:
			lastObj, err = r.processEventArray(publisher, obj)
			if err != nil {
				return err
			}
		case map[string]interface{}:
			response.body = obj
			if r.jsonObjects == "" {
				lastObj, err = r.processEventArray(publisher, []interface{}{obj})
				if err != nil {
					return err
				}
			} else {
				v, err = common.MapStr(obj).GetValue(r.jsonObjects)
				if err != nil {
					if errors.Is(err, common.ErrKeyNotFound) {
						break
					}
					return err
				}
				switch ts := v.(type) {
				case []interface{}:
					lastObj, err = r.processEventArray(publisher, ts)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("content of %s is not a valid array", r.jsonObjects)
				}
			}
		default:
			r.log.Debug("http.response.body is not a valid JSON object", string(responseData))
			return fmt.Errorf("http.response.body is not a valid JSON object, but a %T", obj)
		}

		ri, hasNext, err = r.pagination.nextRequestInfo(ri, response, lastObj)
		if err != nil {
			return err
		}

		if lastObj != nil && r.dateCursor.enabled {
			r.updateCursorState(ri.url, r.dateCursor.getNextValue(lastObj))
		}
	}

	return nil
}

// createHTTPRequest creates an HTTP/HTTPs request for the input
func (r *requester) createHTTPRequest(ctx context.Context, ri *requestInfo) (*http.Request, error) {
	var body io.Reader
	if len(ri.contentMap) == 0 || r.noHTTPBody {
		body = nil
	} else {
		b, err := json.Marshal(ri.contentMap)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(r.method, ri.url, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		if r.authScheme != "" {
			req.Header.Set("Authorization", r.authScheme+" "+r.apiKey)
		} else {
			req.Header.Set("Authorization", r.apiKey)
		}
	}
	for k, v := range ri.headers {
		switch vv := v.(type) {
		case string:
			req.Header.Set(k, vv)
		default:
		}
	}
	return req, nil
}

// processEventArray publishes an event for each object contained in the array. It returns the last object in the array and an error if any.
func (r *requester) processEventArray(publisher cursor.Publisher, events []interface{}) (map[string]interface{}, error) {
	var last map[string]interface{}
	for _, t := range events {
		switch v := t.(type) {
		case map[string]interface{}:
			for _, e := range splitEvent(r.splitEventsBy, v) {
				last = e
				d, err := json.Marshal(e)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal %+v: %w", e, err)
				}
				if err := publisher.Publish(makeEvent(string(d)), r.cursorState); err != nil {
					return nil, fmt.Errorf("failed to publish: %w", err)
				}
			}
		default:
			return nil, fmt.Errorf("expected only JSON objects in the array but got a %T", v)
		}
	}
	return last, nil
}

func splitEvent(splitKey string, event map[string]interface{}) []map[string]interface{} {
	m := common.MapStr(event)

	// NOTE: this notation is only used internally, not meant to be documented
	// and will be removed in the next release
	keys := strings.SplitN(splitKey, "..", 2)
	if len(keys) < 2 {
		// we append an empty key to force the recursive call
		keys = append(keys, "")
	}

	hasSplitKey, _ := m.HasKey(keys[0])
	if keys[0] == "" || !hasSplitKey {
		return []map[string]interface{}{event}
	}

	splitOnIfc, _ := m.GetValue(keys[0])
	splitOn, ok := splitOnIfc.([]interface{})
	// if not an array or is empty, we do nothing
	if !ok || len(splitOn) == 0 {
		return []map[string]interface{}{event}
	}

	var events []map[string]interface{}
	for _, split := range splitOn {
		s, ok := split.(map[string]interface{})
		// if not an object, we do nothing
		if !ok {
			return []map[string]interface{}{event}
		}

		// call splitEvent recursively for each part
		for _, nestedSplit := range splitEvent(keys[1], s) {
			mm := m.Clone()
			if _, err := mm.Put(keys[0], nestedSplit); err != nil {
				return []map[string]interface{}{event}
			}
			events = append(events, mm)
		}
	}

	return events
}

type cursorState struct {
	LastCalledURL       string
	LastDateCursorValue string
}

func (r *requester) updateCursorState(url, value string) {
	r.cursorState.LastCalledURL = url
	r.cursorState.LastDateCursorValue = value
}

func (r *requester) loadCursor(c *cursor.Cursor, log *logp.Logger) {
	if c == nil || c.IsNew() {
		return
	}

	if err := c.Unpack(&r.cursorState); err != nil {
		log.Errorf("Reset http cursor state. Failed to read from registry: %v", err)
	}
}
