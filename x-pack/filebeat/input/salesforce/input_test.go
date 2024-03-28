// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/g8rswimmer/go-sfdc"
	"github.com/g8rswimmer/go-sfdc/soql"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

const (
	PaginationFlow   = "PaginationFlow"
	NoPaginationFlow = "NoPaginationFlow"
	IntervalFlow     = "IntervalFlow"
	BadReponseFlow   = "BadReponseFlow"

	defaultLoginObjectQuery           = "SELECT FIELDS(STANDARD) FROM LoginEvent"
	valueLoginObjectQuery             = "SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > [[ .cursor.object.first_event_time ]]"
	defaultLoginObjectQueryWithCursor = "SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > 2023-12-06T05:44:24.973+0000"

	defaultLoginEventLogFileQuery = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' ORDER BY CreatedDate ASC NULLS FIRST"
	valueLoginEventLogFileQuery   = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.event_log_file.last_event_time ]] ORDER BY CreatedDate ASC NULLS FIRST"

	invalidDefaultLoginEventObjectQuery  = "SELECT FIELDS(STANDARD) FROM LoginEvnt"
	invalidDefaultLoginEventLogFileQuery = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' ORDER BY ASC NULLS FIRST"

	invalidValueLoginObjectQuery       = "SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > [[ .cursor.object.first_event ]]"
	invalidValueLoginEventLogFileQuery = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.event_log_file.last_event ]] ORDER BY CreatedDate ASC NULLS FIRST"

	oneEventLogfileFirstResponseJSON = `{ "totalSize": 1, "done": true, "records": [ { "attributes": { "type": "EventLogFile", "url": "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN" }, "Id": "0AT5j00002LqQTxGAN", "CreatedDate": "2023-12-19T21:04:35.000+0000", "LogDate": "2023-12-18T00:00:00.000+0000", "LogFile": "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN/LogFile" } ] }`
	oneEventLogfileSecondResponseCSV = `"EVENT_TYPE","TIMESTAMP","REQUEST_ID","ORGANIZATION_ID","USER_ID","RUN_TIME","CPU_TIME","URI","SESSION_KEY","LOGIN_KEY","USER_TYPE","REQUEST_STATUS","DB_TOTAL_TIME","LOGIN_TYPE","BROWSER_TYPE","API_TYPE","API_VERSION","USER_NAME","TLS_PROTOCOL","CIPHER_SUITE","AUTHENTICATION_METHOD_REFERENCE","LOGIN_SUB_TYPE","TIMESTAMP_DERIVED","USER_ID_DERIVED","CLIENT_IP","URI_ID_DERIVED","LOGIN_STATUS","SOURCE_IP"
"Login","20231218054831.655","4u6LyuMrDvb_G-l1cJIQk-","00D5j00000DgAYG","0055j00000AT6I1","1219","127","/services/oauth2/token","","bY5Wfv8t/Ith7WVE","Standard","","1051271151","i","Go-http-client/1.1","","9998.0","salesforceinstance@devtest.in","TLSv1.2","ECDHE-RSA-AES256-GCM-SHA384","","","2023-12-18T05:48:31.655Z","0055j00000AT6I1AAL","Salesforce.com IP","","LOGIN_NO_ERROR","103.108.207.58"
`

	expectedELFEvent = `{"API_TYPE":"","API_VERSION":"9998.0","AUTHENTICATION_METHOD_REFERENCE":"","BROWSER_TYPE":"Go-http-client/1.1","CIPHER_SUITE":"ECDHE-RSA-AES256-GCM-SHA384","CLIENT_IP":"Salesforce.com IP","CPU_TIME":"127","DB_TOTAL_TIME":"1051271151","EVENT_TYPE":"Login","LOGIN_KEY":"bY5Wfv8t/Ith7WVE","LOGIN_STATUS":"LOGIN_NO_ERROR","LOGIN_SUB_TYPE":"","LOGIN_TYPE":"i","ORGANIZATION_ID":"00D5j00000DgAYG","REQUEST_ID":"4u6LyuMrDvb_G-l1cJIQk-","REQUEST_STATUS":"","RUN_TIME":"1219","SESSION_KEY":"","SOURCE_IP":"103.108.207.58","TIMESTAMP":"20231218054831.655","TIMESTAMP_DERIVED":"2023-12-18T05:48:31.655Z","TLS_PROTOCOL":"TLSv1.2","URI":"/services/oauth2/token","URI_ID_DERIVED":"","USER_ID":"0055j00000AT6I1","USER_ID_DERIVED":"0055j00000AT6I1AAL","USER_NAME":"salesforceinstance@devtest.in","USER_TYPE":"Standard"}`

	oneObjectEvents        = `{ "totalSize": 1, "done": true, "records": [ { "attributes": { "type": "LoginEvent", "url": "/services/data/v58.0/sobjects/LoginEvent/000000000000000AAA" }, "AdditionalInfo": "{}", "ApiType": "N/A", "ApiVersion": "N/A", "Application": "salesforce_test", "Browser": "Unknown", "CipherSuite": "ECDHE-RSA-AES256-GCM-SHA384", "City": "Mumbai", "ClientVersion": "N/A", "Country": "India", "CountryIso": "IN", "CreatedDate": "2023-12-06T05:44:34.942+0000", "EvaluationTime": 0, "EventDate": "2023-12-06T05:44:24.973+0000", "EventIdentifier": "00044326-ed4a-421a-a0a8-e62ea626f3af", "HttpMethod": "POST", "Id": "000000000000000AAA", "LoginGeoId": "04F5j00003NvV1cEAF", "LoginHistoryId": "0Ya5j00003k2scQCAQ", "LoginKey": "pgOVoLbV96U9o08W", "LoginLatitude": 19.0748, "LoginLongitude": 72.8856, "LoginType": "Remote Access 2.0", "LoginUrl": "login.salesforce.com", "Platform": "Unknown", "PostalCode": "400070", "SessionLevel": "STANDARD", "SourceIp": "134.238.252.19", "Status": "Success", "Subdivision": "Maharashtra", "TlsProtocol": "TLS 1.2", "UserId": "0055j00000AT6I1AAL", "UserType": "Standard", "Username": "salesforceinstance@devtest.in" } ] }`
	oneObjectEventsPageOne = `{ "totalSize": 1, "done": true, "nextRecordsUrl": "/nextRecords/LoginEvents/ABCABCDABCDE", "records": [ { "attributes": { "type": "LoginEvent", "url": "/services/data/v58.0/sobjects/LoginEvent/000000000000000AAA" }, "AdditionalInfo": "{}", "ApiType": "N/A", "ApiVersion": "N/A", "Application": "salesforce_test", "Browser": "Unknown", "CipherSuite": "ECDHE-RSA-AES256-GCM-SHA384", "City": "Mumbai", "ClientVersion": "N/A", "Country": "India", "CountryIso": "IN", "CreatedDate": "2023-12-06T05:44:34.942+0000", "EvaluationTime": 0, "EventDate": "2023-12-06T05:44:24.973+0000", "EventIdentifier": "00044326-ed4a-421a-a0a8-e62ea626f3af", "HttpMethod": "POST", "Id": "000000000000000AAA", "LoginGeoId": "04F5j00003NvV1cEAF", "LoginHistoryId": "0Ya5j00003k2scQCAQ", "LoginKey": "pgOVoLbV96U9o08W", "LoginLatitude": 19.0748, "LoginLongitude": 72.8856, "LoginType": "Remote Access 2.0", "LoginUrl": "login.salesforce.com", "Platform": "Unknown", "PostalCode": "400070", "SessionLevel": "STANDARD", "SourceIp": "134.238.252.19", "Status": "Success", "Subdivision": "Maharashtra", "TlsProtocol": "TLS 1.2", "UserId": "0055j00000AT6I1AAL", "UserType": "Standard", "Username": "salesforceinstance@devtest.in" } ] }`
	oneObjectEventsPageTwo = `{ "totalSize": 1, "done": true, "records": [ { "attributes": { "type": "LoginEvent", "url": "/services/data/v58.0/sobjects/LoginEvent/000000000000000AAA" }, "AdditionalInfo": "{}", "ApiType": "N/A", "ApiVersion": "N/A", "Application": "salesforce_test", "Browser": "Unknown", "CipherSuite": "ECDHE-RSA-AES256-GCM-SHA384", "City": "Mumbai", "ClientVersion": "N/A", "Country": "India", "CountryIso": "IN", "CreatedDate": "2023-12-06T05:44:34.942+0000", "EvaluationTime": 0, "EventDate": "2023-12-06T05:44:24.973+0000", "EventIdentifier": "00044326-ed4a-421a-a0a8-e62ea626f3af", "HttpMethod": "POST", "Id": "000000000000000AAA", "LoginGeoId": "04F5j00003NvV1cEAF", "LoginHistoryId": "0Ya5j00003k2scQCAQ", "LoginKey": "pgOVoLbV96U9o08W", "LoginLatitude": 19.0748, "LoginLongitude": 72.8856, "LoginType": "Remote Access 2.0", "LoginUrl": "login.salesforce.com", "Platform": "Unknown", "PostalCode": "400070", "SessionLevel": "STANDARD", "SourceIp": "134.238.252.19", "Status": "Success", "Subdivision": "Maharashtra", "TlsProtocol": "TLS 1.2", "UserId": "0055j00000AT6I1AAL", "UserType": "Standard", "Username": "salesforceinstance@devtest.in" } ] }`

	expectedObjectEvent = `{"AdditionalInfo":"{}","ApiType":"N/A","ApiVersion":"N/A","Application":"salesforce_test","Browser":"Unknown","CipherSuite":"ECDHE-RSA-AES256-GCM-SHA384","City":"Mumbai","ClientVersion":"N/A","Country":"India","CountryIso":"IN","CreatedDate":"2023-12-06T05:44:34.942+0000","EvaluationTime":0,"EventDate":"2023-12-06T05:44:24.973+0000","EventIdentifier":"00044326-ed4a-421a-a0a8-e62ea626f3af","HttpMethod":"POST","Id":"000000000000000AAA","LoginGeoId":"04F5j00003NvV1cEAF","LoginHistoryId":"0Ya5j00003k2scQCAQ","LoginKey":"pgOVoLbV96U9o08W","LoginLatitude":19.0748,"LoginLongitude":72.8856,"LoginType":"Remote Access 2.0","LoginUrl":"login.salesforce.com","Platform":"Unknown","PostalCode":"400070","SessionLevel":"STANDARD","SourceIp":"134.238.252.19","Status":"Success","Subdivision":"Maharashtra","TlsProtocol":"TLS 1.2","UserId":"0055j00000AT6I1AAL","UserType":"Standard","Username":"salesforceinstance@devtest.in"}`
)

func TestFormQueryWithCursor(t *testing.T) {
	logp.TestingSetup()

	mockTimeNow(time.Date(2023, time.May, 18, 12, 0, 0, 0, time.UTC))
	t.Cleanup(resetTimeNow)

	tests := map[string]struct {
		wantErr             error
		cursor              mapstr.M
		defaultSOQLTemplate string
		valueSOQLTemplate   string
		wantQuery           string
		initialInterval     time.Duration
	}{
		"valid soql templates with nil cursor": { // expect default query with LogDate > initialInterval
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 1440h = 60 days = 2 months
			defaultSOQLTemplate: `SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ (formatTime (now.Add (parseDuration "-1440h")) "RFC3339") ]] ORDER BY CreatedDate ASC NULLS FIRST`,
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > 2023-03-19T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              nil,
		},
		"valid soql templates with non-empty .cursor.object.logdate": { // expect value SOQL query with .cursor.object.logdate set
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 1440h = 60 days = 2 months
			defaultSOQLTemplate: `SELECT Id,CreatedDate,LogDate,LogFile FROM LoginEvent WHERE EventDate > [[ (formatTime (now.Add (parseDuration "-1440h")) "RFC3339") ]]`,
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM LoginEvent WHERE  CreatedDate > [[ .cursor.object.logdate ]]",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM LoginEvent WHERE  CreatedDate > 2023-05-18T12:00:00Z",
			cursor:              mapstr.M{"object": mapstr.M{"logdate": timeNow().Format(formatRFC3339Like)}},
		},
		"valid soql templates with non-empty .cursor.event_log_file.logdate": { // expect value SOQL query with .cursor.event_log_file.logdate set
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 1440h = 60 days = 2 months
			defaultSOQLTemplate: `SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ (formatTime (now.Add (parseDuration "-1440h")) "RFC3339") ]] ORDER BY CreatedDate ASC NULLS FIRST`,
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.event_log_file.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > 2023-05-18T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              mapstr.M{"event_log_file": mapstr.M{"logdate": timeNow().Format(formatRFC3339Like)}},
		},
		"invalid soql templates wrong cursor name .cursor.event_log_file.logdate1": { // expect value SOQL query with .cursor.event_log_file.logdate set
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 1440h = 60 days = 2 months
			defaultSOQLTemplate: `SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ (formatTime (now.Add (parseDuration "-1440h")) "RFC3339") ]] ORDER BY CreatedDate ASC NULLS FIRST`,
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.event_log_file.logdate1 ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > 2023-05-18T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              mapstr.M{"event_log_file": mapstr.M{"logdate": timeNow().Format(formatRFC3339Like)}},
			wantErr:             errors.New(`template: :1:110: executing "" at <.cursor.event_log_file.logdate1>: map has no entry for key "logdate1"`),
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
				config: config{},
				log:    logp.NewLogger("salesforce_test"),
			}

			querier, err := sfInput.FormQueryWithCursor(queryConfig, tc.cursor)
			if fmt.Sprint(tc.wantErr) != fmt.Sprint(err) {
				t.Errorf("got error %v, want error %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}

			assert.EqualValues(t, tc.wantQuery, querier.Query)
		})
	}
}

var (
	defaultUserPasswordFlowMap = map[string]interface{}{
		"user_password_flow": map[string]interface{}{
			"enabled":       true,
			"client.id":     "clientid",
			"client.secret": "clientsecret",
			"token_url":     "https://instance_id.develop.my.salesforce.com/services/oauth2/token",
			"username":      "username",
			"password":      "password",
		},
	}
	wrongUserPasswordFlowMap = map[string]interface{}{
		"user_password_flow": map[string]interface{}{
			"enabled":       true,
			"client.id":     "clientid-wrong",
			"client.secret": "clientsecret-wrong",
			"token_url":     "https://instance_id.develop.my.salesforce.com/services/oauth2/token",
			"username":      "username-wrong",
			"password":      "password-wrong",
		},
	}

	defaultObjectMonitoringMethodConfigMap = map[string]interface{}{
		"interval": "5s",
		"enabled":  true,
		"query": map[string]interface{}{
			"default": defaultLoginObjectQuery,
			"value":   valueLoginObjectQuery,
		},
		"cursor": map[string]interface{}{
			"field": "EventDate",
		},
	}
	defaultEventLogFileMonitoringMethodMap = map[string]interface{}{
		"interval": "5s",
		"enabled":  true,
		"query": map[string]interface{}{
			"default": defaultLoginEventLogFileQuery,
			"value":   valueLoginEventLogFileQuery,
		},
		"cursor": map[string]interface{}{
			"field": "CreatedDate",
		},
	}

	invalidObjectMonitoringMethodMap = map[string]interface{}{
		"interval": "5m",
		"enabled":  true,
		"query": map[string]interface{}{
			"default": invalidDefaultLoginEventObjectQuery,
			"value":   valueLoginEventLogFileQuery,
		},
		"cursor": map[string]interface{}{
			"field": "CreatedDate",
		},
	}
	invalidEventLogFileMonitoringMethodMap = map[string]interface{}{
		"interval": "5m",
		"enabled":  true,
		"query": map[string]interface{}{
			"default": invalidDefaultLoginEventLogFileQuery,
			"value":   invalidValueLoginEventLogFileQuery,
		},
		"cursor": map[string]interface{}{
			"field": "CreatedDate",
		},
	}
)

func TestInput(t *testing.T) {
	logp.TestingSetup()

	tests := []struct {
		setupServer      func(testing.TB, http.HandlerFunc, map[string]interface{})
		baseConfig       map[string]interface{}
		handler          http.HandlerFunc
		persistentCursor *state
		name             string
		expected         []string
		timeout          time.Duration
		wantErr          bool
		AuthFail         bool
	}{
		// Object
		{
			name:        "Positive/event_monitoring_method_object_with_default_query_only",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"version":     56,
				"auth.oauth2": defaultUserPasswordFlowMap,
				"event_monitoring_method": map[string]interface{}{
					"object": defaultObjectMonitoringMethodConfigMap,
				},
			},
			handler:  defaultHandler(NoPaginationFlow, false, "", oneObjectEvents),
			expected: []string{expectedObjectEvent},
		},
		{
			name:        "Negative/event_monitoring_method_object_with_error_in_data_collection",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"version":     56,
				"auth.oauth2": defaultUserPasswordFlowMap,
				"event_monitoring_method": map[string]interface{}{
					"object": invalidObjectMonitoringMethodMap,
				},
			},
			handler: defaultHandler(NoPaginationFlow, false, "", `{"error": "invalid_query"}`),
			wantErr: true,
		},
		{
			name:        "Positive/event_monitoring_method_object_with_interval_5s",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"version":     56,
				"auth.oauth2": defaultUserPasswordFlowMap,
				"event_monitoring_method": map[string]interface{}{
					"object": defaultObjectMonitoringMethodConfigMap,
				},
			},
			handler:  defaultHandler(IntervalFlow, false, "", oneObjectEventsPageTwo),
			expected: []string{expectedObjectEvent, expectedObjectEvent},
			timeout:  20 * time.Second,
		},
		{
			name:        "Positive/event_monitoring_method_object_with_Pagination",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"version":     56,
				"auth.oauth2": defaultUserPasswordFlowMap,
				"event_monitoring_method": map[string]interface{}{
					"object": defaultObjectMonitoringMethodConfigMap,
				},
			},
			handler:  defaultHandler(PaginationFlow, false, oneObjectEventsPageOne, oneObjectEventsPageTwo),
			expected: []string{expectedObjectEvent, expectedObjectEvent},
		},

		// EventLogFile
		{
			name:        "Positive/event_monitoring_method_elf_with_default_query_only",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"version":     56,
				"auth.oauth2": defaultUserPasswordFlowMap,
				"event_monitoring_method": map[string]interface{}{
					"event_log_file": defaultEventLogFileMonitoringMethodMap,
				},
			},
			handler:  defaultHandler(NoPaginationFlow, false, oneEventLogfileFirstResponseJSON, oneEventLogfileSecondResponseCSV),
			expected: []string{expectedELFEvent},
		},
		{
			name:        "Negative/event_monitoring_method_elf_with_error_in_auth",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"version":     56,
				"auth.oauth2": wrongUserPasswordFlowMap,
				"event_monitoring_method": map[string]interface{}{
					"event_log_file": defaultEventLogFileMonitoringMethodMap,
				},
			},
			handler:  defaultHandler(NoPaginationFlow, false, "", `{"error": "invalid_client_id"}`),
			wantErr:  true,
			AuthFail: true,
		},
		{
			name:        "Negative/event_monitoring_method_elf_with_error_in_data_collection",
			setupServer: newTestServer(httptest.NewServer),
			baseConfig: map[string]interface{}{
				"version":     56,
				"auth.oauth2": defaultUserPasswordFlowMap,
				"event_monitoring_method": map[string]interface{}{
					"event_log_file": invalidEventLogFileMonitoringMethodMap,
				},
			},
			handler: defaultHandler(NoPaginationFlow, false, "", `{"error": "invalid_query"}`),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupServer(t, tc.handler, tc.baseConfig)

			cfg := defaultConfig()
			err := conf.MustNewConfigFrom(tc.baseConfig).Unpack(&cfg)
			assert.NoError(t, err)
			timeout := 5 * time.Second
			if tc.timeout != 0 {
				timeout = tc.timeout
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
			if tc.persistentCursor != nil {
				salesforceInput.cursor = tc.persistentCursor
			}
			salesforceInput.ctx = ctx
			salesforceInput.cancel = cancelClause
			salesforceInput.srcConfig = &cfg
			salesforceInput.publisher = &client
			salesforceInput.log = logp.L().With("input_url", "salesforce")

			salesforceInput.sfdcConfig, err = salesforceInput.getSFDCConfig(&cfg)
			assert.NoError(t, err)

			salesforceInput.soqlr, err = salesforceInput.SetupSFClientConnection()
			if err != nil && !tc.wantErr {
				t.Errorf("unexpected error from running input: %v", err)
			}
			if tc.wantErr && tc.AuthFail {
				return
			}

			err = salesforceInput.run()
			if err != nil && !tc.wantErr {
				t.Errorf("unexpected error from running input: %v", err)
			}
			if tc.wantErr {
				return
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

func defaultHandler(flow string, withoutQuery bool, msg1, msg2 string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		switch {
		case flow == PaginationFlow && r.FormValue("q") == defaultLoginObjectQuery:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(msg1))
		case r.RequestURI == "/nextRecords/LoginEvents/ABCABCDABCDE":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(msg2))
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost && r.FormValue("client_id") == "clientid":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"http://` + r.Host + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case r.FormValue("client_id") == "clientid-wrong":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(msg2))
		case r.FormValue("q") == defaultLoginEventLogFileQuery:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(msg1))
		case r.FormValue("q") == defaultLoginObjectQuery, r.FormValue("q") == defaultLoginObjectQueryWithCursor, r.RequestURI == "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN/LogFile":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(msg2))
		case r.FormValue("q") == invalidDefaultLoginEventLogFileQuery, r.FormValue("q") == invalidDefaultLoginEventObjectQuery:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(msg2))
		case flow == BadReponseFlow && (withoutQuery && r.FormValue("q") == ""):
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"internal server error"}`))
		}
	}
}

func newTestServer(newServer func(http.Handler) *httptest.Server) func(testing.TB, http.HandlerFunc, map[string]interface{}) {
	return func(t testing.TB, h http.HandlerFunc, config map[string]interface{}) {
		server := newServer(h)
		config["url"] = server.URL
		config["auth.oauth2"].(map[string]interface{})["user_password_flow"].(map[string]interface{})["token_url"] = server.URL
		t.Cleanup(server.Close)
	}
}

var _ inputcursor.Publisher = (*publisher)(nil)

type publisher struct {
	done      func()
	published []beat.Event
	cursors   []map[string]interface{}
	mu        sync.Mutex
}

func (p *publisher) Publish(e beat.Event, cursor interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.published = append(p.published, e)
	if cursor != nil {
		var cv map[string]interface{}
		err := typeconv.Convert(&cv, cursor)
		if err != nil {
			return err
		}

		p.cursors = append(p.cursors, cv)
	}
	p.done()

	return nil
}

func TestDecodeAsCSV(t *testing.T) {
	sampleELF := `"EVENT_TYPE","TIMESTAMP","REQUEST_ID","ORGANIZATION_ID","USER_ID","RUN_TIME","CPU_TIME","URI","SESSION_KEY","LOGIN_KEY","USER_TYPE","REQUEST_STATUS","DB_TOTAL_TIME","LOGIN_TYPE","BROWSER_TYPE","API_TYPE","API_VERSION","USER_NAME","TLS_PROTOCOL","CIPHER_SUITE","AUTHENTICATION_METHOD_REFERENCE","LOGIN_SUB_TYPE","TIMESTAMP_DERIVED","USER_ID_DERIVED","CLIENT_IP","URI_ID_DERIVED","LOGIN_STATUS","SOURCE_IP"
"Login","20231218054831.655","4u6LyuMrDvb_G-l1cJIQk-","00D5j00000DgAYG","0055j00000AT6I1","1219","127","/services/oauth2/token","","bY5Wfv8t/Ith7WVE","Standard","","1051271151","i","Go-http-client/1.1","","9998.0","salesforceinstance@devtest.in","TLSv1.2","ECDHE-RSA-AES256-GCM-SHA384","","","2023-12-18T05:48:31.655Z","0055j00000AT6I1AAL","Salesforce.com IP","","LOGIN_NO_ERROR","103.108.207.58"
"Login","20231218054832.003","4u6LyuHSDv8LLVl1cJOqGV","00D5j00000DgAYG","0055j00000AT6I1","1277","104","/services/oauth2/token","","u60el7VqW8CSSKcW","Standard","","674857427","i","Go-http-client/1.1","","9998.0","salesforceinstance@devtest.in","TLSv1.2","ECDHE-RSA-AES256-GCM-SHA384","","","2023-12-18T05:48:32.003Z","0055j00000AT6I1AAL","103.108.207.58","","LOGIN_NO_ERROR","103.108.207.58"`

	mp, err := decodeAsCSV([]byte(sampleELF))
	assert.NoError(t, err)

	wantNumOfEvents := 2
	gotNumOfEvents := len(mp)
	assert.Equal(t, wantNumOfEvents, gotNumOfEvents)

	wantEventFields := map[string]string{
		"LOGIN_TYPE":                      "i",
		"API_VERSION":                     "9998.0",
		"TIMESTAMP_DERIVED":               "2023-12-18T05:48:31.655Z",
		"TIMESTAMP":                       "20231218054831.655",
		"USER_NAME":                       "salesforceinstance@devtest.in",
		"SOURCE_IP":                       "103.108.207.58",
		"CPU_TIME":                        "127",
		"REQUEST_STATUS":                  "",
		"DB_TOTAL_TIME":                   "1051271151",
		"TLS_PROTOCOL":                    "TLSv1.2",
		"AUTHENTICATION_METHOD_REFERENCE": "",
		"REQUEST_ID":                      "4u6LyuMrDvb_G-l1cJIQk-",
		"USER_ID":                         "0055j00000AT6I1",
		"RUN_TIME":                        "1219",
		"CIPHER_SUITE":                    "ECDHE-RSA-AES256-GCM-SHA384",
		"CLIENT_IP":                       "Salesforce.com IP",
		"EVENT_TYPE":                      "Login",
		"LOGIN_SUB_TYPE":                  "",
		"USER_ID_DERIVED":                 "0055j00000AT6I1AAL",
		"URI_ID_DERIVED":                  "",
		"ORGANIZATION_ID":                 "00D5j00000DgAYG",
		"URI":                             "/services/oauth2/token",
		"LOGIN_KEY":                       "bY5Wfv8t/Ith7WVE",
		"USER_TYPE":                       "Standard",
		"API_TYPE":                        "",
		"SESSION_KEY":                     "",
		"BROWSER_TYPE":                    "Go-http-client/1.1",
		"LOGIN_STATUS":                    "LOGIN_NO_ERROR",
	}

	assert.Equal(t, wantEventFields, mp[0])
}

func TestSalesforceInputRunWithMethod(t *testing.T) {
	var (
		defaultUserPassAuthConfig = authConfig{
			OAuth2: &OAuth2{
				UserPasswordFlow: &UserPasswordFlow{
					Enabled:      pointer(true),
					TokenURL:     "https://instance_id.develop.my.salesforce.com/services/oauth2/token",
					ClientID:     "clientid",
					ClientSecret: "clientsecret",
					Username:     "username",
					Password:     "password",
				},
			},
		}
		objectEventMonitotingConfig = eventMonitoringMethod{
			Object: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Second * 5,
				Query: &QueryConfig{
					Default: getValueTpl(defaultLoginObjectQuery),
					Value:   getValueTpl(valueLoginObjectQuery),
				},
				Cursor: &cursorConfig{Field: "EventDate"},
			},
		}
		objectEventMonitoringWithWrongQuery = eventMonitoringMethod{
			Object: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Second * 5,
				Query: &QueryConfig{
					Default: getValueTpl(invalidDefaultLoginEventObjectQuery),
					Value:   getValueTpl(invalidValueLoginObjectQuery),
				},
				Cursor: &cursorConfig{Field: "EventDate"},
			},
		}

		elfEventMonitotingConfig = eventMonitoringMethod{
			EventLogFile: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Second * 5,
				Query: &QueryConfig{
					Default: getValueTpl(defaultLoginEventLogFileQuery),
					Value:   getValueTpl(valueLoginEventLogFileQuery),
				},
				Cursor: &cursorConfig{Field: "EventDate"},
			},
		}
		elfEventMonitotingWithWrongQuery = eventMonitoringMethod{
			EventLogFile: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Second * 5,
				Query: &QueryConfig{
					Default: getValueTpl(invalidDefaultLoginEventLogFileQuery),
					Value:   getValueTpl(invalidValueLoginEventLogFileQuery),
				},
				Cursor: &cursorConfig{Field: "EventDate"},
			},
		}
	)

	type fields struct {
		ctx        context.Context
		publisher  inputcursor.Publisher
		cancel     context.CancelCauseFunc
		cursor     *state
		srcConfig  *config
		sfdcConfig *sfdc.Configuration
		soqlr      *soql.Resource
		config     config
	}

	defaultResource := resourceConfig{
		Retry: retryConfig{
			MaxAttempts: pointer(5),
			WaitMin:     pointer(time.Minute),
			WaitMax:     pointer(time.Minute),
		},
		Transport: httpcommon.DefaultHTTPTransportSettings(),
	}

	tests := []struct {
		fields               fields
		setupServer          func(testing.TB, http.HandlerFunc, *config)
		handler              http.HandlerFunc
		method               string
		name                 string
		expected             []string
		wantErr              bool
		AuthFail             bool
		ClientConnectionFail bool
	}{
		// Object
		{
			name:        "Positive/object_get_one_event",
			method:      "Object",
			setupServer: newTestServerBasedOnConfig(httptest.NewServer),
			handler:     defaultHandler(NoPaginationFlow, false, "", oneObjectEvents),
			fields: fields{
				config: config{
					Version:               56,
					Auth:                  &defaultUserPassAuthConfig,
					EventMonitoringMethod: &objectEventMonitotingConfig,
					Resource:              &defaultResource,
				},
				cursor: &state{},
			},
			expected: []string{expectedObjectEvent},
		},
		{
			name:        "Negative/object_error_from_wrong_default_query",
			method:      "Object",
			setupServer: newTestServerBasedOnConfig(httptest.NewServer),
			handler:     defaultHandler(NoPaginationFlow, false, "", oneObjectEvents),
			fields: fields{
				config: config{
					Version:               56,
					Auth:                  &defaultUserPassAuthConfig,
					EventMonitoringMethod: &objectEventMonitoringWithWrongQuery,
					Resource:              &defaultResource,
				},
				cursor: &state{},
			},
			wantErr: true,
		},
		{
			name:        "Negative/object_error_from_wrong_value_query",
			method:      "Object",
			setupServer: newTestServerBasedOnConfig(httptest.NewServer),
			handler:     defaultHandler(NoPaginationFlow, false, "", oneObjectEvents),
			fields: fields{
				config: config{
					Version:               56,
					Auth:                  &defaultUserPassAuthConfig,
					EventMonitoringMethod: &objectEventMonitoringWithWrongQuery,
					Resource:              &defaultResource,
				},
				cursor: &state{
					Object: dateTimeCursor{
						FirstEventTime: "2020-01-01T00:00:00Z",
						LastEventTime:  "2020-01-01T00:00:00Z",
					},
				},
			},
			wantErr: true,
		},

		// EventLogFile
		{
			name:        "Positive/elf_get_one_event",
			method:      "ELF",
			setupServer: newTestServerBasedOnConfig(httptest.NewServer),
			handler:     defaultHandler(NoPaginationFlow, false, oneEventLogfileFirstResponseJSON, oneEventLogfileSecondResponseCSV),
			fields: fields{
				config: config{
					Version:               56,
					Auth:                  &defaultUserPassAuthConfig,
					EventMonitoringMethod: &elfEventMonitotingConfig,
					Resource:              &defaultResource,
				},
				cursor: &state{},
			},
			expected: []string{expectedELFEvent},
		},
		{
			name:        "Negative/elf_error_from_wrong_default_query",
			method:      "ELF",
			setupServer: newTestServerBasedOnConfig(httptest.NewServer),
			handler:     defaultHandler(NoPaginationFlow, false, oneEventLogfileFirstResponseJSON, oneEventLogfileSecondResponseCSV),
			fields: fields{
				config: config{
					Version:               56,
					Auth:                  &defaultUserPassAuthConfig,
					EventMonitoringMethod: &elfEventMonitotingWithWrongQuery,
					Resource:              &defaultResource,
				},
				cursor: &state{},
			},
			wantErr: true,
		},
		{
			name:        "Negative/elf_error_from_wrong_value_query",
			method:      "ELF",
			setupServer: newTestServerBasedOnConfig(httptest.NewServer),
			handler:     defaultHandler(NoPaginationFlow, false, oneEventLogfileFirstResponseJSON, oneEventLogfileSecondResponseCSV),
			fields: fields{
				config: config{
					Version:               56,
					Auth:                  &defaultUserPassAuthConfig,
					EventMonitoringMethod: &elfEventMonitotingWithWrongQuery,
					Resource:              &defaultResource,
				},
				cursor: &state{
					EventLogFile: dateTimeCursor{
						FirstEventTime: "2020-01-01T00:00:00Z",
						LastEventTime:  "2020-01-01T00:00:00Z",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		config := tt.fields.config

		t.Run(tt.name, func(t *testing.T) {
			tt.setupServer(t, tt.handler, &config)

			s := &salesforceInput{
				config:     config,
				ctx:        tt.fields.ctx,
				cancel:     tt.fields.cancel,
				publisher:  tt.fields.publisher,
				cursor:     tt.fields.cursor,
				srcConfig:  tt.fields.srcConfig,
				sfdcConfig: tt.fields.sfdcConfig,
				log:        logp.NewLogger("salesforceInput"),
				soqlr:      tt.fields.soqlr,
			}

			ctx, cancel := context.WithCancelCause(context.Background())
			s.ctx = ctx
			s.cancel = cancel

			var client publisher
			client.done = func() {
				if len(client.published) >= len(tt.expected) {
					cancel(nil)
				}
			}
			s.publisher = &client
			s.srcConfig = &s.config

			var err error
			s.sfdcConfig, err = s.getSFDCConfig(&s.config)
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error from running input: %v", err)
			}
			if tt.wantErr && tt.AuthFail {
				return
			}

			s.soqlr, err = s.SetupSFClientConnection()
			if err != nil && !tt.wantErr {
				t.Errorf("unexpected error from running input: %v", err)
			}
			if tt.wantErr && tt.ClientConnectionFail {
				return
			}

			if tt.method == "Object" {
				if err := s.RunObject(); (err != nil) != tt.wantErr {
					t.Errorf("salesforceInput.RunObject() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				if err := s.RunEventLogFile(); (err != nil) != tt.wantErr {
					t.Errorf("salesforceInput.RunEventLogFile() error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			if len(client.published) < len(tt.expected) {
				t.Errorf("unexpected number of published events: got:%d want at least:%d", len(client.published), len(tt.expected))
				tt.expected = tt.expected[:len(client.published)]
			}

			client.published = client.published[:len(tt.expected)]
			for i, got := range client.published {
				if !reflect.DeepEqual(got.Fields["message"], tt.expected[i]) {
					t.Errorf("unexpected result for event %d: got:- want:+\n%s", i, cmp.Diff(got.Fields, tt.expected[i]))
				}
			}
		})
	}
}

func getValueTpl(in string) *valueTpl {
	vp := &valueTpl{}
	vp.Unpack(in) //nolint:errcheck // ignore error in test

	return vp
}

func newTestServerBasedOnConfig(newServer func(http.Handler) *httptest.Server) func(testing.TB, http.HandlerFunc, *config) {
	return func(t testing.TB, h http.HandlerFunc, config *config) {
		server := newServer(h)
		config.URL = server.URL
		config.Auth.OAuth2.UserPasswordFlow.TokenURL = server.URL
		t.Cleanup(server.Close)
	}
}

func TestPlugin(t *testing.T) {
	_ = Plugin(logp.NewLogger("salesforce_test"), stateStore{})
}
