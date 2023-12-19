// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestFormQueryWithCursor(t *testing.T) {
	mockTimeNow(time.Date(2023, time.May, 18, 12, 0, 0, 0, time.UTC))
	t.Cleanup(resetTimeNow)

	tests := map[string]struct {
		initialInterval     time.Duration
		defaultSOQLTemplate string
		valueSOQLTemplate   string
		wantQuery           string
		cursor              *state
		wantErr             error
	}{
		"valid soql templates with nil cursor": { // expect default query with LogDate > initialInterval
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 60 days (2 months)
			defaultSOQLTemplate: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ .var.initial_interval ]] ORDER BY CreatedDate ASC NULLS FIRST",
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > 2023-03-19T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              &state{},
		},
		"valid soql templates with non-empty .cursor.logdate": { // expect value SOQL query with .cursor.logdate set
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 60 days (2 months)
			defaultSOQLTemplate: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ .var.initial_interval ]] ORDER BY CreatedDate ASC NULLS FIRST",
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > 2023-05-18T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              &state{LogDateTime: timeNow().Format(formatRFC3339Like)},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			v1, v2 := &valueTpl{}, &valueTpl{}

			err := v1.Unpack(tc.defaultSOQLTemplate)
			assert.NoError(t, err)

			err = v2.Unpack(tc.valueSOQLTemplate)
			assert.NoError(t, err)

			queryConfig := &QueryConfig{
				Default: v1,
				Value:   v2,
			}

			sfInput := &salesforceInput{
				config: config{InitialInterval: tc.initialInterval},
				log:    logp.L().With("hello", "world"),
				cursor: tc.cursor,
			}

			querier, err := sfInput.FormQueryWithCursor(queryConfig)
			assert.NoError(t, err)

			assert.EqualValues(t, tc.wantQuery, querier.Query)
		})
	}
}

// sample:
/*
map[string]interface{}{
	"data_collection_method": map[string]interface{}{
		"event_log_file": map[string]interface{}{
			"interval": "1h",
			"enabled":  true,
			"query": map[string]interface{}{
				"default": "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' ORDER BY CreatedDate ASC NULLS FIRST",
				"value":   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			},
			"cursor": map[string]interface{}{
				"field": "CreatedDate",
			},
		},
		"object": map[string]interface{}{
			"interval": "5m",
			"enabled":  true,
			"query": map[string]interface{}{
				"default": "SELECT FIELDS(STANDARD) FROM LoginEvent",
				"value":   "SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > [[ .cursor.logdate ]]",
			},
			"cursor": map[string]interface{}{
				"field": "EventDate",
			},
		},
	},
},
*/

var testCases = []struct {
	name         string
	setupServer  func(testing.TB, http.HandlerFunc, map[string]interface{})
	baseConfig   map[string]interface{}
	handler      http.HandlerFunc
	expected     []string
	expectedFile string
	wantErr      bool

	skipReason string
}{
	{
		name:        "test_data_collection_method_object_with_default_query_only",
		setupServer: newTestServer(httptest.NewServer),
		baseConfig: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"enabled":       true,
				"client.id":     "clientid",
				"client.secret": "clientsecret",
				"token_url":     "https://instance_id.develop.my.salesforce.com/services/oauth2/token",
				"user":          "username",
				"password":      "password",
			},
			"version": 56,
			"data_collection_method": map[string]interface{}{
				"object": map[string]interface{}{
					"interval": "5m",
					"enabled":  true,
					"query": map[string]interface{}{
						"default": "SELECT FIELDS(STANDARD) FROM LoginEvent",
						"value":   "SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > [[ .cursor.logdate ]]",
					},
					"cursor": map[string]interface{}{
						"field": "EventDate",
					},
				},
			},
		},
		handler:  defaultHandler("GET", "", `{"AdditionalInfo":"{}","ApiType":"N/A","ApiVersion":"N/A","Application":"salesforce_test","Browser":"Unknown","CipherSuite":"ECDHE-RSA-AES256-GCM-SHA384","City":"Mumbai","ClientVersion":"N/A","Country":"India","CountryIso":"IN","CreatedDate":"2023-12-06T05:44:34.942+0000","EvaluationTime":0,"EventDate":"2023-12-06T05:44:24.973+0000","EventIdentifier":"00044326-ed4a-421a-a0a8-e62ea626f3af","HttpMethod":"POST","Id":"000000000000000AAA","LoginGeoId":"04F5j00003NvV1cEAF","LoginHistoryId":"0Ya5j00003k2scQCAQ","LoginKey":"pgOVoLbV96U9o08W","LoginLatitude":19.0748,"LoginLongitude":72.8856,"LoginType":"Remote Access 2.0","LoginUrl":"login.salesforce.com","Platform":"Unknown","PostalCode":"400070","SessionLevel":"STANDARD","SourceIp":"134.238.252.19","Status":"Success","Subdivision":"Maharashtra","TlsProtocol":"TLS 1.2","UserId":"0055j00000AT6I1AAL","UserType":"Standard","Username":"salesforceinstance@devtest.in"}`),
		expected: []string{`{"AdditionalInfo":"{}","ApiType":"N/A","ApiVersion":"N/A","Application":"salesforce_test","Browser":"Unknown","CipherSuite":"ECDHE-RSA-AES256-GCM-SHA384","City":"Mumbai","ClientVersion":"N/A","Country":"India","CountryIso":"IN","CreatedDate":"2023-12-06T05:44:34.942+0000","EvaluationTime":0,"EventDate":"2023-12-06T05:44:24.973+0000","EventIdentifier":"00044326-ed4a-421a-a0a8-e62ea626f3af","HttpMethod":"POST","Id":"000000000000000AAA","LoginGeoId":"04F5j00003NvV1cEAF","LoginHistoryId":"0Ya5j00003k2scQCAQ","LoginKey":"pgOVoLbV96U9o08W","LoginLatitude":19.0748,"LoginLongitude":72.8856,"LoginType":"Remote Access 2.0","LoginUrl":"login.salesforce.com","Platform":"Unknown","PostalCode":"400070","SessionLevel":"STANDARD","SourceIp":"134.238.252.19","Status":"Success","Subdivision":"Maharashtra","TlsProtocol":"TLS 1.2","UserId":"0055j00000AT6I1AAL","UserType":"Standard","Username":"salesforceinstance@devtest.in"}`},
	},
}

func TestInput(t *testing.T) {
	logp.TestingSetup()

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			test.setupServer(t, test.handler, test.baseConfig)

			cfg := conf.MustNewConfigFrom(test.baseConfig)

			conf := defaultConfig()
			err := cfg.Unpack(&conf)
			assert.NoError(t, err)

			input := salesforceInput{}
			input.config = conf

			assert.Equal(t, "salesforce", input.Name())

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			v2Ctx := v2.Context{
				Logger:      logp.NewLogger("salesforce_test"),
				ID:          "test_id:" + test.name,
				Cancelation: ctx,
			}

			var client publisher
			client.done = func() {
				if len(client.published) >= len(test.expected) {
					cancel()
				}
			}

			src := &source{conf}

			err = input.run(v2Ctx, src, &state{}, &client)
			if test.wantErr != (err != nil) {
				t.Errorf("unexpected error from running input: got:%v want:%v", err, test.wantErr)
			}

			if len(client.published) < len(test.expected) {
				t.Errorf("unexpected number of published events: got:%d want at least:%d", len(client.published), len(test.expected))
				test.expected = test.expected[:len(client.published)]
			}

			client.published = client.published[:len(test.expected)]
			for i, got := range client.published {
				if !reflect.DeepEqual(got.Fields["event"].(map[string]interface{})["original"], test.expected[i]) {
					t.Errorf("unexpected result for event %d: got:- want:+\n%s", i, cmp.Diff(got.Fields, test.expected[i]))
				}
			}
		})
	}
}

func defaultHandler(expectedMethod, expectedBody, msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch {
		case r.RequestURI == "/services/oauth2/token":
			fmt.Println("in services/oauth2/token")
			w.WriteHeader(http.StatusOK)
			msg = `{"access_token":"abcd","instance_url":"http://` + r.Host + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`
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

		_, _ = w.Write([]byte(msg))
	}
}

func newTestServer(
	newServer func(http.Handler) *httptest.Server,
) func(testing.TB, http.HandlerFunc, map[string]interface{}) {
	return func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
		server := newServer(h)
		fmt.Printf("server.URL: %v\n", server.URL)
		config["url"] = server.URL
		config["auth.oauth2"].(map[string]interface{})["token_url"] = server.URL + "/services/oauth2/token"
		t.Cleanup(server.Close)
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
