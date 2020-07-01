// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const (
	HTTPTestServer int = iota
	TLSTestServer
	RateLimitRetryServer
	ErrorRetryServer
	ArrayResponseServer
)

var (
	once sync.Once
)

func testSetup(t *testing.T) {
	t.Helper()
	once.Do(func() {
		logp.TestingSetup()
	})
}

func createTestServer(testServer int) *httptest.Server {
	var ts *httptest.Server
	newServer := httptest.NewServer
	switch testServer {
	case HTTPTestServer:
		ts = createServer(newServer)
	case TLSTestServer:
		ts = createServer(httptest.NewTLSServer)
	case RateLimitRetryServer:
		ts = createCustomServer(newServer)
	case ErrorRetryServer:
		ts = createCustomRetryServer(newServer)
	case ArrayResponseServer:
		ts = createCustomServerWithArrayResponse(newServer)
	default:
		ts = createServer(newServer)
	}
	return ts
}

func createServer(newServer func(handler http.Handler) *httptest.Server) *httptest.Server {
	return newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			req, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				log.Fatalln(err)
			}
			var m interface{}
			err = json.Unmarshal(req, &m)
			w.Header().Set("Content-Type", "application/json")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write(req)
			}
		} else {
			message := map[string]interface{}{
				"hello": "world",
				"embedded": map[string]string{
					"hello": "world",
				},
				"list": []map[string]interface{}{
					{"foo": "bar"},
					{"hello": "world"},
				},
			}
			b, _ := json.Marshal(message)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}
	}))
}

func createCustomServer(newServer func(handler http.Handler) *httptest.Server) *httptest.Server {
	var isRetry bool
	return newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !isRetry {
			w.Header().Set("X-Rate-Limit-Limit", "0")
			w.Header().Set("X-Rate-Limit-Remaining", "0")
			w.Header().Set("X-Rate-Limit-Reset", strconv.FormatInt(time.Now().Unix(), 10))
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte{})
			isRetry = true
		} else {
			message := map[string]interface{}{
				"hello": "world",
				"embedded": map[string]string{
					"hello": "world",
				},
			}
			b, _ := json.Marshal(message)
			w.WriteHeader(http.StatusOK)
			w.Write(b)
			isRetry = false
		}
	}))
}

func createCustomRetryServer(newServer func(handler http.Handler) *httptest.Server) *httptest.Server {
	retryCount := 0
	statusCodes := []int{http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusHTTPVersionNotSupported, http.StatusVariantAlsoNegotiates, http.StatusInsufficientStorage, http.StatusLoopDetected, http.StatusNotExtended, http.StatusNetworkAuthenticationRequired}
	return newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Test retry for two times
		if retryCount < 2 {
			rand.Seed(time.Now().Unix())
			code := statusCodes[rand.Intn(len(statusCodes))]
			w.WriteHeader(code)
			w.Write([]byte{})
			retryCount++
		} else {
			message := map[string]interface{}{
				"hello": "world",
				"embedded": map[string]string{
					"hello": "world",
				},
			}
			b, _ := json.Marshal(message)
			w.WriteHeader(http.StatusOK)
			w.Write(b)
			retryCount = 0
		}
	}))
}

func createCustomServerWithArrayResponse(newServer func(handler http.Handler) *httptest.Server) *httptest.Server {
	return newServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		message := map[string]interface{}{
			"hello": []map[string]interface{}{
				{
					"foo": "bar",
					"list": []map[string]interface{}{
						{"foo": "bar"},
						{"hello": "world"},
					},
				},
				{
					"foo": "bar",
					"list": []map[string]interface{}{
						{"foo": "bar"},
					},
				},
				{
					"bar":  "foo",
					"list": []map[string]interface{}{},
				},
				{"bar": "foo"},
			},
		}
		b, _ := json.Marshal(message)
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))
}

func runTest(t *testing.T, ts *httptest.Server, m map[string]interface{}, run func(input *HttpjsonInput, out *stubOutleter, t *testing.T)) {
	testSetup(t)
	defer ts.Close()
	m["url"] = ts.URL
	cfg := common.MustNewConfigFrom(m)
	// Simulate input.Context from Filebeat input runner.
	inputCtx := newInputContext()
	defer close(inputCtx.Done)

	// Stub outlet for receiving events generated by the input.
	eventOutlet := newStubOutlet()
	defer eventOutlet.Close()

	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return eventOutlet, nil
	})

	in, err := NewInput(cfg, connector, inputCtx)
	if err != nil {
		t.Fatal(err)
	}
	input := in.(*HttpjsonInput)
	defer input.Stop()

	run(input, eventOutlet, t)
}

func newInputContext() input.Context {
	return input.Context{
		Done: make(chan struct{}),
	}
}

type stubOutleter struct {
	sync.Mutex
	cond   *sync.Cond
	done   bool
	Events []beat.Event
}

func newStubOutlet() *stubOutleter {
	o := &stubOutleter{}
	o.cond = sync.NewCond(o)
	return o
}

func (o *stubOutleter) waitForEvents(numEvents int) ([]beat.Event, bool) {
	o.Lock()
	defer o.Unlock()

	for len(o.Events) < numEvents && !o.done {
		o.cond.Wait()
	}

	size := numEvents
	if size >= len(o.Events) {
		size = len(o.Events)
	}

	out := make([]beat.Event, size)
	copy(out, o.Events)
	return out, len(out) == numEvents
}

func (o *stubOutleter) Close() error {
	o.Lock()
	defer o.Unlock()
	o.done = true
	return nil
}

func (o *stubOutleter) Done() <-chan struct{} { return nil }

func (o *stubOutleter) OnEvent(event beat.Event) bool {
	o.Lock()
	defer o.Unlock()
	o.Events = append(o.Events, event)
	o.cond.Broadcast()
	return !o.done
}

func newOAuth2TestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != "POST" {
			t.Errorf("expected POST request, got %v", r.Method)
			return
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("no error expected, got %q", err)
			return
		}

		if gt := r.FormValue("grant_type"); gt != "client_credentials" {
			t.Errorf("expected grant_type was client_credentials, got %q", gt)
			return
		}

		clientID := r.FormValue("client_id")
		clientSecret := r.FormValue("client_secret")
		if clientID == "" || clientSecret == "" {
			clientID, clientSecret, _ = r.BasicAuth()
		}
		if clientID != "a_client_id" || clientSecret != "a_client_secret" {
			t.Errorf("expected client credentials \"a_client_id:a_client_secret\", got \"%s:%s\"", clientID, clientSecret)
		}

		if s := r.FormValue("scope"); s != "scope1 scope2" {
			t.Errorf("expected scope was scope1+scope2, got %q", s)
			return
		}

		expectedParams := []string{"v1", "v2"}
		if p := r.Form["param1"]; !reflect.DeepEqual(expectedParams, p) {
			t.Errorf("expected params were %q, but got %q", expectedParams, p)
			return
		}

		w.Header().Set("content-type", "application/json")
		w.Write([]byte(`{"token_type":"Bearer","expires_in":"3599","access_token":"abcdef1234567890"}`))
	}))
}

// --- Test Cases

func TestGetNextLinkFromHeader(t *testing.T) {
	header := make(http.Header)
	header.Add("Link", "<https://dev-168980.okta.com/api/v1/logs>; rel=\"self\"")
	header.Add("Link", "<https://dev-168980.okta.com/api/v1/logs?after=1581658181086_1>; rel=\"next\"")
	re, _ := regexp.Compile("<([^>]+)>; *rel=\"next\"(?:,|$)")
	url, err := getNextLinkFromHeader(header, "Link", re)
	if url != "https://dev-168980.okta.com/api/v1/logs?after=1581658181086_1" {
		t.Fatal("Failed to test getNextLinkFromHeader. URL " + url + " is not expected")
	}
	if err != nil {
		t.Fatal("Failed to test getNextLinkFromHeader with error:", err)
	}
}

func TestCreateRequestInfoFromBody(t *testing.T) {
	m := map[string]interface{}{
		"id": 100,
	}
	extraBodyContent := common.MapStr{"extra_body": "abc"}
	ri, err := createRequestInfoFromBody(common.MapStr(m), "id", "pagination_id", extraBodyContent, "https://test-123", &RequestInfo{
		URL:        "",
		ContentMap: common.MapStr{},
		Headers:    common.MapStr{},
	})
	if ri.URL != "https://test-123" {
		t.Fatal("Failed to test createRequestInfoFromBody. URL should be https://test-123.")
	}
	p, err := ri.ContentMap.GetValue("pagination_id")
	if err != nil {
		t.Fatal("Failed to test createRequestInfoFromBody with error", err)
	}
	switch pt := p.(type) {
	case int:
		if pt != 100 {
			t.Fatalf("Failed to test createRequestInfoFromBody. pagination_id value %d should be 100.", pt)
		}
	default:
		t.Fatalf("Failed to test createRequestInfoFromBody. pagination_id value %T should be int.", pt)
	}
	b, err := ri.ContentMap.GetValue("extra_body")
	if err != nil {
		t.Fatal("Failed to test createRequestInfoFromBody with error", err)
	}
	switch bt := b.(type) {
	case string:
		if bt != "abc" {
			t.Fatalf("Failed to test createRequestInfoFromBody. extra_body value %s does not match \"abc\".", bt)
		}
	default:
		t.Fatalf("Failed to test createRequestInfoFromBody. extra_body type %T should be string.", bt)
	}
}

// Test getRateLimit function with a remaining quota, expect to receive 0, nil.
func TestGetRateLimitCase1(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "118")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	rateLimit := &RateLimit{
		Limit:     "X-Rate-Limit-Limit",
		Reset:     "X-Rate-Limit-Reset",
		Remaining: "X-Rate-Limit-Remaining",
	}
	epoch, err := getRateLimit(header, rateLimit)
	if err != nil || epoch != 0 {
		t.Fatal("Failed to test getRateLimit.")
	}
}

// Test getRateLimit function with a past time, expect to receive 0, nil.
func TestGetRateLimitCase2(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	rateLimit := &RateLimit{
		Limit:     "X-Rate-Limit-Limit",
		Reset:     "X-Rate-Limit-Reset",
		Remaining: "X-Rate-Limit-Remaining",
	}
	epoch, err := getRateLimit(header, rateLimit)
	if err != nil || epoch != 0 {
		t.Fatal("Failed to test getRateLimit.")
	}
}

// Test getRateLimit function with a time yet to come, expect to receive <reset-value>, nil.
func TestGetRateLimitCase3(t *testing.T) {
	epoch := time.Now().Unix() + 100
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(epoch, 10))
	rateLimit := &RateLimit{
		Limit:     "X-Rate-Limit-Limit",
		Reset:     "X-Rate-Limit-Reset",
		Remaining: "X-Rate-Limit-Remaining",
	}
	epoch2, err := getRateLimit(header, rateLimit)
	if err != nil || epoch2 != epoch {
		t.Fatal("Failed to test getRateLimit.")
	}
}

func TestGET(t *testing.T) {
	m := map[string]interface{}{
		"http_method": "GET",
		"interval":    0,
	}
	ts := createTestServer(HTTPTestServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(1)
		if !ok {
			t.Fatalf("Expected 1 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestGetHTTPS(t *testing.T) {
	m := map[string]interface{}{
		"http_method":           "GET",
		"interval":              0,
		"ssl.verification_mode": "none",
	}
	ts := createTestServer(HTTPTestServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(1)
		if !ok {
			t.Fatalf("Expected 1 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRateLimitRetry(t *testing.T) {
	m := map[string]interface{}{
		"http_method": "GET",
		"interval":    0,
	}
	ts := createTestServer(RateLimitRetryServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(1)
		if !ok {
			t.Fatalf("Expected 1 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestErrorRetry(t *testing.T) {
	m := map[string]interface{}{
		"http_method": "GET",
		"interval":    0,
	}
	ts := createTestServer(ErrorRetryServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(1)
		if !ok {
			t.Fatalf("Expected 1 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestArrayResponse(t *testing.T) {
	m := map[string]interface{}{
		"http_method":        "GET",
		"json_objects_array": "hello",
		"interval":           0,
	}
	ts := createTestServer(ArrayResponseServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(2)
		if !ok {
			t.Fatalf("Expected 2 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPOST(t *testing.T) {
	m := map[string]interface{}{
		"http_method":       "POST",
		"http_request_body": map[string]interface{}{"test": "abc", "testNested": map[string]interface{}{"testNested1": 123}},
		"interval":          0,
	}
	ts := createTestServer(HTTPTestServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(1)
		if !ok {
			t.Fatalf("Expected 1 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRepeatedPOST(t *testing.T) {
	m := map[string]interface{}{
		"http_method":       "POST",
		"http_request_body": map[string]interface{}{"test": "abc", "testNested": map[string]interface{}{"testNested1": 123}},
		"interval":          10 ^ 9,
	}
	ts := createTestServer(HTTPTestServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(3)
		if !ok {
			t.Fatalf("Expected 3 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRunStop(t *testing.T) {
	m := map[string]interface{}{
		"http_method": "GET",
		"interval":    0,
	}
	ts := createTestServer(HTTPTestServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		input.Run()
		input.Stop()
		input.Run()
		input.Stop()
	})
}

func TestOAuth2(t *testing.T) {
	oAuth2Server := newOAuth2TestServer(t)
	defer oAuth2Server.Close()
	ts := createTestServer(HTTPTestServer)
	defer ts.Close()
	m := map[string]interface{}{
		"http_method":          "GET",
		"oauth2.client.id":     "a_client_id",
		"oauth2.client.secret": "a_client_secret",
		"oauth2.token_url":     oAuth2Server.URL,
		"oauth2.endpoint_params": map[string][]string{
			"param1": {"v1", "v2"},
		},
		"oauth2.scopes": []string{"scope1", "scope2"},
		"interval":      0,
	}

	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(1)
		if !ok {
			t.Fatalf("Expected 1 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSplitResponseWithKey(t *testing.T) {
	m := map[string]interface{}{
		"http_method":     "GET",
		"split_events_by": "list",
		"interval":        0,
	}
	ts := createTestServer(HTTPTestServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(2)
		if !ok {
			t.Fatalf("Expected 2 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSplitResponseWithoutKey(t *testing.T) {
	m := map[string]interface{}{
		"http_method":     "GET",
		"split_events_by": "not_found",
		"interval":        0,
	}
	ts := createTestServer(HTTPTestServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(1)
		if !ok {
			t.Fatalf("Expected 1 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestArrayWithSplitResponse(t *testing.T) {
	m := map[string]interface{}{
		"http_method":        "GET",
		"json_objects_array": "hello",
		"split_events_by":    "list",
		"interval":           0,
	}

	expectedFields := []string{
		`{
			"foo": "bar",
			"list": {
				"foo": "bar"
			}
		}`,
		`{
			"foo": "bar",
			"list": {
				"hello": "world"
			}
		}`,
		`{
			"foo": "bar",
			"list": {
				"foo": "bar"
			}
		}`,
		`{
			"bar":  "foo",
			"list": []
		}`,
		`{"bar": "foo"}`,
	}

	ts := createTestServer(ArrayResponseServer)
	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(5)
		if !ok {
			t.Fatalf("Expected 5 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}

		for i, e := range events {
			message, _ := e.GetValue("message")
			assert.JSONEq(t, expectedFields[i], message.(string))
		}
	})
}

func TestCursor(t *testing.T) {
	m := map[string]interface{}{
		"http_method":                  "GET",
		"date_cursor.field":            "@timestamp",
		"date_cursor.url_field":        "$filter",
		"date_cursor.value_template":   "alertCreationTime ge {{.}}",
		"date_cursor.initial_interval": "10m",
		"date_cursor.date_format":      "2006-01-02T15:04:05Z",
	}

	timeNow = func() time.Time {
		t, _ := time.Parse("2006-01-02T15:04:05Z", "2002-10-02T15:10:00Z")
		return t
	}

	const (
		expectedQuery           = "%24filter=alertCreationTime+ge+2002-10-02T15%3A00%3A00Z"
		expectedNextCursorValue = "2002-10-02T15:00:01Z"
		expectedNextQuery       = "%24filter=alertCreationTime+ge+2002-10-02T15%3A00%3A01Z"
	)
	var gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		gotQuery = r.URL.Query().Encode()
		w.Write([]byte(`[{"@timestamp":"2002-10-02T15:00:00Z"},{"@timestamp":"2002-10-02T15:00:01Z"}]`))
	}))

	runTest(t, ts, m, func(input *HttpjsonInput, out *stubOutleter, t *testing.T) {
		group, _ := errgroup.WithContext(context.Background())
		group.Go(input.run)

		events, ok := out.waitForEvents(2)
		if !ok {
			t.Fatalf("Expected 2 events, but got %d.", len(events))
		}
		input.Stop()

		if err := group.Wait(); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, expectedQuery, gotQuery)
		assert.Equal(t, expectedNextCursorValue, input.nextCursorValue)
		assert.Equal(t, fmt.Sprintf("%s?%s", ts.URL, expectedNextQuery), input.getURL())
	})
}
