// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/common/useragent"
	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

const (
	inputName = "httpjson"
)

var userAgent = useragent.UserAgent("Filebeat")

// for testing
var timeNow = time.Now

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

// HttpjsonInput struct has the HttpJsonInput configuration and other userful info.
type HttpjsonInput struct {
	config
	log      *logp.Logger
	outlet   channel.Outleter // Output of received messages.
	inputCtx context.Context  // Wraps the Done channel from parent input.Context.

	workerCtx    context.Context    // Worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc // Used to signal that the worker should stop.
	workerOnce   sync.Once          // Guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup     // Waits on worker goroutine.

	nextCursorValue string
}

// RequestInfo struct has the information for generating an HTTP request
type RequestInfo struct {
	URL        string
	ContentMap common.MapStr
	Headers    common.MapStr
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

// NewInput creates a new httpjson input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	// Extract and validate the input's configuration.
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		return nil, err
	}
	// Build outlet for events.
	out, err := connector.Connect(cfg)
	if err != nil {
		return nil, err
	}

	// Wrap input.Context's Done channel with a context.Context. This goroutine
	// stops with the parent closes the Done channel.
	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Done:
		case <-inputCtx.Done():
		}
	}()

	// If the input ever needs to be made restartable, then context would need
	// to be recreated with each restart.
	workerCtx, workerCancel := context.WithCancel(inputCtx)

	in := &HttpjsonInput{
		config: conf,
		log: logp.NewLogger("httpjson").With(
			"url", conf.URL),
		outlet:       out,
		inputCtx:     inputCtx,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	in.log.Info("Initialized httpjson input.")
	return in, nil
}

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (in *HttpjsonInput) Run() {
	in.workerOnce.Do(func() {
		in.workerWg.Add(1)
		go func() {
			in.log.Info("httpjson input worker has started.")
			defer in.log.Info("httpjson input worker has stopped.")
			defer in.workerWg.Done()
			defer in.workerCancel()
			if err := in.run(); err != nil {
				in.log.Error(err)
				return
			}
		}()
	})
}

// createHTTPRequest creates an HTTP/HTTPs request for the input
func (in *HttpjsonInput) createHTTPRequest(ctx context.Context, ri *RequestInfo) (*http.Request, error) {
	var body io.Reader
	if len(ri.ContentMap) == 0 || in.config.NoHTTPBody {
		body = nil
	} else {
		b, err := json.Marshal(ri.ContentMap)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(in.config.HTTPMethod, ri.URL, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if in.config.APIKey != "" {
		if in.config.AuthenticationScheme != "" {
			req.Header.Set("Authorization", in.config.AuthenticationScheme+" "+in.config.APIKey)
		} else {
			req.Header.Set("Authorization", in.config.APIKey)
		}
	}
	for k, v := range ri.Headers {
		switch vv := v.(type) {
		case string:
			req.Header.Set(k, vv)
		default:
		}
	}
	return req, nil
}

// processEventArray publishes an event for each object contained in the array. It returns the last object in the array and an error if any.
func (in *HttpjsonInput) processEventArray(events []interface{}) (map[string]interface{}, error) {
	var last map[string]interface{}
	for _, t := range events {
		switch v := t.(type) {
		case map[string]interface{}:
			for _, e := range in.splitEvent(v) {
				last = e
				d, err := json.Marshal(e)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to marshal %+v", e)
				}
				ok := in.outlet.OnEvent(makeEvent(string(d)))
				if !ok {
					return nil, errors.New("function OnEvent returned false")
				}
			}
		default:
			return nil, errors.Errorf("expected only JSON objects in the array but got a %T", v)
		}
	}
	return last, nil
}

func (in *HttpjsonInput) splitEvent(event map[string]interface{}) []map[string]interface{} {
	m := common.MapStr(event)

	hasSplitKey, _ := m.HasKey(in.config.SplitEventsBy)
	if in.config.SplitEventsBy == "" || !hasSplitKey {
		return []map[string]interface{}{event}
	}

	splitOnIfc, _ := m.GetValue(in.config.SplitEventsBy)
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
		_, err := mm.Put(in.config.SplitEventsBy, s)
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
func (in *HttpjsonInput) applyRateLimit(ctx context.Context, header http.Header, rateLimit *RateLimit) error {
	epoch, err := getRateLimit(header, rateLimit)
	if err != nil {
		return err
	}
	t := time.Unix(epoch, 0)
	w := time.Until(t)
	if epoch == 0 || w <= 0 {
		in.log.Debugf("Rate Limit: No need to apply rate limit.")
		return nil
	}
	in.log.Debugf("Rate Limit: Wait until %v for the rate limit to reset.", t)
	ticker := time.NewTicker(w)
	defer ticker.Stop()
	select {
	case <-ctx.Done():
		in.log.Info("Context done.")
		return nil
	case <-ticker.C:
		in.log.Debug("Rate Limit: time is up.")
		return nil
	}
}

// createRequestInfoFromBody creates a new RequestInfo for a new HTTP request in pagination based on HTTP response body
func createRequestInfoFromBody(m common.MapStr, idField string, requestField string, extraBodyContent common.MapStr, url string, ri *RequestInfo) (*RequestInfo, error) {
	v, err := m.GetValue(idField)
	if err != nil {
		if err == common.ErrKeyNotFound {
			return nil, nil
		} else {
			return nil, errors.Wrapf(err, "failed to retrieve id_field for pagination")
		}
	}
	if requestField != "" {
		ri.ContentMap.Put(requestField, v)
		if url != "" {
			ri.URL = url
		}
	} else {
		switch vt := v.(type) {
		case string:
			ri.URL = vt
		default:
			return nil, errors.New("pagination ID is not of string type")
		}
	}
	if len(extraBodyContent) > 0 {
		ri.ContentMap.Update(extraBodyContent)
	}
	return ri, nil
}

// processHTTPRequest processes HTTP request, and handles pagination if enabled
func (in *HttpjsonInput) processHTTPRequest(ctx context.Context, client *http.Client, ri *RequestInfo) error {
	ri.URL = in.getURL()
	for {
		req, err := in.createHTTPRequest(ctx, ri)
		if err != nil {
			return errors.Wrapf(err, "failed to create http request")
		}
		msg, err := client.Do(req)
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
			in.log.Debugw("HTTP request failed", "http.response.status_code", msg.StatusCode, "http.response.body", string(responseData))
			if msg.StatusCode == http.StatusTooManyRequests {
				if err = in.applyRateLimit(ctx, header, in.config.RateLimit); err != nil {
					return err
				}
				continue
			}
			return errors.Errorf("http request was unsuccessful with a status code %d", msg.StatusCode)
		}
		var m, v interface{}
		var mm map[string]interface{}
		err = json.Unmarshal(responseData, &m)
		if err != nil {
			in.log.Debug("failed to unmarshal http.response.body", string(responseData))
			return errors.Wrapf(err, "failed to unmarshal http.response.body")
		}
		switch obj := m.(type) {
		// Top level Array
		case []interface{}:
			mm, err = in.processEventArray(obj)
			if err != nil {
				return err
			}
		case map[string]interface{}:
			if in.config.JSONObjects == "" {
				mm, err = in.processEventArray([]interface{}{obj})
				if err != nil {
					return err
				}
			} else {
				v, err = common.MapStr(obj).GetValue(in.config.JSONObjects)
				if err != nil {
					return err
				}
				switch ts := v.(type) {
				case []interface{}:
					mm, err = in.processEventArray(ts)
					if err != nil {
						return err
					}
				default:
					return errors.Errorf("content of %s is not a valid array", in.config.JSONObjects)
				}
			}
		default:
			in.log.Debug("http.response.body is not a valid JSON object", string(responseData))
			return errors.Errorf("http.response.body is not a valid JSON object, but a %T", obj)
		}
		if mm != nil && in.config.Pagination.IsEnabled() {
			if in.config.Pagination.Header != nil {
				// Pagination control using HTTP Header
				url, err := getNextLinkFromHeader(header, in.config.Pagination.Header.FieldName, in.config.Pagination.Header.RegexPattern)
				if err != nil {
					return errors.Wrapf(err, "failed to retrieve the next URL for pagination")
				}
				if ri.URL == url || url == "" {
					in.log.Info("Pagination finished.")
					return nil
				}
				ri.URL = url
				if err = in.applyRateLimit(ctx, header, in.config.RateLimit); err != nil {
					return err
				}
				in.log.Info("Continuing with pagination to URL: ", ri.URL)
				continue
			} else {
				// Pagination control using HTTP Body fields
				ri, err = createRequestInfoFromBody(common.MapStr(mm), in.config.Pagination.IDField, in.config.Pagination.RequestField, common.MapStr(in.config.Pagination.ExtraBodyContent), in.config.Pagination.URL, ri)
				if err != nil {
					return err
				}
				if ri == nil {
					return nil
				}
				if err = in.applyRateLimit(ctx, header, in.config.RateLimit); err != nil {
					return err
				}
				in.log.Info("Continuing with pagination to URL: ", ri.URL)
				continue
			}
		}
		if mm != nil && in.config.DateCursor.IsEnabled() {
			in.advanceCursor(common.MapStr(mm))
		}
		return nil
	}
}

func (in *HttpjsonInput) getURL() string {
	if !in.config.DateCursor.IsEnabled() {
		return in.config.URL
	}

	var dateStr string
	if in.nextCursorValue == "" {
		t := timeNow().UTC().Add(-in.config.DateCursor.InitialInterval)
		dateStr = t.Format(in.config.DateCursor.GetDateFormat())
	} else {
		dateStr = in.nextCursorValue
	}

	url, err := url.Parse(in.config.URL)
	if err != nil {
		return in.config.URL
	}

	q := url.Query()

	var value string
	if in.config.DateCursor.ValueTemplate == nil {
		value = dateStr
	} else {
		buf := new(bytes.Buffer)
		if err := in.config.DateCursor.ValueTemplate.Execute(buf, dateStr); err != nil {
			return in.config.URL
		}
		value = buf.String()
	}

	q.Set(in.config.DateCursor.URLField, value)

	url.RawQuery = q.Encode()

	return url.String()
}

func (in *HttpjsonInput) advanceCursor(m common.MapStr) {
	v, err := m.GetValue(in.config.DateCursor.Field)
	if err != nil {
		in.log.Warnf("date_cursor field: %q", err)
		return
	}
	switch t := v.(type) {
	case string:
		_, err := time.Parse(in.config.DateCursor.GetDateFormat(), t)
		if err != nil {
			return
		}
		in.nextCursorValue = t
	default:
		in.log.Warn("date_cursor field must be a string, cursor will not advance")
		return
	}
}

func (in *HttpjsonInput) run() error {
	ctx, cancel := context.WithCancel(in.workerCtx)
	defer cancel()

	client, err := in.newHTTPClient(ctx)
	if err != nil {
		return err
	}

	ri := &RequestInfo{
		ContentMap: common.MapStr{},
		Headers:    in.HTTPHeaders,
	}
	if in.config.HTTPMethod == "POST" && in.config.HTTPRequestBody != nil {
		ri.ContentMap.Update(common.MapStr(in.config.HTTPRequestBody))
	}
	err = in.processHTTPRequest(ctx, client, ri)
	if err == nil && in.Interval > 0 {
		ticker := time.NewTicker(in.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				in.log.Info("Context done.")
				return nil
			case <-ticker.C:
				in.log.Info("Process another repeated request.")
				err = in.processHTTPRequest(ctx, client, ri)
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}

// Stop stops the misp input and waits for it to fully stop.
func (in *HttpjsonInput) Stop() {
	in.workerCancel()
	in.workerWg.Wait()
}

// Wait is an alias for Stop.
func (in *HttpjsonInput) Wait() {
	in.Stop()
}

func (in *HttpjsonInput) newHTTPClient(ctx context.Context) (*http.Client, error) {
	tlsConfig, err := tlscommon.LoadTLSConfig(in.config.TLS)
	if err != nil {
		return nil, err
	}

	// Make retryable HTTP client
	var client *retryablehttp.Client = &retryablehttp.Client{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: in.config.HTTPClientTimeout,
				}).DialContext,
				TLSClientConfig:   tlsConfig.ToConfig(),
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
