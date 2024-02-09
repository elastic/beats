// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	basicToken   = "dXNlcjpwYXNz"
	bearerToken  = "BXNlcjpwYVVz"
	customHeader = "X-Api-Key"
	customValue  = "my-api-key"
)

// WebSocketHandler is a type for handling WebSocket messages.
type WebSocketHandler func(*testing.T, *websocket.Conn, []string)

var inputTests = []struct {
	name          string
	server        func(*testing.T, WebSocketHandler, map[string]interface{}, []string)
	handler       WebSocketHandler
	config        map[string]interface{}
	response      []string
	time          func() time.Time
	persistCursor map[string]interface{}
	want          []map[string]interface{}
	wantErr       error
}{
	{
		name:    "single_event",
		server:  newWebSocketTestServer(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
		},
		response: []string{`
			{
				"pps": {
					"agent": "example.proofpoint.com",
					"cid": "mmeng_uivm071"
				},
				"ts": "2017-08-17T14:54:12.949180-07:00",
				"data": "2017-08-17T14:54:12.949180-07:00 example sendmail[30641]:v7HLqYbx029423: to=/dev/null, ctladdr=<user1@example.com> (8/0),delay=00:00:00, xdelay=00:00:00, mailer=*file*, tls_verify=NONE, pri=35342,dsn=2.0.0, stat=Sent",
				"sm": {
					"tls": {
						"verify": "NONE"
					},
					"stat": "Sent",
					"qid": "v7HLqYbx029423",
					"dsn": "2.0.0",
					"mailer": "*file*",
					"to": [
						"/dev/null"
					],
					"ctladdr": "<user1@example.com> (8/0)",
					"delay": "00:00:00",
					"xdelay": "00:00:00",
					"pri": 35342
				},
				"id": "ZeYGULpZmL5N0151HN1OyA"
		   }`},
		want: []map[string]interface{}{
			{
				"pps": map[string]interface{}{
					"agent": "example.proofpoint.com",
					"cid":   "mmeng_uivm071",
				},
				"ts":   "2017-08-17T14:54:12.949180-07:00",
				"data": "2017-08-17T14:54:12.949180-07:00 example sendmail[30641]:v7HLqYbx029423: to=/dev/null, ctladdr=<user1@example.com> (8/0),delay=00:00:00, xdelay=00:00:00, mailer=*file*, tls_verify=NONE, pri=35342,dsn=2.0.0, stat=Sent",
				"sm": map[string]interface{}{
					"tls": map[string]interface{}{
						"verify": "NONE",
					},
					"stat":   "Sent",
					"qid":    "v7HLqYbx029423",
					"dsn":    "2.0.0",
					"mailer": "*file*",
					"to": []interface{}{
						"/dev/null",
					},
					"ctladdr": "<user1@example.com> (8/0)",
					"delay":   "00:00:00",
					"xdelay":  "00:00:00",
					"pri":     float64(35342),
				},
				"id": "ZeYGULpZmL5N0151HN1OyA",
			},
		},
	},
	{
		name:    "multiple_events",
		server:  newWebSocketTestServer(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
		},
		response: []string{`
			{
				"pps": {
					"agent": "example.proofpoint.com",
					"cid": "mmeng_uivm071"
				},
				"ts": "2017-08-17T14:54:12.949180-07:00",
				"data": "2017-08-17T14:54:12.949180-07:00 example sendmail[30641]:v7HLqYbx029423: to=/dev/null, ctladdr=<user1@example.com> (8/0),delay=00:00:00, xdelay=00:00:00, mailer=*file*, tls_verify=NONE, pri=35342,dsn=2.0.0, stat=Sent",
				"sm": {
					"tls": {
						"verify": "NONE"
					},
					"stat": "Sent",
					"qid": "v7HLqYbx029423",
					"dsn": "2.0.0",
					"mailer": "*file*",
					"to": [
						"/dev/null"
					],
					"ctladdr": "<user1@example.com> (8/0)",
					"delay": "00:00:00",
					"xdelay": "00:00:00",
					"pri": 35342
				},
				"id": "ZeYGULpZmL5N0151HN1OyA"
		   }`,
			`{
				"pps": {
					"agent": "example.proofpoint.com",
					"cid": "mmeng_uivm071"
				},
				"ts": "2017-08-17T14:54:12.949180-07:00",
				"data": "2017-08-17T14:54:12.949180-07:00 example sendmail[30641]:v7HLqYbx029423: to=/dev/null, ctladdr=<user1@example.com> (8/0),delay=00:00:00, xdelay=00:00:00, mailer=*file*, tls_verify=NONE, pri=35342,dsn=2.0.0, stat=Sent",
				"sm": {
					"tls": {
						"verify": "NONE"
					},
					"stat": "Sent",
					"qid": "v7HLqYbx029423",
					"dsn": "2.0.0",
					"mailer": "*file*",
					"to": [
						"/dev/null"
					],
					"ctladdr": "<user1@example.com> (8/0)",
					"delay": "00:00:00",
					"xdelay": "00:00:00",
					"pri": 35342
				},
				"id": "ZeYGULpZmL5N0151HN1OyX"
	   }`},
		want: []map[string]interface{}{
			{
				"pps": map[string]interface{}{
					"agent": "example.proofpoint.com",
					"cid":   "mmeng_uivm071",
				},
				"ts":   "2017-08-17T14:54:12.949180-07:00",
				"data": "2017-08-17T14:54:12.949180-07:00 example sendmail[30641]:v7HLqYbx029423: to=/dev/null, ctladdr=<user1@example.com> (8/0),delay=00:00:00, xdelay=00:00:00, mailer=*file*, tls_verify=NONE, pri=35342,dsn=2.0.0, stat=Sent",
				"sm": map[string]interface{}{
					"tls": map[string]interface{}{
						"verify": "NONE",
					},
					"stat":   "Sent",
					"qid":    "v7HLqYbx029423",
					"dsn":    "2.0.0",
					"mailer": "*file*",
					"to": []interface{}{
						"/dev/null",
					},
					"ctladdr": "<user1@example.com> (8/0)",
					"delay":   "00:00:00",
					"xdelay":  "00:00:00",
					"pri":     float64(35342),
				},
				"id": "ZeYGULpZmL5N0151HN1OyA",
			},
			{
				"pps": map[string]interface{}{
					"agent": "example.proofpoint.com",
					"cid":   "mmeng_uivm071",
				},
				"ts":   "2017-08-17T14:54:12.949180-07:00",
				"data": "2017-08-17T14:54:12.949180-07:00 example sendmail[30641]:v7HLqYbx029423: to=/dev/null, ctladdr=<user1@example.com> (8/0),delay=00:00:00, xdelay=00:00:00, mailer=*file*, tls_verify=NONE, pri=35342,dsn=2.0.0, stat=Sent",
				"sm": map[string]interface{}{
					"tls": map[string]interface{}{
						"verify": "NONE",
					},
					"stat":   "Sent",
					"qid":    "v7HLqYbx029423",
					"dsn":    "2.0.0",
					"mailer": "*file*",
					"to": []interface{}{
						"/dev/null",
					},
					"ctladdr": "<user1@example.com> (8/0)",
					"delay":   "00:00:00",
					"xdelay":  "00:00:00",
					"pri":     float64(35342),
				},
				"id": "ZeYGULpZmL5N0151HN1OyX",
			},
		},
	},
	{
		name:    "bad_cursor",
		server:  newWebSocketTestServer(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
					"cursor":["What's next?"],
				})`,
		},
		response: []string{`
			 {
				"pps": {
					"agent": "example.proofpoint.com",
					"cid": "mmeng_uivm071"
				},
			}`},
		wantErr: fmt.Errorf("unexpected type returned for evaluation cursor element: %T", "What's next?"),
	},
	{
		name:    "invalid_url_scheme",
		server:  invalidWebSocketTestServer(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
		},
		wantErr: fmt.Errorf("unsupported scheme: http accessing config"),
	},
	{
		name:    "cursor_condition_check",
		server:  newWebSocketTestServer(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
                bytes(state.response).decode_json().as(inner_body,{
					"events": has(state.cursor) && inner_body.ts > state.cursor.last_updated ?  [inner_body] : [], 
            })`,
			"state": map[string]interface{}{
				"cursor": map[string]int{
					"last_updated": 1502908200,
				},
			},
		},
		response: []string{`
         {
            "pps": {
                "agent": "example.proofpoint.com",
                "cid": "mmeng_uivm071"
            },
            "ts": 1502908200
        }`,
			`{
            "pps": {
                "agent": "example.proofpoint-1.com",
                "cid": "mmeng_vxciml"
            },
            "ts": 1503081000
        }`,
		},
		want: []map[string]interface{}{
			{
				"pps": map[string]interface{}{
					"agent": "example.proofpoint-1.com",
					"cid":   "mmeng_vxciml",
				},
				"ts": float64(1503081000),
			},
		},
	},
	{
		name:    "auth_basic_token",
		server:  webSocketTestServerWithAuth(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"auth": map[string]interface{}{
				"basic_token": basicToken,
			},
		},
		response: []string{`
         {
            "pps": {
                "agent": "example.proofpoint.com",
                "cid": "mmeng_uivm071"
            },
            "ts": 1502908200
        }`,
		},
		want: []map[string]interface{}{
			{
				"pps": map[string]interface{}{
					"agent": "example.proofpoint.com",
					"cid":   "mmeng_uivm071",
				},
				"ts": float64(1502908200),
			},
		},
	},
	{
		name:    "auth_bearer_token",
		server:  webSocketTestServerWithAuth(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"auth": map[string]interface{}{
				"bearer_token": bearerToken,
			},
		},
		response: []string{`
         {
            "pps": {
                "agent": "example.proofpoint.com",
                "cid": "mmeng_uivm071"
            },
            "ts": 1502908200
        }`,
		},
		want: []map[string]interface{}{
			{
				"pps": map[string]interface{}{
					"agent": "example.proofpoint.com",
					"cid":   "mmeng_uivm071",
				},
				"ts": float64(1502908200),
			},
		},
	},
	{
		name:    "auth_custom",
		server:  webSocketTestServerWithAuth(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"auth": map[string]interface{}{
				"custom": map[string]interface{}{
					"header": customHeader,
					"value":  customValue,
				},
			},
		},
		response: []string{`
         {
            "pps": {
                "agent": "example.proofpoint.com",
                "cid": "mmeng_uivm071"
            },
            "ts": 1502908200
        }`,
		},
		want: []map[string]interface{}{
			{
				"pps": map[string]interface{}{
					"agent": "example.proofpoint.com",
					"cid":   "mmeng_uivm071",
				},
				"ts": float64(1502908200),
			},
		},
	},
}

func TestInput(t *testing.T) {
	// tests will ignore context cancelled errors, since they are expected
	ctxCancelledError := fmt.Errorf("context canceled")
	logp.TestingSetup()
	for _, test := range inputTests {
		t.Run(test.name, func(t *testing.T) {
			if test.server != nil {
				test.server(t, test.handler, test.config, test.response)
			}

			cfg := conf.MustNewConfigFrom(test.config)

			conf := config{}
			conf.Redact = &redact{} // Make sure we pass the redact requirement.
			err := cfg.Unpack(&conf)
			if err != nil {
				if test.wantErr != nil {
					assert.EqualError(t, err, test.wantErr.Error())
					return
				}
				t.Fatalf("unexpected error unpacking config: %v", err)
			}

			name := input{}.Name()
			if name != "websocket" {
				t.Errorf(`unexpected input name: got:%q want:"websocket"`, name)
			}
			src := &source{conf}
			err = input{}.Test(src, v2.TestContext{})
			if err != nil {
				t.Fatalf("unexpected error running test: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Second)
			defer cancel()

			v2Ctx := v2.Context{
				Logger:      logp.NewLogger("websocket_test"),
				ID:          "test_id:" + test.name,
				Cancelation: ctx,
			}
			var client publisher
			client.done = func() {
				if len(client.published) >= len(test.want) {
					cancel()
				}
			}

			err = input{test.time, conf}.run(v2Ctx, src, test.persistCursor, &client)
			if (fmt.Sprint(err) != fmt.Sprint(ctxCancelledError)) && (fmt.Sprint(err) != fmt.Sprint(test.wantErr)) {
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
					t.Errorf("unexpected result for event %d: got:- want:+\n%s", i, cmp.Diff(got.Fields, mapstr.M(test.want[i])))
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

func newWebSocketTestServer(serve func(http.Handler) *httptest.Server) func(*testing.T, WebSocketHandler, map[string]interface{}, []string) {
	return func(t *testing.T, handler WebSocketHandler, config map[string]interface{}, response []string) {
		server := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			}

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("error upgrading connection to WebSocket: %v", err)
				return
			}

			handler(t, conn, response)
		}))
		// only set the resource URL if it is not already set
		if config["url"] == nil {
			config["url"] = "ws" + server.URL[4:]
		}
		t.Cleanup(server.Close)
	}
}

// invalidWebSocketTestServer returns a function that creates a WebSocket server with an invalid URL scheme.
func invalidWebSocketTestServer(serve func(http.Handler) *httptest.Server) func(*testing.T, WebSocketHandler, map[string]interface{}, []string) {
	return func(t *testing.T, handler WebSocketHandler, config map[string]interface{}, response []string) {
		server := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			}

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("error upgrading connection to WebSocket: %v", err)
				return
			}

			handler(t, conn, response)
		}))
		config["url"] = server.URL
		t.Cleanup(server.Close)
	}
}

// webSocketTestServerWithAuth returns a function that creates a WebSocket server with authentication. This does not however simulate a TLS connection.
func webSocketTestServerWithAuth(serve func(http.Handler) *httptest.Server) func(*testing.T, WebSocketHandler, map[string]interface{}, []string) {
	return func(t *testing.T, handler WebSocketHandler, config map[string]interface{}, response []string) {
		server := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					// check for auth token
					authToken := r.Header.Get("Authorization")
					if authToken == "" {
						authToken = r.Header.Get(customHeader)
						if authToken == "" {
							return false
						}
					}

					switch {
					case authToken == "Bearer "+bearerToken:
						return true
					case authToken == "Basic "+basicToken:
						return true
					case authToken == customValue:
						return true
					default:
						return false

					}
				},
			}

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("error upgrading connection to WebSocket: %v", err)
				return
			}

			handler(t, conn, response)
		}))
		// only set the resource URL if it is not already set
		if config["url"] == nil {
			config["url"] = "ws" + server.URL[4:]
		}
		t.Cleanup(server.Close)
	}
}

// defaultHandler is a default handler for WebSocket connections.
func defaultHandler(t *testing.T, conn *websocket.Conn, response []string) {
	for _, r := range response {
		err := conn.WriteMessage(websocket.TextMessage, []byte(r))
		if err != nil {
			t.Fatalf("error writing message to WebSocket: %v", err)
		}
	}
}
