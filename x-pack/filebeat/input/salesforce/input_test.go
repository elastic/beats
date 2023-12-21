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

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFormQueryWithCursor(t *testing.T) {
	logp.TestingSetup()

	mockTimeNow(time.Date(2023, time.May, 18, 12, 0, 0, 0, time.UTC))
	t.Cleanup(resetTimeNow)

	tests := map[string]struct {
		initialInterval     time.Duration
		defaultSOQLTemplate string
		valueSOQLTemplate   string
		wantQuery           string
		cursor              mapstr.M
		wantErr             error
	}{
		"valid soql templates with nil cursor": { // expect default query with LogDate > initialInterval
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 60 days (2 months)
			defaultSOQLTemplate: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ .var.initial_interval ]] ORDER BY CreatedDate ASC NULLS FIRST",
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > 2023-03-19T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              nil,
		},
		"valid soql templates with non-empty .cursor.object.logdate": { // expect value SOQL query with .cursor.object.logdate set
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 60 days (2 months)
			defaultSOQLTemplate: "SELECT Id,CreatedDate,LogDate,LogFile FROM LoginEvent WHERE EventDate > [[ .var.initial_interval ]]",
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM LoginEvent WHERE  CreatedDate > [[ .cursor.object.logdate ]]",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM LoginEvent WHERE  CreatedDate > 2023-05-18T12:00:00Z",
			cursor:              mapstr.M{"object": mapstr.M{"logdate": timeNow().Format(formatRFC3339Like)}},
		},
		"valid soql templates with non-empty .cursor.event_log_file.logdate": { // expect value SOQL query with .cursor.event_log_file.logdate set
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 60 days (2 months)
			defaultSOQLTemplate: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ .var.initial_interval ]] ORDER BY CreatedDate ASC NULLS FIRST",
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.event_log_file.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > 2023-05-18T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              mapstr.M{"event_log_file": mapstr.M{"logdate": timeNow().Format(formatRFC3339Like)}},
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
			}

			querier, err := sfInput.FormQueryWithCursor(queryConfig, tc.cursor)
			assert.NoError(t, err)

			assert.EqualValues(t, tc.wantQuery, querier.Query)
		})
	}
}

func TestInput(t *testing.T) {
	logp.TestingSetup()

	tests := []struct {
		name        string
		setupServer func(testing.TB, http.HandlerFunc, map[string]interface{})
		baseConfig  map[string]interface{}
		handler     http.HandlerFunc
		expected    []string
		wantErr     bool
	}{
		{
			name:        "event_monitoring_method_object_with_default_query_only",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"enabled":       pointer(true),
					"client.id":     "clientid",
					"client.secret": "clientsecret",
					"token_url":     "https://instance_id.develop.my.salesforce.com/services/oauth2/token",
					"user":          "username",
					"password":      "password",
				},
				"version": 56,
				"event_monitoring_method": map[string]interface{}{
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupServer(t, tc.handler, tc.baseConfig)

			cfg := defaultConfig()
			err := conf.MustNewConfigFrom(tc.baseConfig).Unpack(&cfg)
			assert.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var client publisher
			client.done = func() {
				if len(client.published) >= len(tc.expected) {
					cancel()
				}
			}

			salesforceInput := salesforceInput{config: cfg}
			assert.Equal(t, "salesforce", salesforceInput.Name())

			ctx, cancelClause := context.WithCancelCause(ctx)

			salesforceInput.cursor = &state{}
			salesforceInput.ctx = ctx
			salesforceInput.cancel = cancelClause
			salesforceInput.srcConfig = &cfg
			salesforceInput.publisher = &client
			salesforceInput.log = logp.NewLogger("salesforce")

			salesforceInput.sfdcConfig, err = getSFDCConfig(&cfg)
			assert.NoError(t, err)

			salesforceInput.soqlr, err = salesforceInput.SetupSFClientConnection()
			assert.NoError(t, err)

			err = salesforceInput.run()
			if tc.wantErr != (err != nil) {
				t.Errorf("unexpected error from running input: got:%v want:%v", err, tc.wantErr)
			}

			if len(client.published) < len(tc.expected) {
				t.Errorf("unexpected number of published events: got:%d want at least:%d", len(client.published), len(tc.expected))
				tc.expected = tc.expected[:len(client.published)]
			}

			client.published = client.published[:len(tc.expected)]
			for i, got := range client.published {
				if !reflect.DeepEqual(got.Fields["message"], tc.expected[i]) {
					t.Errorf("unexpected result for event %d: got:- want:+\n%s", i, cmp.Diff(got.Fields, tc.expected[i]))
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

func newTestServer(newServer func(http.Handler) *httptest.Server) func(testing.TB, http.HandlerFunc, map[string]interface{}) {
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
	defer p.mu.Unlock()

	p.published = append(p.published, e)
	if cursor != nil {
		c, ok := cursor.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid cursor type for testing: %T", cursor)
		}
		p.cursors = append(p.cursors, c)
	}
	p.done()

	return nil
}
