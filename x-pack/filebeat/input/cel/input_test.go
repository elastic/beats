// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//nolint:deadcode,unused // This code will be used later.
package cel

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var inputTests = []struct {
	name          string
	server        func(*testing.T, http.HandlerFunc, map[string]interface{})
	handler       http.HandlerFunc
	config        map[string]interface{}
	persistCursor map[string]interface{}
	want          []map[string]interface{}
	wantCursor    []map[string]interface{}
	wantErr       error
}{
	// Autonomous tests (no FS or net dependency).
	{
		name: "hello_world",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events":[{"message":"Hello, World!"}]}`,
			"state":    nil,
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		want: []map[string]interface{}{
			{"message": "Hello, World!"},
		},
	},
	{
		name: "bad_events_type",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events":["Hello, World!"]}`,
			"state":    nil,
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		wantErr: fmt.Errorf("unexpected type returned for evaluation events: %T", "Hello, World!"),
	},
	{
		name: "hello_world_non_nil_state",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events":[{"message":"Hello, World!"}]}`,
			"state":    map[string]interface{}{},
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		want: []map[string]interface{}{
			{"message": "Hello, World!"},
		},
	},
	{
		name: "what_is_next",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events":[{"message":"Hello, World!"}],"cursor":[{"todo":"What's next?"}]}`,
			"state":    nil,
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		want: []map[string]interface{}{
			{"message": "Hello, World!"},
		},
		wantCursor: []map[string]interface{}{
			{"todo": "What's next?"},
		},
	},
	{
		name: "bad_cursor_type",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events":[{"message":"Hello, World!"}],"cursor":["What's next?"]}`,
			"state":    nil,
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		wantErr: fmt.Errorf("unexpected type returned for evaluation cursor element: %T", "What's next?"),
	},
	{
		name: "show_state",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events":[state]}`,
			"state":    nil,
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		want: []map[string]interface{}{
			{"url": ""},
		},
	},
	{
		name: "show_provided_state",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events":[state]}`,
			"state": map[string]interface{}{
				"we":   "can",
				"put":  []string{"a", "bunch"},
				"of":   "stuff",
				"here": "!",
			},
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		want: []map[string]interface{}{
			{
				"we":   "can",
				"put":  []interface{}{"a", "bunch"}, // We lose typing.
				"of":   "stuff",
				"here": "!",
				"url":  "",
			},
		},
	},
	{
		name: "iterative_state",
		config: map[string]interface{}{
			"interval": 1,
			"program": `
{
	"events":[
		{"message": state.data[state.cursor.next]},
	],
	"cursor":[
		{"next": int(state.cursor.next)+1}, // Ensure we have a number index.
	],
	"data": state.data, // Make sure we have this for the next iteration.
}
`,
			"state": map[string]interface{}{
				"data":   []string{"first", "second", "third"},
				"cursor": map[string]int{"next": 0},
			},
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		want: []map[string]interface{}{
			{"message": "first"},
			{"message": "second"},
			{"message": "third"},
		},
		wantCursor: []map[string]interface{}{
			// The serialisation of numbers is to float when under 1<<53 (strings above).
			// This is not visible within CEL, but presents in Go testing.
			{"next": 1.0},
			{"next": 2.0},
			{"next": 3.0},
		},
	},
	{
		name: "iterative_state_implicit_initial_cursor",
		config: map[string]interface{}{
			"interval": 1,
			"program": `
int(has(state.cursor) && has(state.cursor.next) ? state.cursor.next : 0).as(index, {
	"events":[
		{"message": state.data[index]},
	],
	"cursor":[
		{"next": index+1},
	],
	"data": state.data, // Make sure we have this for the next iteration.
})
`,
			"state": map[string]interface{}{
				"data": []string{"first", "second", "third"},
			},
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		want: []map[string]interface{}{
			{"message": "first"},
			{"message": "second"},
			{"message": "third"},
		},
		wantCursor: []map[string]interface{}{
			// The serialisation of numbers is to float when under 1<<53 (strings above).
			// This is not visible within CEL, but presents in Go testing.
			{"next": 1.0},
			{"next": 2.0},
			{"next": 3.0},
		},
	},
	{
		name: "iterative_state_provided_stored_cursor",
		config: map[string]interface{}{
			"interval": 1,
			"program": `
{
	"events":[
		{"message": state.data[state.cursor.next]},
	],
	"cursor":[
		{"next": int(state.cursor.next)+1}, // Ensure we have a number index.
	],
	"data": state.data, // Make sure we have this for the next iteration.
}
`,
			"state": map[string]interface{}{
				"data":   []string{"first", "second", "third"},
				"cursor": map[string]int{"next": 0},
			},
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		persistCursor: map[string]interface{}{
			"next": 1,
		},
		want: []map[string]interface{}{
			{"message": "second"},
			{"message": "third"},
		},
		wantCursor: []map[string]interface{}{
			// The serialisation of numbers is to float when under 1<<53 (strings above).
			// This is not visible within CEL, but presents in Go testing.
			{"next": 2.0},
			{"next": 3.0},
		},
	},
	{
		name: "iterative_state_implicit_initial_cursor_provided_stored_cursor",
		config: map[string]interface{}{
			"interval": 1,
			"program": `
int(has(state.cursor) && has(state.cursor.next) ? state.cursor.next : 0).as(index, {
	"events":[
		{"message": state.data[index]},
	],
	"cursor":[
		{"next": index+1},
	],
	"data": state.data, // Make sure we have this for the next iteration.
})
`,
			"state": map[string]interface{}{
				"data": []string{"first", "second", "third"},
			},
			"resource": map[string]interface{}{
				"url": "",
			},
		},
		persistCursor: map[string]interface{}{
			"next": 1,
		},
		want: []map[string]interface{}{
			{"message": "second"},
			{"message": "third"},
		},
		wantCursor: []map[string]interface{}{
			// The serialisation of numbers is to float when under 1<<53 (strings above).
			// This is not visible within CEL, but presents in Go testing.
			{"next": 2.0},
			{"next": 3.0},
		},
	},

	// FS-based tests.
	{
		name: "ndjson_log_file_simple",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events": try(file(state.url, "application/x-ndjson").map(e, try(e, "error.message")), "file.error")}`,
			"resource": map[string]interface{}{
				"url": "testdata/log-1.ndjson",
			},
		},
		want: []map[string]interface{}{
			{"level": "info", "message": "something happened"},
			{"level": "error", "message": "something bad happened"},
		},
	},
	{
		name: "ndjson_log_file_simple_file_scheme",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events": try(file(state.url, "application/x-ndjson").map(e, try(e, "error.message")), "file.error")}`,
			"resource": map[string]interface{}{
				"url": fileSchemePath("testdata/log-1.ndjson"),
			},
		},
		want: []map[string]interface{}{
			{"level": "info", "message": "something happened"},
			{"level": "error", "message": "something bad happened"},
		},
	},
	{
		name: "ndjson_log_file_corrupted",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events": try(file(state.url, "application/x-ndjson").map(e, try(e, "error.message")), "file.error")}`,
			"resource": map[string]interface{}{
				"url": "testdata/corrupted-log-1.ndjson",
			},
		},
		want: []map[string]interface{}{
			{"level": "info", "message": "something happened"},
			{"error.message": `unexpected end of JSON input: {"message":"Dave, stop. Stop, will you? Stop, Dave. Will you stop, Dave? Stop, Dave."`},
			{"level": "error", "message": "something bad happened"},
		},
	},
	{
		name: "missing_file",
		config: map[string]interface{}{
			"interval": 1,
			"program":  `{"events": try(file(state.url, "application/x-ndjson").map(e, try(e, "error.message")), "file.error")}`,
			"resource": map[string]interface{}{
				"url": "testdata/absent.ndjson",
			},
		},
		want: []map[string]interface{}{
			{"file.error": "file: " + missingFileError("testdata/absent.ndjson")},
		},
	},

	// HTTP-based tests.
	{
		name:   "GET_request",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
bytes(get(state.url).Body).as(body, {
	"events": [body.decode_json()]
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"hello": []interface{}{
					map[string]interface{}{
						"world": "moon",
					},
					map[string]interface{}{
						"space": []interface{}{
							map[string]interface{}{
								"cake": "pumpkin",
							},
						},
					},
				},
			},
		},
	},
	{
		name:   "GET_request_TLS",
		server: newTestServer(httptest.NewTLSServer),
		config: map[string]interface{}{
			"interval":                       1,
			"resource.ssl.verification_mode": "none",
			"program": `
bytes(get(state.url).Body).as(body, {
	"events": [body.decode_json()]
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"hello": []interface{}{
					map[string]interface{}{
						"world": "moon",
					},
					map[string]interface{}{
						"space": []interface{}{
							map[string]interface{}{
								"cake": "pumpkin",
							},
						},
					},
				},
			},
		},
	},
	{
		name:   "retry_after_request",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
get(state.url).as(resp, {
	"url": state.url,
	"events": [bytes(resp.Body).decode_json()],
	"status_code": resp.StatusCode,
	"header": resp.Header,
})
`,
		},
		handler: retryAfterHandler("1"),
		want: []map[string]interface{}{
			{"hello": "world"},
		},
	},
	{
		name:   "retry_after_request_time",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
get(state.url).as(resp, {
	"url": state.url,
	"events": [bytes(resp.Body).decode_json()],
	"status_code": resp.StatusCode,
	"header": resp.Header,
})
`,
		},
		handler: retryAfterHandler(time.Now().Add(time.Second).UTC().Format(http.TimeFormat)),
		want: []map[string]interface{}{
			{"hello": "world"},
		},
	},
	{
		name:   "rate_limit_request_0",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
get(state.url).as(resp, {
	"url": state.url,
	"events": [bytes(resp.Body).decode_json()],
	"status_code": resp.StatusCode,
	"header": resp.Header,
	"rate_limit": rate_limit(resp.Header, 'okta', duration('1m')),
})
`,
		},
		handler: rateLimitHandler("0", 100*time.Millisecond),
		want: []map[string]interface{}{
			{"hello": "world"},
		},
	},
	{
		name:   "rate_limit_request_10",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
get(state.url).as(resp, {
	"url": state.url,
	"events": [bytes(resp.Body).decode_json()],
	"status_code": resp.StatusCode,
	"header": resp.Header,
	"rate_limit": rate_limit(resp.Header, 'okta', duration('1m')),
})
`,
		},
		handler: rateLimitHandler("10", 100*time.Millisecond),
		want: []map[string]interface{}{
			{"hello": "world"},
		},
	},
	{
		name:   "rate_limit_request_10_too_slow",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
get(state.url).as(resp, {
	"url": state.url,
	"events": [bytes(resp.Body).decode_json()],
	"status_code": resp.StatusCode,
	"header": resp.Header,
	"rate_limit": rate_limit(resp.Header, 'okta', duration('1m')),
})
`,
		},
		handler: rateLimitHandler("10", 10*time.Second),
		want:    []map[string]interface{}{},
	},
	{
		name:   "retry_failure",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
get(state.url).as(resp, {
	"url": state.url,
	"events": [bytes(resp.Body).decode_json()],
	"status_code": resp.StatusCode,
	"header": resp.Header,
})
`,
		},
		handler: retryHandler(),
		want: []map[string]interface{}{
			{"hello": "world"},
		},
	},

	{
		name:   "POST_request",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
bytes(post(state.url, "application/json", '{"test":"abc"}').Body).as(body, {
	"url": state.url,
	"events": [body.decode_json()]
})
`,
		},
		handler: defaultHandler(http.MethodPost, `{"test":"abc"}`),
		want: []map[string]interface{}{
			{
				"hello": []interface{}{
					map[string]interface{}{
						"world": "moon",
					},
					map[string]interface{}{
						"space": []interface{}{
							map[string]interface{}{
								"cake": "pumpkin",
							},
						},
					},
				},
			},
		},
	},
	{
		name:   "repeated_POST_request",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": "100ms",
			"program": `
bytes(post(state.url, "application/json", '{"test":"abc"}').Body).as(body, {
	"url": state.url,
	"events": [body.decode_json()]
})
`,
		},
		handler: defaultHandler(http.MethodPost, `{"test":"abc"}`),
		want: []map[string]interface{}{
			{
				"hello": []interface{}{
					map[string]interface{}{
						"world": "moon",
					},
					map[string]interface{}{
						"space": []interface{}{
							map[string]interface{}{
								"cake": "pumpkin",
							},
						},
					},
				},
			},
			{
				"hello": []interface{}{
					map[string]interface{}{
						"world": "moon",
					},
					map[string]interface{}{
						"space": []interface{}{
							map[string]interface{}{
								"cake": "pumpkin",
							},
						},
					},
				},
			},
		},
	},
	{
		name:   "split_events",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
bytes(get(state.url).Body).as(body, {
	"events": body.decode_json().hello
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"world": "moon",
			},
			{
				"space": []interface{}{
					map[string]interface{}{
						"cake": "pumpkin",
					},
				},
			},
		},
	},
	{
		name:   "split_events_keep_parent",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
bytes(get(state.url).Body).as(body, {
	"events": body.decode_json().hello.map(e,
	{
		"hello": e
	})
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"hello": map[string]interface{}{
					"world": "moon",
				},
			},
			{
				"hello": map[string]interface{}{
					"space": []interface{}{
						map[string]interface{}{
							"cake": "pumpkin",
						},
					},
				},
			},
		},
	},
	{
		name:   "nested_split_events",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
bytes(get(state.url).Body).decode_json().as(e0, {
	"events": e0.hello.map(e1, has(e1.space) ?
		e1.space.map(e2, {
			"space": e2,
		})
	:
		[e1] // Make sure the two conditions are the same shape.
	).flatten()
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"world": "moon",
			},
			{
				"space": map[string]interface{}{
					"cake": "pumpkin",
				},
			},
		},
	},
	{
		name:   "absent_split",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
bytes(get(state.url).Body).decode_json().as(e, {
	"url": state.url,
	"events": has(e.unknown) ?
		e.unknown.map(u, {
			"unknown": u,
		})
	:
		[]
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want:    []map[string]interface{}(nil),
	},

	// Cursor/pagination tests.
	{
		name:   "date_cursor",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"state": map[string]interface{}{
				"fake_now": "2002-10-02T15:00:00Z",
			},
			"program": `
// Use terse non-standard check for presence of timestamp. The standard
// alternative is to use has(state.cursor) && has(state.cursor.timestamp).
(!is_error(state.cursor.timestamp) ?
	state.cursor.timestamp
:
	timestamp(state.fake_now)-duration('10m')
).as(time_cursor,
string(state.url).parse_url().with_replace({
	"RawQuery": {"$filter": ["alertCreationTime ge "+string(time_cursor)]}.format_query()
}).format_url().as(url, bytes(get(url).Body)).decode_json().as(event, {
	"events": [event],
	// Get the timestamp from the event if it exists, otherwise advance a little to break a request loop.
	// Due to the name of the @timestamp field, we can't use has(), so use is_error().
	"cursor": [{"timestamp": !is_error(event["@timestamp"]) ? event["@timestamp"] : time_cursor+duration('1s')}],

	// Just for testing, cycle this back into the next state.
	"fake_now": state.fake_now
}))
`,
		},
		handler: dateCursorHandler(),
		want: []map[string]interface{}{
			{"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
			{"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"},
			{"@timestamp": "2002-10-02T15:00:02Z", "foo": "bar"},
		},
		wantCursor: []map[string]interface{}{
			{"timestamp": "2002-10-02T15:00:00Z"},
			{"timestamp": "2002-10-02T15:00:01Z"},
			{"timestamp": "2002-10-02T15:00:02Z"},
		},
	},
	{
		name:   "pagination_cursor_object",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
(!is_error(state.cursor.page) ?
	state.cursor.page
:
	""
).as(page_cursor,
string(state.url).parse_url().with_replace({
	"RawQuery": (page_cursor != "" ? {"page": [page_cursor]}.format_query() : "")
}).format_url().as(url, bytes(get(url).Body)).decode_json().as(resp, {
	"events": resp.items,
	"cursor": (has(resp.nextPageToken) ? resp.nextPageToken : "").as(page, {"page": page}),
}))
`,
		},
		handler: paginationHandler(),
		want: []map[string]interface{}{
			{"foo": "a"},
			{"foo": "b"},
		},
		wantCursor: []map[string]interface{}{
			{"page": "bar"},
			{"page": ""},
		},
	},
	{
		name:   "pagination_cursor_array",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
(!is_error(state.cursor.page) ?
	state.cursor.page
:
	""
).as(page_cursor,
string(state.url).parse_url().with_replace({
	"RawQuery": (page_cursor != "" ? {"page": [page_cursor]}.format_query() : "")
}).format_url().as(url, bytes(get(url).Body)).decode_json().as(resp, {
	"events": resp.items,

	// The use of map here is to ensure the cursor is size-matched with the
	// events. In the test case all the items arrays are size 1, but this
	// may not be the case. In any case, calculate the page token only once.
	"cursor": (has(resp.nextPageToken) ? resp.nextPageToken : "").as(page, resp.items.map(e, {"page": page})),
}))
`,
		},
		handler: paginationHandler(),
		want: []map[string]interface{}{
			{"foo": "a"},
			{"foo": "b"},
		},
		wantCursor: []map[string]interface{}{
			{"page": "bar"},
			{"page": ""},
		},
	},
	{
		// This doesn't match the behaviour of the equivalent test in httpjson ("Test first
		// event"), but I am not entirely sure what the basis of that behaviour is.
		// In particular the transition {"first":"a", "foo":"b"} => {"first":"a", "foo":"c"}
		// retaining identity in "first" doesn't follow a logic that I understand.
		name:   "first_event_cursor",
		server: newTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
(!is_error(state.cursor.page) ?
	state.cursor.page
:
	""
).as(page_cursor,
string(state.url).parse_url().with_replace({
	"RawQuery": (page_cursor != "" ? {"page": [page_cursor]}.format_query() : "")
}).format_url().as(url, bytes(get(url).Body)).decode_json().as(resp, {
	"events": resp.items.map(e, e.with_update({
		"first": (!is_error(state.cursor.first) ? state.cursor.first : "none"),
	})),
	"cursor": (has(resp.nextPageToken) ? resp.nextPageToken : "").as(page, resp.items.map(e, {
		"page": page,
		"first": e.foo,
	})),
}))
`,
		},
		handler: paginationHandler(),
		want: []map[string]interface{}{
			{"first": "none", "foo": "a"},
			{"first": "a", "foo": "b"},
			{"first": "b", "foo": "c"},
			{"first": "c", "foo": "d"},
		},
		wantCursor: []map[string]interface{}{
			{"first": "a", "page": "bar"},
			{"first": "b", "page": ""},
			{"first": "c", "page": ""},
			{"first": "d", "page": ""},
		},
	},

	// Authenticated access tests.
	{
		name: "OAuth2",
		server: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
			s := httptest.NewServer(h)
			config["resource.url"] = s.URL
			config["auth.oauth2.token_url"] = s.URL + "/token"
			t.Cleanup(s.Close)
		},
		config: map[string]interface{}{
			"interval":                  1,
			"auth.oauth2.client.id":     "a_client_id",
			"auth.oauth2.client.secret": "a_client_secret",
			"auth.oauth2.endpoint_params": map[string]interface{}{
				"param1": "v1",
			},
			"auth.oauth2.scopes": []string{"scope1", "scope2"},
			"program": `
bytes(post(state.url, '', '').Body).as(body, {
	"events": body.decode_json()
})
`,
		},
		handler: oauth2Handler,
		want: []map[string]interface{}{
			{"hello": "world"},
		},
	},

	// Multi-step requests.
	{
		name:   "simple_multistep_GET_request",
		server: newChainTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
// Get the record IDs.
bytes(get(state.url).Body).decode_json().records.map(r,
	// Get each event by its ID.
	bytes(get(state.url+'/'+string(r.id)).Body).decode_json()).as(events, {
		"events": events,
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"hello": []interface{}{
					map[string]interface{}{
						"world": "moon",
					},
					map[string]interface{}{
						"space": []interface{}{
							map[string]interface{}{
								"cake": "pumpkin",
							},
						},
					},
				},
			},
		},
	},
	{
		name: "three_step_GET_request",
		server: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
			r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/":
					fmt.Fprintln(w, `{"records":[{"id":1}]}`)
				case "/1":
					fmt.Fprintln(w, `{"file_name": "file_1"}`)
				case "/file_1":
					fmt.Fprintln(w, `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`)
				}
			})
			server := httptest.NewServer(r)
			config["resource.url"] = server.URL
			t.Cleanup(server.Close)
		},
		config: map[string]interface{}{
			"interval": 1,
			"program": `
// Get the record IDs.
bytes(get(state.url).Body).decode_json().records.map(r,
	// Get the set of all files from the set of IDs.
	bytes(get(state.url+'/'+string(r.id)).Body).decode_json()).map(f,
	// Collate all the files into the events list.
	bytes(get(state.url+'/'+f.file_name).Body).decode_json()).as(events, {
		"events": events,
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"hello": []interface{}{
					map[string]interface{}{
						"world": "moon",
					},
					map[string]interface{}{
						"space": []interface{}{
							map[string]interface{}{
								"cake": "pumpkin",
							},
						},
					},
				},
			},
		},
	},

	// Programmer error.
	{
		name:   "type_error_message",
		server: newChainTestServer(httptest.NewServer),
		config: map[string]interface{}{
			"interval": 1,
			"program": `
bytes(get(state.url).Body).decode_json().records.map(r,
	bytes(get(state.url+'/'+r.id).Body).decode_json()).as(events, {
//                          ^~~~ r.id not converted to string: can't add integer to string.
		"events": events,
})
`,
		},
		handler: defaultHandler(http.MethodGet, ""),
		want: []map[string]interface{}{
			{
				"error.message": "failed eval: no such overload", // This is the best we get for some errors from CEL.
			},
		},
	},

	// not yet done from httpjson (some are redundant since they are compositional products).
	//
	// cursor/pagination (place above auth test block)
	//  Test pagination with array response
	//  Test request transforms can access state from previous transforms
	//  Test response transforms can't access request state from previous transforms
	// more chain tests (place after other chain tests)
	//  Test date cursor while using chain
	//  Test split by json objects array in chain
	//  Test split by json objects array with keep parent in chain
	//  Test nested split in chain
}

func TestInput(t *testing.T) {
	skipOnWindows := map[string]string{
		"ndjson_log_file_simple_file_scheme": "Path handling on Windows is incompatible with url.Parse/url.URL.String. See go.dev/issue/6027.",
	}
	for _, test := range inputTests {
		t.Run(test.name, func(t *testing.T) {
			if reason, skip := skipOnWindows[test.name]; runtime.GOOS == "windows" && skip {
				t.Skip(reason)
			}
			if test.server != nil {
				test.server(t, test.handler, test.config)
			}

			cfg := conf.MustNewConfigFrom(test.config)

			conf := defaultConfig()
			err := cfg.Unpack(&conf)
			if err != nil {
				t.Fatalf("unexpected error unpacking config: %v", err)
			}

			name := input{}.Name()
			if name != "cel" {
				t.Errorf(`unexpected input name: got:%q want:"cel"`, name)
			}
			src := &source{conf}
			err = input{}.Test(src, v2.TestContext{})
			if err != nil {
				t.Fatalf("unexpected error running test: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			v2Ctx := v2.Context{
				Logger:      logp.NewLogger("cel_test"),
				ID:          "test_id:" + test.name,
				Cancelation: ctx,
			}
			var client publisher
			client.done = func() {
				if len(client.published) >= len(test.want) {
					cancel()
				}
			}
			err = input{}.run(v2Ctx, src, test.persistCursor, &client)
			if fmt.Sprint(err) != fmt.Sprint(test.wantErr) {
				t.Errorf("unexpected error from running input: got:%v want:%v", err, test.wantErr)
			}
			if test.wantErr != nil {
				return
			}

			if len(client.published) < len(test.want) {
				t.Errorf("unexpected number of published events: got:%d want at least:%d", len(client.published), len(test.want))
				test.want = test.want[:len(client.published)]
			}
			client.published = client.published[:len(test.want)]
			for i, got := range client.published {
				if !reflect.DeepEqual(got.Fields, mapstr.M(test.want[i])) {
					t.Errorf("unexpected result for event %d: got:- want:+\n%s", i, cmp.Diff(got.Fields, test.want[i]))
				}
			}

			switch {
			case len(test.wantCursor) == 0 && len(client.cursors) == 0:
				return
			case len(test.wantCursor) == 0:
				t.Errorf("unexpected cursors: %v", client.cursors)
				return
			}
			if len(client.cursors) < len(test.wantCursor) {
				t.Errorf("unexpected number of cursors events: got:%d want at least:%d", len(client.cursors), len(test.wantCursor))
				test.wantCursor = test.wantCursor[:len(client.published)]
			}
			client.published = client.published[:len(test.want)]
			for i, got := range client.cursors {
				if !reflect.DeepEqual(mapstr.M(got), mapstr.M(test.wantCursor[i])) {
					t.Errorf("unexpected cursor for event %d: got:- want:+\n%s", i, cmp.Diff(got, test.wantCursor[i]))
				}
			}
		})
	}
}

var _ inputcursor.Publisher = (*publisher)(nil)

type publisher struct {
	done      func()
	mu        sync.Mutex
	published []beat.Event
	cursors   []map[string]interface{}
}

func (p *publisher) Publish(e beat.Event, cursor interface{}) error {
	p.mu.Lock()
	p.published = append(p.published, e)
	if cursor != nil {
		c, ok := cursor.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid cursor type for testing: %T", cursor)
		}
		p.cursors = append(p.cursors, c)
	}
	p.done()
	p.mu.Unlock()
	return nil
}

func fileSchemePath(path string) string {
	p, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

// missingFileError returns the string of an error for opening a file. This is needed
// for cross-platform testing since Windows returns a different error string.
func missingFileError(path string) string {
	f, err := os.Open(path)
	if err == nil {
		f.Close()
	}
	return fmt.Sprint(err)
}

func newTestServer(serve func(http.Handler) *httptest.Server) func(*testing.T, http.HandlerFunc, map[string]interface{}) {
	return func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
		server := serve(h)
		config["resource.url"] = server.URL
		t.Cleanup(server.Close)
	}
}

func newChainTestServer(serve func(http.Handler) *httptest.Server) func(*testing.T, http.HandlerFunc, map[string]interface{}) {
	return func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
		r := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				fmt.Fprintln(w, `{"records":[{"id":1}]}`)
			case "/1":
				fmt.Fprintln(w, `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`)
			}
		})
		server := httptest.NewServer(r)
		config["resource.url"] = server.URL
		t.Cleanup(server.Close)
	}
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("httpjson_test"),
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
}

//nolint:errcheck // No point checking errors in test server.
func defaultHandler(expectedMethod, expectedBody string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		msg := `{"hello":[{"world":"moon"},{"space":[{"cake":"pumpkin"}]}]}`
		switch {
		case r.Method != expectedMethod:
			w.WriteHeader(http.StatusBadRequest)
			msg = fmt.Sprintf(`{"error":"expected method was %q"}`, expectedMethod)
		case expectedBody != "":
			body, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if expectedBody != string(body) {
				w.WriteHeader(http.StatusBadRequest)
				msg = fmt.Sprintf(`{"error":"expected body was %q"}`, expectedBody)
			}
		}

		w.Write([]byte(msg))
	}
}

//nolint:errcheck // No point checking errors in test server.
func retryAfterHandler(after string) http.HandlerFunc {
	var isRetry bool
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if isRetry {
			w.Write([]byte(`{"hello":"world"}`))
			return
		}
		w.Header().Set("Retry-After", after)
		w.WriteHeader(http.StatusTooManyRequests)
		isRetry = true
		w.Write([]byte(`{"error":"too many requests"}`))
	}
}

//nolint:errcheck // No point checking errors in test server.
func rateLimitHandler(limit string, wait time.Duration) http.HandlerFunc {
	var isRetry bool
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if isRetry {
			w.Write([]byte(`{"hello":"world"}`))
			return
		}
		w.Header().Set("X-Rate-Limit-Limit", limit)
		w.Header().Set("X-Rate-Limit-Remaining", "0")
		w.Header().Set("X-Rate-Limit-Reset", fmt.Sprint(time.Now().Add(wait).Unix()))
		w.WriteHeader(http.StatusTooManyRequests)
		isRetry = true
		w.Write([]byte(`{"error":"too many requests"}`))
	}
}

//nolint:errcheck // No point checking errors in test server.
func retryHandler() http.HandlerFunc {
	var count int
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		if count == 2 {
			w.Write([]byte(`{"hello":"world"}`))
			return
		}
		w.WriteHeader(rand.Intn(100) + 500)
		count++
	}
}

//nolint:errcheck // No point checking errors in test server.
func oauth2Handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/token" {
		oauth2TokenHandler(w, r)
		return
	}

	w.Header().Set("content-type", "application/json")
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong method"}`))
	case r.Header.Get("Authorization") != "Bearer abcd":
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong bearer"}`))
	default:
		w.Write([]byte(`{"hello":"world"}`))
	}
}

//nolint:errcheck // No point checking errors in test server.
func oauth2TokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	r.ParseForm()
	switch {
	case r.Method != http.MethodPost:
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong method"}`))
	case r.FormValue("grant_type") != "client_credentials":
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong grant_type"}`))
	case r.FormValue("client_id") != "a_client_id":
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong client_id"}`))
	case r.FormValue("client_secret") != "a_client_secret":
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong client_secret"}`))
	case r.FormValue("scope") != "scope1 scope2":
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong scope"}`))
	case r.FormValue("param1") != "v1":
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"wrong param1"}`))
	default:
		w.Write([]byte(`{"token_type": "Bearer", "expires_in": "60", "access_token": "abcd"}`))
	}
}

//nolint:errcheck // No point checking errors in test server.
func dateCursorHandler() http.HandlerFunc {
	var count int
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch count {
		case 0:
			if q := r.URL.Query().Get("$filter"); q != "alertCreationTime ge 2002-10-02T14:50:00Z" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"wrong initial cursor value: ` + q + `"}`))
				return
			}
			w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:00Z","foo":"bar"}`))
		case 1:
			if q := r.URL.Query().Get("$filter"); q != "alertCreationTime ge 2002-10-02T15:00:00Z" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"wrong cursor value: ` + q + `"}`))
				return
			}
			w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:01Z","foo":"bar"}`))
		case 2:
			if q := r.URL.Query().Get("$filter"); q != "alertCreationTime ge 2002-10-02T15:00:01Z" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"wrong cursor value: ` + q + `"}`))
				return
			}
			w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:02Z","foo":"bar"}`))
		}
		count++
	}
}

//nolint:errcheck // No point checking errors in test server.
func paginationHandler() http.HandlerFunc {
	var count int
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch count {
		case 0:
			w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:00Z","nextPageToken":"bar","items":[{"foo":"a"}]}`))
		case 1:
			if r.URL.Query().Get("page") != "bar" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"wrong page token value"}`))
				return
			}
			w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:01Z","items":[{"foo":"b"}]}`))
		case 2:
			w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:02Z","items":[{"foo":"c"}]}`))
		case 3:
			w.Write([]byte(`{"@timestamp":"2002-10-02T15:00:03Z","items":[{"foo":"d"}]}`))
		}
		count++
	}
}

//nolint:errcheck // No point checking errors in test server.
func paginationArrayHandler() http.HandlerFunc {
	var count int
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch count {
		case 0:
			w.Write([]byte(`[{"nextPageToken":"bar","foo":"bar"},{"foo":"bar"}]`))
		case 1:
			if r.URL.Query().Get("page") != "bar" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"wrong page token value"}`))
				return
			}
			w.Write([]byte(`[{"foo":"bar"}]`))
		}
		count++
	}
}
