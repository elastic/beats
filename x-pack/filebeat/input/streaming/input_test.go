// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"crypto/tls"
	"errors"
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
	"github.com/elastic/beats/v7/testing/testutils"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

//nolint:gosec // These are test tokens and are not used in production code.
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
	proxyServer   func(*testing.T, WebSocketHandler, map[string]interface{}, []string) *httptest.Server
	oauth2Server  func(*testing.T, http.HandlerFunc, map[string]interface{})
	handler       WebSocketHandler
	oauth2Handler http.HandlerFunc
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
	{
		name:    "test_retry_success",
		server:  webSocketServerWithRetry(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"retry": map[string]interface{}{
				"max_attempts": 3,
				"wait_min":     "1s",
				"wait_max":     "2s",
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
		name:    "test_retry_failure",
		server:  webSocketServerWithRetry(httptest.NewServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"retry": map[string]interface{}{
				"max_attempts": 2,
				"wait_min":     "1s",
				"wait_max":     "2s",
			},
		},
		wantErr: fmt.Errorf("failed to establish WebSocket connection after 2 attempts with error websocket: bad handshake and (status 403)"),
	},
	{
		name:    "single_event_tls",
		server:  webSocketServerWithTLS(httptest.NewUnstartedServer),
		handler: defaultHandler,
		config: map[string]interface{}{
			"program": `
					bytes(state.response).decode_json().as(inner_body,{
					"events": [inner_body],
				})`,
			"ssl": map[string]interface{}{
				"enabled":                 true,
				"certificate_authorities": []string{"testdata/certs/ca.crt"},
				"certificate":             "testdata/certs/cert.pem",
				"key":                     "testdata/certs/key.pem",
			},
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
		name:        "basic_proxy_forwarding",
		proxyServer: newWebSocketProxyTestServer,
		handler:     defaultHandler,
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
		name: "oauth2_blank_auth_style",
		oauth2Server: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
			s := httptest.NewServer(h)
			config["auth.token_url"] = s.URL + "/token"
			config["url"] = "ws://placeholder"
			t.Cleanup(s.Close)
		},
		oauth2Handler: oauth2TokenHandler,
		server:        webSocketTestServerWithAuth(httptest.NewServer),
		handler:       defaultHandler,
		config: map[string]interface{}{
			"auth": map[string]interface{}{
				"client_id":     "a_client_id",
				"client_secret": "a_client_secret",
				"scopes": []string{
					"scope1",
					"scope2",
				},
				"endpoint_params": map[string]string{
					"param1": "v1",
				},
			},
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
		name: "oauth2_in_params_auth_style",
		oauth2Server: func(t *testing.T, h http.HandlerFunc, config map[string]interface{}) {
			s := httptest.NewServer(h)
			config["auth.token_url"] = s.URL + "/token"
			config["url"] = "ws://placeholder"
			t.Cleanup(s.Close)
		},
		oauth2Handler: oauth2TokenHandler,
		server:        webSocketTestServerWithAuth(httptest.NewServer),
		handler:       defaultHandler,
		config: map[string]interface{}{
			"auth": map[string]interface{}{
				"auth_style":    "in_params",
				"client_id":     "a_client_id",
				"client_secret": "a_client_secret",
				"scopes": []string{
					"scope1",
					"scope2",
				},
				"endpoint_params": map[string]string{
					"param1": "v1",
				},
			},
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
}

var urlEvalTests = []struct {
	name   string
	config map[string]interface{}
	time   func() time.Time
	want   string
}{
	{
		name: "cursor based url modification",
		config: map[string]interface{}{
			"url":         "ws://testapi/getresults",
			"url_program": `has(state.cursor) && has(state.cursor.since) ? state.url+"?since="+ state.cursor.since : state.url`,
			"state": map[string]interface{}{
				"cursor": map[string]interface{}{
					"since": "2017-08-17T14:54:12",
				},
			},
		},
		want: "ws://testapi/getresults?since=2017-08-17T14:54:12",
	},
	{
		name: "cursor based url modification using simplified query",
		config: map[string]interface{}{
			"url":         "ws://testapi/getresults",
			"url_program": `state.url + "?since=" + state.?cursor.since.orValue(state.url)`,
			"state": map[string]interface{}{
				"cursor": map[string]interface{}{
					"since": "2017-08-17T14:54:12",
				},
			},
		},
		want: "ws://testapi/getresults?since=2017-08-17T14:54:12",
	},
	{
		name: "url modification with no cursor",
		config: map[string]interface{}{
			"url":         "ws://testapi/getresults",
			"url_program": `has(state.cursor) && has(state.cursor.since) ? state.url+"?since="+ state.cursor.since: state.url+"?since="+ state.initial_start_time`,
			"state": map[string]interface{}{
				"initial_start_time": "2022-01-01T00:00:00Z",
			},
		},
		want: "ws://testapi/getresults?since=2022-01-01T00:00:00Z",
	},
	{
		name: "url modification with no cursor, using simplified query",
		config: map[string]interface{}{
			"url":         "ws://testapi/getresults",
			"url_program": `state.url + "?since=" + state.?cursor.since.orValue(state.initial_start_time)`,
			"state": map[string]interface{}{
				"initial_start_time": "2022-01-01T00:00:00Z",
			},
		},
		want: "ws://testapi/getresults?since=2022-01-01T00:00:00Z",
	},
}

func TestURLEval(t *testing.T) {
	logp.TestingSetup()
	for _, test := range urlEvalTests {
		t.Run(test.name, func(t *testing.T) {

			cfg := conf.MustNewConfigFrom(test.config)

			conf := config{}
			conf.Redact = &redact{}
			err := cfg.Unpack(&conf)
			if err != nil {
				t.Fatalf("unexpected error unpacking config: %v", err)
			}

			name := input{}.Name()
			if name != "streaming" {
				t.Errorf(`unexpected input name: got:%q want:"streaming"`, name)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var state map[string]interface{}
			if conf.State == nil {
				state = make(map[string]interface{})
			} else {
				state = conf.State
			}

			now := test.time
			if now == nil {
				now = time.Now
			}
			response, err := getURL(ctx, "websocket", conf.URLProgram, conf.URL.String(), state, conf.Redact, logp.NewLogger("websocket_url_eval_test"), now)
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("unexpected error from running input: got:%v want:%v", err, nil)
			}

			assert.Equal(t, test.want, response)
		})
	}
}

func TestInput(t *testing.T) {
	testutils.SkipIfFIPSOnly(t, "websocket setup requires SHA-1.")
	// tests will ignore context cancelled errors, since they are expected
	ctxCancelledError := fmt.Errorf("context canceled")
	logp.TestingSetup()
	for _, test := range inputTests {
		t.Run(test.name, func(t *testing.T) {
			if test.oauth2Server != nil {
				test.oauth2Server(t, test.oauth2Handler, test.config)
			}
			if test.server != nil {
				test.server(t, test.handler, test.config, test.response)
			}
			if test.proxyServer != nil {
				test.proxyServer(t, test.handler, test.config, test.response)
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
			if name != "streaming" {
				t.Errorf(`unexpected input name: got:%q want:"streaming"`, name)
			}
			src := &source{conf}
			err = input{}.Test(src, v2.TestContext{})
			if err != nil {
				t.Fatalf("unexpected error running test: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

			err = input{time: test.time, cfg: conf}.run(v2Ctx, src, test.persistCursor, &client)
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
		if config["url"] == nil || config["url"] == "ws://placeholder" {
			config["url"] = "ws" + server.URL[4:]
		}
		t.Cleanup(server.Close)
	}
}

// webSocketServerWithRetry returns a function that creates a WebSocket server that rejects the first two connection attempts and accepts the third.
func webSocketServerWithRetry(serve func(http.Handler) *httptest.Server) func(*testing.T, WebSocketHandler, map[string]interface{}, []string) {
	var attempt int
	return func(t *testing.T, handler WebSocketHandler, config map[string]interface{}, response []string) {
		server := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempt++
			if attempt <= 2 {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprintf(w, "connection attempt %d rejected", attempt)
				return
			}
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

// webSocketServerWithTLS simulates a WebSocket server with TLS based authentication.
func webSocketServerWithTLS(serve func(http.Handler) *httptest.Server) func(*testing.T, WebSocketHandler, map[string]interface{}, []string) {
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
		//nolint:gosec // there is no need to use a secure cert for testing
		server.TLS = &tls.Config{
			Certificates: []tls.Certificate{generateSelfSignedCert(t)},
		}
		server.StartTLS()

		if config["url"] == nil {
			config["url"] = "ws" + server.URL[4:]
		}
		t.Cleanup(server.Close)
	}
}

// generateSelfSignedCert returns a self-signed certificate for testing purposes based on the dummy certs in the testdata directory
func generateSelfSignedCert(t *testing.T) tls.Certificate {
	cert, err := tls.LoadX509KeyPair("testdata/certs/cert.pem", "testdata/certs/key.pem")
	if err != nil {
		t.Fatalf("failed to generate self-signed cert: %v", err)
	}
	return cert
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

// webSocketTestServer creates a WebSocket target server that communicates with the proxy handler.
func webSocketTestServer(t *testing.T, handler WebSocketHandler, response []string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("failed to upgrade WebSocket connection: %v", err)
			return
		}
		handler(t, conn, response)
	}))
}

// webSocketProxyHandler forwards WebSocket connections to the target server.
//
//nolint:errcheck //we can safely ignore errors checks here
func webSocketProxyHandler(targetURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Response.Body.Close()
		//nolint:bodyclose // we can ignore the body close here
		targetConn, _, err := websocket.DefaultDialer.Dial(targetURL, nil)
		if err != nil {
			http.Error(w, "failed to connect to backend WebSocket server", http.StatusBadGateway)
			return
		}
		defer targetConn.Close()

		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			http.Error(w, "failed to upgrade client connection", http.StatusInternalServerError)
			return
		}
		defer clientConn.Close()
		// forward messages between client and target server
		go func() {
			for {
				messageType, message, err := targetConn.ReadMessage()
				if err != nil {
					break
				}
				clientConn.WriteMessage(messageType, message)
			}
		}()
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				break
			}
			targetConn.WriteMessage(messageType, message)
		}
	}
}

// newWebSocketProxyTestServer creates a proxy server forwarding WebSocket traffic.
func newWebSocketProxyTestServer(t *testing.T, handler WebSocketHandler, config map[string]interface{}, response []string) *httptest.Server {
	backendServer := webSocketTestServer(t, handler, response)
	config["url"] = "ws" + backendServer.URL[4:]
	config["proxy_url"] = "ws" + backendServer.URL[4:]
	return httptest.NewServer(webSocketProxyHandler(config["url"].(string)))
}

//nolint:errcheck // no point checking errors in test server.
func oauth2TokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/token" {
		return
	}
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
		w.Write([]byte(`{"token_type": "Bearer", "expires_in": "3600", "access_token": "` + bearerToken + `"}`))
	}
}
