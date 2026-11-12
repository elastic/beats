// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/go-sfdc"
	"github.com/elastic/go-sfdc/soql"

	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/mapstr"
	"github.com/elastic/beats/v7/pkg/transport/httpcommon"
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

	s := &salesforceInput{log: logp.NewLogger("salesforceInput")}

	mp, err := s.decodeAsCSV([]byte(sampleELF))
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

// TestJWTBearerFlowTokenURL verifies that the salesforce input correctly passes
// the token_url configuration to the underlying go-sfdc library, which handles
// the fallback logic (use token_url if set, otherwise fall back to url).
func TestJWTBearerFlowTokenURL(t *testing.T) {
	t.Parallel()

	// Create a test RSA key for JWT signing
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), 0o600))

	// oauthHandler returns a handler that tracks hits and responds with valid OAuth tokens
	oauthHandler := func(hits *int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			*hits++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"access_token":"token","instance_url":%q,"token_type":"Bearer"}`, "http://"+r.Host)
		}
	}

	// newConfig creates a minimal JWT auth config for testing
	newConfig := func(url, tokenURL, keyPath string) *config {
		return &config{
			Version: 56,
			URL:     url,
			Auth: &authConfig{OAuth2: &OAuth2{JWTBearerFlow: &JWTBearerFlow{
				Enabled:        pointer(true),
				URL:            url,
				TokenURL:       tokenURL,
				ClientID:       "test-client",
				ClientUsername: "test@example.com",
				ClientKeyPath:  keyPath,
			}}},
			EventMonitoringMethod: &eventMonitoringMethod{Object: EventMonitoringConfig{
				Enabled:  pointer(true),
				Interval: time.Second,
				Query:    &QueryConfig{Default: getValueTpl("SELECT Id FROM Account")},
				Cursor:   &cursorConfig{Field: "Id"},
			}},
			Resource: &resourceConfig{
				Retry:     retryConfig{MaxAttempts: pointer(1), WaitMin: pointer(time.Second), WaitMax: pointer(time.Second)},
				Transport: httpcommon.DefaultHTTPTransportSettings(),
			},
		}
	}

	t.Run("token_url empty - requests go to url", func(t *testing.T) {
		t.Parallel()

		var hits int
		srv := httptest.NewServer(oauthHandler(&hits))
		t.Cleanup(srv.Close)

		cfg := newConfig(srv.URL, "", keyPath) // token_url is empty

		input := &salesforceInput{config: *cfg, log: logp.NewLogger("test")}
		sfdcCfg, err := input.getSFDCConfig(cfg)
		require.NoError(t, err)

		input.sfdcConfig = sfdcCfg
		_, _ = input.SetupSFClientConnection()

		assert.Equal(t, 1, hits, "url should receive the OAuth request when token_url is empty")
	})

	t.Run("token_url set - requests go to token_url", func(t *testing.T) {
		t.Parallel()

		var urlHits, tokenURLHits int
		urlSrv := httptest.NewServer(oauthHandler(&urlHits))
		tokenURLSrv := httptest.NewServer(oauthHandler(&tokenURLHits))
		t.Cleanup(urlSrv.Close)
		t.Cleanup(tokenURLSrv.Close)

		cfg := newConfig(urlSrv.URL, tokenURLSrv.URL, keyPath) // token_url is set

		input := &salesforceInput{config: *cfg, log: logp.NewLogger("test")}
		sfdcCfg, err := input.getSFDCConfig(cfg)
		require.NoError(t, err)

		input.sfdcConfig = sfdcCfg
		_, _ = input.SetupSFClientConnection()

		assert.Equal(t, 0, urlHits, "url should NOT receive requests when token_url is set")
		assert.Equal(t, 1, tokenURLHits, "token_url should receive the OAuth request")
	})
<<<<<<< HEAD
=======

	t.Run("full token_url endpoint is normalized before request", func(t *testing.T) {
		t.Parallel()

		var urlHits, tokenURLHits int
		urlSrv := httptest.NewServer(oauthHandler(&urlHits))
		tokenURLSrv := httptest.NewServer(oauthHandler(&tokenURLHits))
		t.Cleanup(urlSrv.Close)
		t.Cleanup(tokenURLSrv.Close)

		cfg := newConfig(urlSrv.URL, tokenURLSrv.URL+"/services/oauth2/token", keyPath)

		input := &salesforceInput{config: *cfg, log: logptest.NewTestingLogger(t, "test")}
		sfdcCfg, err := input.getSFDCConfig(cfg)
		require.NoError(t, err)

		input.sfdcConfig = sfdcCfg
		_, err = input.SetupSFClientConnection()
		require.NoError(t, err, "expected a full jwt_bearer_flow.token_url endpoint to be normalized before go-sfdc appends its path")

		assert.Equal(t, 0, urlHits, "url should NOT receive requests when token_url is set")
		assert.Equal(t, 1, tokenURLHits, "token_url should receive the OAuth request on the canonical path")
	})
}

func TestGetSFDCConfigRejectsNilOAuth2(t *testing.T) {
	cfg := defaultConfig()
	cfg.Auth = &authConfig{}

	input := &salesforceInput{config: cfg, log: logptest.NewTestingLogger(t, "test")}

	var (
		sfdcCfg *sfdc.Configuration
		err     error
	)
	require.NotPanics(t, func() {
		sfdcCfg, err = input.getSFDCConfig(&cfg)
	}, "expected getSFDCConfig to reject nil OAuth2 without panicking")

	assert.Nil(t, sfdcCfg, "expected no Salesforce config when OAuth2 is missing")
	require.Error(t, err, "expected getSFDCConfig to reject nil OAuth2")
	assert.ErrorContains(t, err, "no auth provider enabled")
}

func TestRunWithMixedMonitoringMethodsStartsEventLogFileBeforeObject(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	var (
		requestKinds []string
		server       *httptest.Server
	)
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"` + server.URL + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case r.FormValue("q") == defaultLoginEventLogFileQuery:
			requestKinds = append(requestKinds, "elf_soql")
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileFirstResponseJSON))
		case r.RequestURI == "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN/LogFile":
			requestKinds = append(requestKinds, "elf_log_get")
			w.Header().Set("content-type", "text/csv")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileSecondResponseCSV))
		case r.FormValue("q") == defaultLoginObjectQuery:
			requestKinds = append(requestKinds, "object_soql")
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneObjectEvents))
		default:
			t.Fatalf("unexpected request: uri=%s query=%q", r.RequestURI, r.FormValue("q"))
		}
	}))
	t.Cleanup(server.Close)

	baseConfig := map[string]any{
		"url":         server.URL,
		"version":     56,
		"auth.oauth2": defaultUserPasswordFlowMap,
		"event_monitoring_method": map[string]any{
			"event_log_file": defaultEventLogFileMonitoringMethodMap,
			"object":         defaultObjectMonitoringMethodConfigMap,
		},
	}
	authOAuth2, _ := baseConfig["auth.oauth2"].(map[string]any)
	userPasswordFlow, _ := authOAuth2["user_password_flow"].(map[string]any)
	userPasswordFlow["token_url"] = server.URL

	cfg := defaultConfig()
	err := conf.MustNewConfigFrom(baseConfig).Unpack(&cfg)
	require.NoError(t, err, "expected mixed event monitoring config to unpack")

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	inputCtx := v2.Context{
		Logger: logptest.NewTestingLogger(t, "salesforce"),
		ID:     "test_id",
	}

	var client publisher
	ctx, cancel := context.WithCancelCause(timeoutCtx)
	client.done = func() {
		if len(client.published) >= 2 {
			cancel(nil)
		}
	}

	s := &salesforceInput{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancel,
		cursor:    &state{},
		srcConfig: &cfg,
		publisher: &client,
		log:       logptest.NewTestingLogger(t, "").With("input_url", "salesforce"),
	}

	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected Salesforce auth config to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected Salesforce query client setup to succeed")

	err = s.run(inputCtx)
	require.NoError(t, err, "expected mixed event monitoring run to stop cleanly after collecting both startup events")

	assert.Equal(t, []string{"elf_soql", "elf_log_get", "object_soql"}, requestKinds, "expected initial EventLogFile collection to complete before initial object collection starts")
	require.Len(t, client.published, 2, "expected one ELF event and one object event from the mixed startup path")
	elfMsg, ok := client.published[0].Fields["message"].(string)
	require.True(t, ok, "expected ELF event message to be a string")
	objMsg, ok := client.published[1].Fields["message"].(string)
	require.True(t, ok, "expected object event message to be a string")
	assert.JSONEq(t, expectedELFEvent, elfMsg, "expected the first published startup event to come from EventLogFile")
	assert.JSONEq(t, expectedObjectEvent, objMsg, "expected the second published startup event to come from Object monitoring")
}

func TestRunWithMixedMonitoringMethodsRunsBothTickersAfterStartup(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	const valueLoginEventLogFileQueryWithCursor = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND (CreatedDate > 2023-12-19T21:04:35.000+0000 OR (CreatedDate = 2023-12-19T21:04:35.000+0000 AND Id > '0AT5j00002LqQTxGAN')) ORDER BY CreatedDate ASC NULLS FIRST, Id ASC"

	var (
		requestKinds []string
		server       *httptest.Server
		mu           sync.Mutex
	)
	countKindLocked := func(kind string) int {
		count := 0
		for _, requestKind := range requestKinds {
			if requestKind == kind {
				count++
			}
		}
		return count
	}

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"` + server.URL + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case r.FormValue("q") == defaultLoginEventLogFileQuery, r.FormValue("q") == valueLoginEventLogFileQueryWithCursor:
			mu.Lock()
			requestKinds = append(requestKinds, "elf_soql")
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileFirstResponseJSON))
		case r.RequestURI == "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN/LogFile":
			mu.Lock()
			requestKinds = append(requestKinds, "elf_log_get")
			mu.Unlock()
			w.Header().Set("content-type", "text/csv")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileSecondResponseCSV))
		case r.FormValue("q") == defaultLoginObjectQuery, r.FormValue("q") == defaultLoginObjectQueryWithCursor:
			mu.Lock()
			requestKinds = append(requestKinds, "object_soql")
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneObjectEvents))
		default:
			t.Fatalf("unexpected request: uri=%s query=%q", r.RequestURI, r.FormValue("q"))
		}
	}))
	t.Cleanup(server.Close)

	baseConfig := map[string]any{
		"url":                         server.URL,
		"version":                     56,
		"resource.retry.max_attempts": 1,
		"auth.oauth2": map[string]any{
			"user_password_flow": map[string]any{
				"enabled":       true,
				"client.id":     "clientid",
				"client.secret": "clientsecret",
				"token_url":     server.URL,
				"username":      "username",
				"password":      "password",
			},
		},
		"event_monitoring_method": map[string]any{
			"event_log_file": map[string]any{
				"interval": "50ms",
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginEventLogFileQuery,
					"value":   valueLoginEventLogFileQuery,
				},
				"cursor": map[string]any{
					"field": "CreatedDate",
				},
			},
			"object": map[string]any{
				"interval": "80ms",
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginObjectQuery,
					"value":   valueLoginObjectQuery,
				},
				"cursor": map[string]any{
					"field": "EventDate",
				},
			},
		},
	}

	cfg := defaultConfig()
	err := conf.MustNewConfigFrom(baseConfig).Unpack(&cfg)
	require.NoError(t, err, "expected mixed event monitoring config to unpack")

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	inputCtx := v2.Context{
		Logger: logptest.NewTestingLogger(t, "salesforce"),
		ID:     "test_id",
	}

	var client publisher
	ctx, cancelCause := context.WithCancelCause(timeoutCtx)
	// Cancel once the publisher has observed at least two ELF and two
	// Object events. Gating cancellation on published events (rather than
	// inbound server requests) guarantees the in-flight SOQL response that
	// produced the second event was fully consumed before the input's
	// context is cancelled; otherwise input cancellation would abort the
	// in-flight HTTP request through ctxTransport and the test would count
	// fewer publications than requests.
	client.done = func() {
		var elfCount, objectCount int
		for _, ev := range client.published {
			switch ev.Fields["message"] {
			case expectedELFEvent:
				elfCount++
			case expectedObjectEvent:
				objectCount++
			}
		}
		if elfCount >= 2 && objectCount >= 2 {
			cancelCause(nil)
		}
	}

	s := &salesforceInput{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancelCause,
		cursor:    &state{},
		srcConfig: &cfg,
		publisher: &client,
		log:       logptest.NewTestingLogger(t, "").With("input_url", "salesforce"),
	}

	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected Salesforce auth config to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected Salesforce query client setup to succeed")

	err = s.run(inputCtx)
	require.NoError(t, err, "expected mixed event monitoring run to stop cleanly after both ticker branches ran")

	mu.Lock()
	requestKindsCopy := append([]string(nil), requestKinds...)
	elfSOQLCount := countKindLocked("elf_soql")
	elfLogGetCount := countKindLocked("elf_log_get")
	objectSOQLCount := countKindLocked("object_soql")
	mu.Unlock()

	require.GreaterOrEqual(t, len(requestKindsCopy), 6, "expected startup plus at least one ticker run for each mixed monitoring method")
	assert.Equal(t, []string{"elf_soql", "elf_log_get", "object_soql"}, requestKindsCopy[:3], "expected mixed startup ordering to remain unchanged before ticker work starts")
	assert.GreaterOrEqual(t, elfSOQLCount, 2, "expected EventLogFile SOQL to run at startup and at least once from its ticker")
	assert.GreaterOrEqual(t, elfLogGetCount, 2, "expected EventLogFile log fetch to run at startup and at least once from its ticker")
	assert.GreaterOrEqual(t, objectSOQLCount, 2, "expected Object SOQL to run at startup and at least once from its ticker")

	var elfPublished, objectPublished int
	for _, event := range client.published {
		if event.Fields["message"] == expectedELFEvent {
			elfPublished++
		}
		if event.Fields["message"] == expectedObjectEvent {
			objectPublished++
		}
	}
	assert.GreaterOrEqual(t, elfPublished, 2, "expected EventLogFile to publish at startup and at least once from ticker-driven collection")
	assert.GreaterOrEqual(t, objectPublished, 2, "expected Object monitoring to publish at startup and at least once from ticker-driven collection")
}

func TestRunEventLogFileSkipsQueuedTickerAfterFailure(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	const valueLoginEventLogFileQueryWithCursor = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND (CreatedDate > 2023-12-19T21:04:35.000+0000 OR (CreatedDate = 2023-12-19T21:04:35.000+0000 AND Id > '0AT5j00002LqQTxGAN')) ORDER BY CreatedDate ASC NULLS FIRST, Id ASC"

	var (
		cancel           context.CancelCauseFunc
		firstFailureSeen bool
		requestKinds     []string
		server           *httptest.Server
		mu               sync.Mutex
	)

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"` + server.URL + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case r.FormValue("q") == defaultLoginEventLogFileQuery:
			mu.Lock()
			requestKinds = append(requestKinds, "elf_soql")
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileFirstResponseJSON))
		case r.RequestURI == "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN/LogFile":
			mu.Lock()
			requestKinds = append(requestKinds, "elf_log_get")
			mu.Unlock()
			w.Header().Set("content-type", "text/csv")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileSecondResponseCSV))
		case r.FormValue("q") == valueLoginEventLogFileQueryWithCursor:
			mu.Lock()
			requestKinds = append(requestKinds, "elf_soql_error")
			shouldCancelSoon := !firstFailureSeen
			firstFailureSeen = true
			mu.Unlock()

			time.Sleep(150 * time.Millisecond)
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))

			if shouldCancelSoon && cancel != nil {
				go func() {
					time.Sleep(20 * time.Millisecond)
					cancel(nil)
				}()
			}
		default:
			t.Fatalf("unexpected request: uri=%s query=%q", r.RequestURI, r.FormValue("q"))
		}
	}))
	t.Cleanup(server.Close)

	baseConfig := map[string]any{
		"url":                         server.URL,
		"version":                     56,
		"resource.retry.max_attempts": 1,
		"auth.oauth2": map[string]any{
			"user_password_flow": map[string]any{
				"enabled":       true,
				"client.id":     "clientid",
				"client.secret": "clientsecret",
				"token_url":     server.URL,
				"username":      "username",
				"password":      "password",
			},
		},
		"event_monitoring_method": map[string]any{
			"event_log_file": map[string]any{
				"interval": "50ms",
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginEventLogFileQuery,
					"value":   valueLoginEventLogFileQuery,
				},
				"cursor": map[string]any{
					"field": "CreatedDate",
				},
			},
		},
	}

	cfg := defaultConfig()
	err := conf.MustNewConfigFrom(baseConfig).Unpack(&cfg)
	require.NoError(t, err, "expected EventLogFile monitoring config to unpack")

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	inputCtx := v2.Context{
		Logger: logptest.NewTestingLogger(t, "salesforce"),
		ID:     "test_id",
	}

	var client publisher
	client.done = func() {}
	ctx, cancelCause := context.WithCancelCause(timeoutCtx)
	cancel = cancelCause

	s := &salesforceInput{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancelCause,
		cursor:    &state{},
		srcConfig: &cfg,
		publisher: &client,
		log:       logptest.NewTestingLogger(t, "").With("input_url", "salesforce"),
	}

	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected Salesforce auth config to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected Salesforce query client setup to succeed")

	err = s.run(inputCtx)
	require.NoError(t, err, "expected EventLogFile run loop to stop cleanly after the first failed ticker run")

	mu.Lock()
	requestKindsCopy := append([]string(nil), requestKinds...)
	mu.Unlock()

	var failureCount int
	for _, kind := range requestKindsCopy {
		if kind == "elf_soql_error" {
			failureCount++
		}
	}

	require.GreaterOrEqual(t, len(requestKindsCopy), 3, "expected startup EventLogFile requests followed by a ticker-driven failure")
	assert.Equal(t, []string{"elf_soql", "elf_log_get", "elf_soql_error"}, requestKindsCopy[:3], "expected startup EventLogFile requests before the first ticker failure")
	assert.Equal(t, 1, failureCount, "expected a failed EventLogFile ticker run to wait for a fresh interval before retrying instead of immediately consuming a queued tick")
}

func TestRunWithMixedMonitoringMethodsContinuesObjectAfterEventLogFileTickerFailure(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	const valueLoginEventLogFileQueryWithCursor = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND (CreatedDate > 2023-12-19T21:04:35.000+0000 OR (CreatedDate = 2023-12-19T21:04:35.000+0000 AND Id > '0AT5j00002LqQTxGAN')) ORDER BY CreatedDate ASC NULLS FIRST, Id ASC"

	var (
		requestKinds []string
		server       *httptest.Server
		mu           sync.Mutex
	)
	countKindLocked := func(kind string) int {
		count := 0
		for _, requestKind := range requestKinds {
			if requestKind == kind {
				count++
			}
		}
		return count
	}

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"` + server.URL + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case r.FormValue("q") == defaultLoginEventLogFileQuery:
			mu.Lock()
			requestKinds = append(requestKinds, "elf_soql")
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileFirstResponseJSON))
		case r.FormValue("q") == valueLoginEventLogFileQueryWithCursor:
			mu.Lock()
			requestKinds = append(requestKinds, "elf_soql_error")
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		case r.RequestURI == "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN/LogFile":
			mu.Lock()
			requestKinds = append(requestKinds, "elf_log_get")
			mu.Unlock()
			w.Header().Set("content-type", "text/csv")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileSecondResponseCSV))
		case r.FormValue("q") == defaultLoginObjectQuery, r.FormValue("q") == defaultLoginObjectQueryWithCursor:
			mu.Lock()
			requestKinds = append(requestKinds, "object_soql")
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneObjectEvents))
		default:
			t.Fatalf("unexpected request: uri=%s query=%q", r.RequestURI, r.FormValue("q"))
		}
	}))
	t.Cleanup(server.Close)

	baseConfig := map[string]any{
		"url":                         server.URL,
		"version":                     56,
		"resource.retry.max_attempts": 1,
		"auth.oauth2": map[string]any{
			"user_password_flow": map[string]any{
				"enabled":       true,
				"client.id":     "clientid",
				"client.secret": "clientsecret",
				"token_url":     server.URL,
				"username":      "username",
				"password":      "password",
			},
		},
		"event_monitoring_method": map[string]any{
			"event_log_file": map[string]any{
				"interval": "40ms",
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginEventLogFileQuery,
					"value":   valueLoginEventLogFileQuery,
				},
				"cursor": map[string]any{
					"field": "CreatedDate",
				},
			},
			"object": map[string]any{
				"interval": "80ms",
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginObjectQuery,
					"value":   valueLoginObjectQuery,
				},
				"cursor": map[string]any{
					"field": "EventDate",
				},
			},
		},
	}

	cfg := defaultConfig()
	err := conf.MustNewConfigFrom(baseConfig).Unpack(&cfg)
	require.NoError(t, err, "expected mixed event monitoring config to unpack")

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	inputCtx := v2.Context{
		Logger: logptest.NewTestingLogger(t, "salesforce"),
		ID:     "test_id",
	}

	var client publisher
	ctx, cancelCause := context.WithCancelCause(timeoutCtx)
	// Cancel once at least two Object events have been published so the
	// in-flight Object response has been fully consumed before ctxTransport
	// propagates the cancelled ctx into outstanding HTTP reads. The failing
	// ELF ticker request is allowed to complete on its own schedule since
	// the test also asserts elfSOQLErrorCount >= 1 independently.
	var elfFailSeen bool
	client.done = func() {
		var objectCount int
		for _, ev := range client.published {
			if ev.Fields["message"] == expectedObjectEvent {
				objectCount++
			}
		}
		mu.Lock()
		elfFailSeen = elfFailSeen || countKindLocked("elf_soql_error") >= 1
		mu.Unlock()
		if elfFailSeen && objectCount >= 2 {
			cancelCause(nil)
		}
	}

	s := &salesforceInput{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancelCause,
		cursor:    &state{},
		srcConfig: &cfg,
		publisher: &client,
		log:       logptest.NewTestingLogger(t, "").With("input_url", "salesforce"),
	}

	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected Salesforce auth config to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected Salesforce query client setup to succeed")

	err = s.run(inputCtx)
	require.NoError(t, err, "expected mixed event monitoring run to stop cleanly after ELF ticker failure and later object progress")

	mu.Lock()
	requestKindsCopy := append([]string(nil), requestKinds...)
	elfSOQLErrorCount := countKindLocked("elf_soql_error")
	elfLogGetCount := countKindLocked("elf_log_get")
	objectSOQLCount := countKindLocked("object_soql")
	mu.Unlock()

	require.GreaterOrEqual(t, len(requestKindsCopy), 4, "expected startup requests plus the failing ELF ticker request and a later object request")
	assert.Equal(t, []string{"elf_soql", "elf_log_get", "object_soql"}, requestKindsCopy[:3], "expected mixed startup ordering to remain unchanged before the failing ELF ticker run")
	assert.GreaterOrEqual(t, elfSOQLErrorCount, 1, "expected at least one ticker-driven EventLogFile SOQL failure")
	assert.Equal(t, 1, elfLogGetCount, "expected no additional EventLogFile log fetch after the ticker-driven SOQL failure")
	assert.GreaterOrEqual(t, objectSOQLCount, 2, "expected Object monitoring to keep running after EventLogFile ticker failure")

	var elfPublished, objectPublished int
	for _, event := range client.published {
		if event.Fields["message"] == expectedELFEvent {
			elfPublished++
		}
		if event.Fields["message"] == expectedObjectEvent {
			objectPublished++
		}
	}
	assert.Equal(t, 1, elfPublished, "expected only the startup EventLogFile run to publish when later ticker SOQL requests fail")
	assert.GreaterOrEqual(t, objectPublished, 2, "expected Object monitoring to continue publishing after EventLogFile ticker failure")
}

func TestRunWithMixedMonitoringMethodsContinuesEventLogFileAfterObjectTickerFailure(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	const valueLoginEventLogFileQueryWithCursor = "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND (CreatedDate > 2023-12-19T21:04:35.000+0000 OR (CreatedDate = 2023-12-19T21:04:35.000+0000 AND Id > '0AT5j00002LqQTxGAN')) ORDER BY CreatedDate ASC NULLS FIRST, Id ASC"

	var (
		requestKinds []string
		server       *httptest.Server
		mu           sync.Mutex
	)
	countKindLocked := func(kind string) int {
		count := 0
		for _, requestKind := range requestKinds {
			if requestKind == kind {
				count++
			}
		}
		return count
	}
	maybeCancelLocked := func() {
		// This test cancels from publisher.done so the second ELF publish is
		// observed after the object ticker failure.
	}

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"` + server.URL + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case r.FormValue("q") == defaultLoginEventLogFileQuery, r.FormValue("q") == valueLoginEventLogFileQueryWithCursor:
			mu.Lock()
			requestKinds = append(requestKinds, "elf_soql")
			maybeCancelLocked()
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileFirstResponseJSON))
		case r.RequestURI == "/services/data/v58.0/sobjects/EventLogFile/0AT5j00002LqQTxGAN/LogFile":
			mu.Lock()
			requestKinds = append(requestKinds, "elf_log_get")
			maybeCancelLocked()
			mu.Unlock()
			w.Header().Set("content-type", "text/csv")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneEventLogfileSecondResponseCSV))
		case r.FormValue("q") == defaultLoginObjectQuery:
			mu.Lock()
			requestKinds = append(requestKinds, "object_soql")
			maybeCancelLocked()
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(oneObjectEvents))
		case r.FormValue("q") == defaultLoginObjectQueryWithCursor:
			mu.Lock()
			requestKinds = append(requestKinds, "object_soql_error")
			maybeCancelLocked()
			mu.Unlock()
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		default:
			t.Fatalf("unexpected request: uri=%s query=%q", r.RequestURI, r.FormValue("q"))
		}
	}))
	t.Cleanup(server.Close)

	baseConfig := map[string]any{
		"url":                         server.URL,
		"version":                     56,
		"resource.retry.max_attempts": 1,
		"auth.oauth2": map[string]any{
			"user_password_flow": map[string]any{
				"enabled":       true,
				"client.id":     "clientid",
				"client.secret": "clientsecret",
				"token_url":     server.URL,
				"username":      "username",
				"password":      "password",
			},
		},
		"event_monitoring_method": map[string]any{
			"event_log_file": map[string]any{
				"interval": "80ms",
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginEventLogFileQuery,
					"value":   valueLoginEventLogFileQuery,
				},
				"cursor": map[string]any{
					"field": "CreatedDate",
				},
			},
			"object": map[string]any{
				"interval": "40ms",
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginObjectQuery,
					"value":   valueLoginObjectQuery,
				},
				"cursor": map[string]any{
					"field": "EventDate",
				},
			},
		},
	}

	cfg := defaultConfig()
	err := conf.MustNewConfigFrom(baseConfig).Unpack(&cfg)
	require.NoError(t, err, "expected mixed event monitoring config to unpack")

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelTimeout()

	inputCtx := v2.Context{
		Logger: logptest.NewTestingLogger(t, "salesforce"),
		ID:     "test_id",
	}

	var client publisher
	ctx, cancelCause := context.WithCancelCause(timeoutCtx)
	client.done = func() {
		mu.Lock()
		objectSOQLErrorCount := countKindLocked("object_soql_error")
		mu.Unlock()

		elfPublished := 0
		for _, event := range client.published {
			if event.Fields["message"] == expectedELFEvent {
				elfPublished++
			}
		}
		if objectSOQLErrorCount >= 1 && elfPublished >= 2 {
			cancelCause(nil)
		}
	}

	s := &salesforceInput{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancelCause,
		cursor:    &state{},
		srcConfig: &cfg,
		publisher: &client,
		log:       logptest.NewTestingLogger(t, "").With("input_url", "salesforce"),
	}

	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected Salesforce auth config to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected Salesforce query client setup to succeed")

	err = s.run(inputCtx)
	require.NoError(t, err, "expected mixed event monitoring run to stop cleanly after object ticker failure and later EventLogFile progress")

	mu.Lock()
	requestKindsCopy := append([]string(nil), requestKinds...)
	elfSOQLCount := countKindLocked("elf_soql")
	elfLogGetCount := countKindLocked("elf_log_get")
	objectSOQLErrorCount := countKindLocked("object_soql_error")
	mu.Unlock()

	require.GreaterOrEqual(t, len(requestKindsCopy), 5, "expected startup requests plus failing object ticker request and later EventLogFile requests")
	assert.Equal(t, []string{"elf_soql", "elf_log_get", "object_soql"}, requestKindsCopy[:3], "expected mixed startup ordering to remain unchanged before the failing object ticker run")
	assert.GreaterOrEqual(t, objectSOQLErrorCount, 1, "expected at least one ticker-driven object SOQL failure")
	assert.GreaterOrEqual(t, elfSOQLCount, 2, "expected EventLogFile SOQL to continue after object ticker failure")
	assert.GreaterOrEqual(t, elfLogGetCount, 2, "expected EventLogFile log fetch to continue after object ticker failure")

	var elfPublished, objectPublished int
	for _, event := range client.published {
		if event.Fields["message"] == expectedELFEvent {
			elfPublished++
		}
		if event.Fields["message"] == expectedObjectEvent {
			objectPublished++
		}
	}
	assert.GreaterOrEqual(t, elfPublished, 2, "expected EventLogFile to keep publishing after object ticker failure")
	assert.Equal(t, 1, objectPublished, "expected only the startup Object run to publish when later ticker SOQL requests fail")
}

// newClientForCancellationTest builds a Salesforce HTTP client through the
// production newClient with short retry waits so cancellation tests stay
// quick. getCtx is plumbed unchanged into newClient.
func newClientForCancellationTest(t *testing.T, getCtx func() context.Context, maxAttempts int) *http.Client {
	t.Helper()
	waitMin := 500 * time.Millisecond
	waitMax := 1 * time.Second
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = 5 * time.Second
	cfg := config{
		Resource: &resourceConfig{
			Retry: retryConfig{
				MaxAttempts: &maxAttempts,
				WaitMin:     &waitMin,
				WaitMax:     &waitMax,
			},
			Transport: transport,
		},
	}
	c, err := newClient(cfg, getCtx, logptest.NewTestingLogger(t, "salesforce-test"))
	require.NoError(t, err, "newClient should succeed with valid retry config")
	return c
}

// TestNewClientShortCircuitsRetryOnContextCancel asserts that cancelling
// the input context while the retryablehttp wrapper is mid-backoff returns
// promptly instead of running the full attempt budget. ctxTransport on the
// inner http.Transport alone is not enough — retryablehttp inspects
// req.Context() (which go-sfdc leaves as context.Background) when deciding
// whether to retry, so the input context must also be threaded through the
// retryablehttp boundary.
func TestNewClientShortCircuitsRetryOnContextCancel(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	var attempts atomic.Int32
	firstSeen := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			select {
			case firstSeen <- struct{}{}:
			default:
			}
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := newClientForCancellationTest(t, func() context.Context { return ctx }, 5)

	go func() {
		select {
		case <-firstSeen:
		case <-time.After(2 * time.Second):
			return
		}
		// Cancel mid-backoff: retryablehttp's first sleep is 500ms, so
		// 50ms leaves a wide margin for scheduler lag on loaded runners.
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.Error(t, err, "client.Do should surface the cancellation as an error")
	require.ErrorIs(t, err, context.Canceled, "cancellation should propagate as context.Canceled")
	// Without the fix, retryablehttp runs the full backoff (~4s with
	// these settings: 500+1000+1000+1000+1000 ms) and the server receives 5
	// attempts. With the fix retryablehttp bails after attempt 1.
	require.Less(t, elapsed, 500*time.Millisecond, "retryablehttp ran the full backoff instead of honoring input cancellation (took %s)", elapsed)
	require.LessOrEqual(t, int(attempts.Load()), 1, "server received %d attempts; retryablehttp should have stopped after the first once the input context was cancelled", attempts.Load())
}

// TestNewClientPreCancelledContextShortCircuits asserts that when the
// input context is already cancelled before a request is issued, the
// retryablehttp wrapper returns immediately and the request never makes
// it through more than a single attempt.
func TestNewClientPreCancelledContextShortCircuits(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := newClientForCancellationTest(t, func() context.Context { return ctx }, 5)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.Error(t, err, "client.Do should fail when the input context is already cancelled")
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, elapsed, 200*time.Millisecond, "pre-cancelled context should short-circuit (took %s)", elapsed)
	require.LessOrEqual(t, int(attempts.Load()), 1, "pre-cancelled context should not produce a full retry storm; server saw %d attempts", attempts.Load())
}

// TestNewClientNilGetCtxStillRetries asserts the retry behavior is
// unchanged when no input context is plumbed through (getCtx == nil),
// covering the unit-test path and any other caller that constructs a
// client without an active input.
func TestNewClientNilGetCtxStillRetries(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	const maxAttempts = 2
	client := newClientForCancellationTest(t, nil, maxAttempts)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err, "retryablehttp surfaces the last 5xx response without an error")
	require.NotNil(t, resp)
	defer resp.Body.Close()

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	// retryablehttp counts retries, not total attempts, so MaxAttempts=N
	// produces 1 initial + N retries = N+1 hits at the server.
	require.Equal(t, int32(maxAttempts+1), attempts.Load(), "with no input context plumbed in, retryablehttp should still run the full attempt budget")
}

// TestRunObjectCooldownSuppressesNextTickerTick exercises the steady-state
// post-failure cooldown documented in doc.go: a failed Object collection is
// expected to suppress the next ticker tick so a sustained outage produces one
// failed Salesforce API call per (2 * interval), not one per interval. A naive
// "now + interval" cooldown races the ticker — the next tick fires at the same
// instant the cooldown expires (time.NewTicker(interval) keeps that cadence),
// so time.Now().Before(backoff) evaluates to false at every tick and no tick
// is ever skipped.
//
// The test fails fast on every Object SOQL call (no handler latency) and lets
// the natural ticker cadence run for ~5 intervals. Without a safety margin
// every tick fires a request; with the cooldown padded past the next tick,
// every other tick is suppressed.
func TestRunObjectCooldownSuppressesNextTickerTick(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	const interval = 100 * time.Millisecond
	const runFor = 550 * time.Millisecond

	var (
		objectFailureCount atomic.Int32
		server             *httptest.Server
	)

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"` + server.URL + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case strings.Contains(r.FormValue("q"), "FROM LoginEvent"):
			objectFailureCount.Add(1)
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		default:
			t.Fatalf("unexpected request: uri=%s query=%q", r.RequestURI, r.FormValue("q"))
		}
	}))
	t.Cleanup(server.Close)

	baseConfig := map[string]any{
		"url":                         server.URL,
		"version":                     56,
		"resource.retry.max_attempts": 1,
		"auth.oauth2": map[string]any{
			"user_password_flow": map[string]any{
				"enabled":       true,
				"client.id":     "clientid",
				"client.secret": "clientsecret",
				"token_url":     server.URL,
				"username":      "username",
				"password":      "password",
			},
		},
		"event_monitoring_method": map[string]any{
			"object": map[string]any{
				"interval": interval.String(),
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginObjectQuery,
					"value":   valueLoginObjectQuery,
				},
				"cursor": map[string]any{
					"field": "EventDate",
				},
			},
		},
	}

	cfg := defaultConfig()
	err := conf.MustNewConfigFrom(baseConfig).Unpack(&cfg)
	require.NoError(t, err, "expected Object monitoring config to unpack")

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), runFor)
	defer cancelTimeout()

	inputCtx := v2.Context{
		Logger: logptest.NewTestingLogger(t, "salesforce"),
		ID:     "test_id",
	}

	var client publisher
	client.done = func() {}
	ctx, cancelCause := context.WithCancelCause(timeoutCtx)

	s := &salesforceInput{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancelCause,
		cursor:    &state{},
		srcConfig: &cfg,
		publisher: &client,
		log:       logptest.NewTestingLogger(t, "").With("input_url", "salesforce"),
	}

	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected Salesforce auth config to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected Salesforce query client setup to succeed")

	err = s.run(inputCtx)
	require.NoError(t, err, "expected Object run loop to stop cleanly when the timeout fires")

	got := int(objectFailureCount.Load())
	// Without the cooldown fix the loop runs the initial request plus a
	// failed request on every tick, so over ~5 intervals we observe 5 or
	// more requests. With the cooldown firing once per failure, every
	// second tick is suppressed and the count stays at <= 3 (initial run +
	// at most two ticker-driven retries within the 550ms window).
	require.LessOrEqual(t, got, 3, "expected the post-failure cooldown to suppress alternating Object ticks under sustained failure (got %d Object SOQL hits in %s with interval=%s)", got, runFor, interval)
	require.GreaterOrEqual(t, got, 1, "expected at least the startup Object SOQL request to reach the server")
}

// TestRunObjectInitialFailureSetsCooldown asserts that a failure during the
// startup-phase Object collection (the run() invocation that happens BEFORE
// the ticker starts) also installs the cooldown. Without it, the very first
// ticker tick after a startup failure always fires a fresh request because
// the startup error path increments objectFails and updates status but
// leaves objectBackoffUntil zero — so the first ticker tick sees no backoff
// and runs another doomed request one interval later.
func TestRunObjectInitialFailureSetsCooldown(t *testing.T) {
	logptest.NewTestingLogger(t, "")

	const interval = 100 * time.Millisecond
	const runFor = 180 * time.Millisecond

	var (
		objectFailureCount atomic.Int32
		server             *httptest.Server
	)

	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.RequestURI == "/services/oauth2/token" && r.Method == http.MethodPost:
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"abcd","instance_url":"` + server.URL + `","token_type":"Bearer","id_token":"abcd","refresh_token":"abcd"}`))
		case strings.Contains(r.FormValue("q"), "FROM LoginEvent"):
			objectFailureCount.Add(1)
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"boom"}`))
		default:
			t.Fatalf("unexpected request: uri=%s query=%q", r.RequestURI, r.FormValue("q"))
		}
	}))
	t.Cleanup(server.Close)

	baseConfig := map[string]any{
		"url":                         server.URL,
		"version":                     56,
		"resource.retry.max_attempts": 1,
		"auth.oauth2": map[string]any{
			"user_password_flow": map[string]any{
				"enabled":       true,
				"client.id":     "clientid",
				"client.secret": "clientsecret",
				"token_url":     server.URL,
				"username":      "username",
				"password":      "password",
			},
		},
		"event_monitoring_method": map[string]any{
			"object": map[string]any{
				"interval": interval.String(),
				"enabled":  true,
				"query": map[string]any{
					"default": defaultLoginObjectQuery,
					"value":   valueLoginObjectQuery,
				},
				"cursor": map[string]any{
					"field": "EventDate",
				},
			},
		},
	}

	cfg := defaultConfig()
	err := conf.MustNewConfigFrom(baseConfig).Unpack(&cfg)
	require.NoError(t, err, "expected Object monitoring config to unpack")

	timeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), runFor)
	defer cancelTimeout()

	inputCtx := v2.Context{
		Logger: logptest.NewTestingLogger(t, "salesforce"),
		ID:     "test_id",
	}

	var client publisher
	client.done = func() {}
	ctx, cancelCause := context.WithCancelCause(timeoutCtx)

	s := &salesforceInput{
		config:    cfg,
		ctx:       ctx,
		cancel:    cancelCause,
		cursor:    &state{},
		srcConfig: &cfg,
		publisher: &client,
		log:       logptest.NewTestingLogger(t, "").With("input_url", "salesforce"),
	}

	s.sfdcConfig, err = s.getSFDCConfig(&cfg)
	require.NoError(t, err, "expected Salesforce auth config to succeed")

	s.soqlr, err = s.SetupSFClientConnection()
	require.NoError(t, err, "expected Salesforce query client setup to succeed")

	err = s.run(inputCtx)
	require.NoError(t, err, "expected Object run loop to stop cleanly when the timeout fires")

	got := int(objectFailureCount.Load())
	// Window is initial run (T~0) + the first ticker tick (T~interval) + a
	// small buffer. With the bug the first ticker tick fires a second
	// request because the startup failure path leaves objectBackoffUntil
	// zero. With the fix the cooldown installed by the startup failure
	// suppresses the first tick, so only the startup request is observed.
	require.Equal(t, 1, got, "expected the startup Object failure to install a cooldown that suppresses the first ticker tick (got %d Object SOQL hits in %s with interval=%s)", got, runFor, interval)
>>>>>>> 0894882 (increase timeouts, should make runs on busy CI machines less flaky (#54548))
}
