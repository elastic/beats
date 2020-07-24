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
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type requestInfo struct {
	url        string
	contentMap common.MapStr
	headers    common.MapStr
}

type retryLogger struct {
	log *logp.Logger
}

func newRetryLogger() *retryLogger {
	return &retryLogger{
		log: logp.NewLogger("httpjson.retryablehttp", zap.AddCallerSkip(1)),
	}
}

func (l *retryLogger) Printf(s string, args ...interface{}) {
	l.log.Debugf(s, args...)
}

type requester struct {
	log         *logp.Logger
	config      config
	client      *http.Client
	cursorValue string
}

func (r requester) Name() string { return r.config.URL }

// createHTTPRequest creates an HTTP/HTTPs request for the input
func (r *requester) createHTTPRequest(ctx context.Context, ri *requestInfo) (*http.Request, error) {
	var body io.Reader
	if len(ri.contentMap) == 0 || r.config.NoHTTPBody {
		body = nil
	} else {
		b, err := json.Marshal(ri.contentMap)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(r.config.HTTPMethod, ri.url, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if r.config.APIKey != "" {
		if r.config.AuthenticationScheme != "" {
			req.Header.Set("Authorization", r.config.AuthenticationScheme+" "+r.config.APIKey)
		} else {
			req.Header.Set("Authorization", r.config.APIKey)
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
			for _, e := range r.splitEvent(v) {
				last = e
				d, err := json.Marshal(e)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to marshal %+v", e)
				}
				if err := publisher.Publish(makeEvent(string(d)), r.cursorValue); err != nil {
					return nil, err
				}
			}
		default:
			return nil, errors.Errorf("expected only JSON objects in the array but got a %T", v)
		}
	}
	return last, nil
}

func (r *requester) splitEvent(event map[string]interface{}) []map[string]interface{} {
	m := common.MapStr(event)

	hasSplitKey, _ := m.HasKey(r.config.SplitEventsBy)
	if r.config.SplitEventsBy == "" || !hasSplitKey {
		return []map[string]interface{}{event}
	}

	splitOnIfc, _ := m.GetValue(r.config.SplitEventsBy)
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

		mm := m.Clone()
		_, err := mm.Put(r.config.SplitEventsBy, s)
		if err != nil {
			return []map[string]interface{}{event}
		}

		events = append(events, mm)
	}

	return events
}

// getNextLinkFromHeader retrieves the next URL for pagination from the HTTP Header of the response
func getNextLinkFromHeader(header http.Header, fieldName string, re *regexp.Regexp) (string, error) {
	links, ok := header[fieldName]
	if !ok {
		return "", errors.Errorf("field %s does not exist in the HTTP Header", fieldName)
	}
	for _, link := range links {
		matchArray := re.FindAllStringSubmatch(link, -1)
		if len(matchArray) == 1 {
			return matchArray[0][1], nil
		}
	}
	return "", nil
}

// getRateLimit get the rate limit value if specified in the HTTP Header of the response,
// and returns an init64 value in seconds since unix epoch for rate limit reset time.
// When there is a remaining rate limit quota, or when the rate limit reset time has expired, it
// returns 0 for the epoch value.
func getRateLimit(header http.Header, rateLimit *RateLimit) (int64, error) {
	if rateLimit != nil {
		if rateLimit.Remaining != "" {
			remaining := header.Get(rateLimit.Remaining)
			if remaining == "" {
				return 0, errors.Errorf("field %s does not exist in the HTTP Header, or is empty", rateLimit.Remaining)
			}
			m, err := strconv.ParseInt(remaining, 10, 64)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to parse rate-limit remaining value")
			}
			if m == 0 {
				reset := header.Get(rateLimit.Reset)
				if reset == "" {
					return 0, errors.Errorf("field %s does not exist in the HTTP Header, or is empty", rateLimit.Reset)
				}
				epoch, err := strconv.ParseInt(reset, 10, 64)
				if err != nil {
					return 0, errors.Wrapf(err, "failed to parse rate-limit reset value")
				}
				if time.Unix(epoch, 0).Sub(time.Now()) <= 0 {
					return 0, nil
				}
				return epoch, nil
			}
		}
	}
	return 0, nil
}

// applyRateLimit applies appropriate rate limit if specified in the HTTP Header of the response
func (r *requester) applyRateLimit(ctx context.Context, header http.Header, rateLimit *RateLimit) error {
	epoch, err := getRateLimit(header, rateLimit)
	if err != nil {
		return err
	}
	t := time.Unix(epoch, 0)
	w := time.Until(t)
	if epoch == 0 || w <= 0 {
		r.log.Debugf("Rate Limit: No need to apply rate limit.")
		return nil
	}
	r.log.Debugf("Rate Limit: Wait until %v for the rate limit to reset.", t)
	ticker := time.NewTicker(w)
	defer ticker.Stop()
	select {
	case <-ctx.Done():
		r.log.Info("Context done.")
		return nil
	case <-ticker.C:
		r.log.Debug("Rate Limit: time is up.")
		return nil
	}
}

// createRequestInfoFromBody creates a new RequestInfo for a new HTTP request in pagination based on HTTP response body
func createRequestInfoFromBody(config *Pagination, response, last common.MapStr, ri *requestInfo) (*requestInfo, error) {
	// we try to get it from last element, if not found, from the original response
	v, err := last.GetValue(config.IDField)
	if err == common.ErrKeyNotFound {
		v, err = response.GetValue(config.IDField)
	}

	if err == common.ErrKeyNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve id_field for pagination")
	}

	if config.RequestField != "" {
		ri.contentMap.Put(config.RequestField, v)
		if config.URL != "" {
			ri.url = config.URL
		}
	} else if config.URLField != "" {
		url, err := url.Parse(ri.url)
		if err == nil {
			q := url.Query()
			q.Set(config.URLField, fmt.Sprint(v))
			url.RawQuery = q.Encode()
			ri.url = url.String()
		}
	} else {
		switch vt := v.(type) {
		case string:
			ri.url = vt
		default:
			return nil, errors.New("pagination ID is not of string type")
		}
	}
	if len(config.ExtraBodyContent) > 0 {
		ri.contentMap.Update(common.MapStr(config.ExtraBodyContent))
	}
	return ri, nil
}

// processHTTPRequest processes HTTP request, and handles pagination if enabled
func (r *requester) processHTTPRequest(ctx context.Context, publisher cursor.Publisher, ri *requestInfo) error {
	ri.url = r.getURL()
	fmt.Println(ri.url)
	var (
		m, v         interface{}
		response, mm map[string]interface{}
	)

	for {
		req, err := r.createHTTPRequest(ctx, ri)
		if err != nil {
			return errors.Wrapf(err, "failed to create http request")
		}
		msg, err := r.client.Do(req)
		if err != nil {
			return errors.Wrapf(err, "failed to execute http client.Do")
		}
		responseData, err := ioutil.ReadAll(msg.Body)
		header := msg.Header
		msg.Body.Close()
		if err != nil {
			return errors.Wrapf(err, "failed to read http.response.body")
		}
		if msg.StatusCode != http.StatusOK {
			r.log.Debugw("HTTP request failed", "http.response.status_code", msg.StatusCode, "http.response.body", string(responseData))
			if msg.StatusCode == http.StatusTooManyRequests {
				if err = r.applyRateLimit(ctx, header, r.config.RateLimit); err != nil {
					return err
				}
				continue
			}
			return errors.Errorf("http request was unsuccessful with a status code %d", msg.StatusCode)
		}

		err = json.Unmarshal(responseData, &m)
		if err != nil {
			r.log.Debug("failed to unmarshal http.response.body", string(responseData))
			return errors.Wrapf(err, "failed to unmarshal http.response.body")
		}
		switch obj := m.(type) {
		// Top level Array
		case []interface{}:
			mm, err = r.processEventArray(publisher, obj)
			if err != nil {
				return err
			}
		case map[string]interface{}:
			response = obj
			if r.config.JSONObjects == "" {
				mm, err = r.processEventArray(publisher, []interface{}{obj})
				if err != nil {
					return err
				}
			} else {
				v, err = common.MapStr(obj).GetValue(r.config.JSONObjects)
				if err != nil {
					if err == common.ErrKeyNotFound {
						break
					}
					return err
				}
				switch ts := v.(type) {
				case []interface{}:
					mm, err = r.processEventArray(publisher, ts)
					if err != nil {
						return err
					}
				default:
					return errors.Errorf("content of %s is not a valid array", r.config.JSONObjects)
				}
			}
		default:
			r.log.Debug("http.response.body is not a valid JSON object", string(responseData))
			return errors.Errorf("http.response.body is not a valid JSON object, but a %T", obj)
		}

		if mm != nil && r.config.Pagination.IsEnabled() {
			if r.config.Pagination.Header != nil {
				// Pagination control using HTTP Header
				url, err := getNextLinkFromHeader(header, r.config.Pagination.Header.FieldName, r.config.Pagination.Header.RegexPattern)
				if err != nil {
					return errors.Wrapf(err, "failed to retrieve the next URL for pagination")
				}
				if ri.url == url || url == "" {
					r.log.Info("Pagination finished.")
					break
				}
				ri.url = url
				if err = r.applyRateLimit(ctx, header, r.config.RateLimit); err != nil {
					return err
				}
				r.log.Info("Continuing with pagination to URL: ", ri.url)
				continue
			} else {
				// Pagination control using HTTP Body fields
				ri, err = createRequestInfoFromBody(r.config.Pagination, common.MapStr(response), common.MapStr(mm), ri)
				if err != nil {
					return err
				}
				if ri == nil {
					break
				}
				if err = r.applyRateLimit(ctx, header, r.config.RateLimit); err != nil {
					return err
				}
				r.log.Info("Continuing with pagination to URL: ", ri.url)
				continue
			}
		}
		break
	}

	if mm != nil && r.config.DateCursor.IsEnabled() {
		r.advanceCursor(common.MapStr(mm))
	}

	return nil
}

func (r *requester) getURL() string {
	if !r.config.DateCursor.IsEnabled() {
		return r.config.URL
	}

	var dateStr string
	if r.cursorValue == "" {
		t := timeNow().UTC().Add(-r.config.DateCursor.InitialInterval)
		dateStr = t.Format(r.config.DateCursor.GetDateFormat())
	} else {
		dateStr = r.cursorValue
	}

	url, err := url.Parse(r.config.URL)
	if err != nil {
		return r.config.URL
	}

	q := url.Query()

	var value string
	if r.config.DateCursor.ValueTemplate == nil {
		value = dateStr
	} else {
		buf := new(bytes.Buffer)
		if err := r.config.DateCursor.ValueTemplate.Execute(buf, dateStr); err != nil {
			return r.config.URL
		}
		value = buf.String()
	}

	q.Set(r.config.DateCursor.URLField, value)

	url.RawQuery = q.Encode()

	return url.String()
}

func (r *requester) advanceCursor(m common.MapStr) {
	if r.config.DateCursor.Field == "" {
		r.cursorValue = time.Now().UTC().Format(r.config.DateCursor.GetDateFormat())
		return
	}

	v, err := m.GetValue(r.config.DateCursor.Field)
	if err != nil {
		r.log.Warnf("date_cursor field: %q", err)
		return
	}
	switch t := v.(type) {
	case string:
		_, err := time.Parse(r.config.DateCursor.GetDateFormat(), t)
		if err != nil {
			r.log.Warn("date_cursor field does not have the expected layout")
			return
		}
		r.cursorValue = t
	default:
		r.log.Warn("date_cursor field must be a string, cursor will not advance")
		return
	}
}

func (r *requester) loadCheckpoint(cursor cursor.Cursor) {
	var nextCursorValue string
	if cursor.IsNew() {
		return
	}

	if err := cursor.Unpack(&nextCursorValue); err != nil {
		r.log.Errorf("Reset cursor position. Failed to read checkpoint from registry: %v", err)
		return
	}

	r.cursorValue = nextCursorValue
}
