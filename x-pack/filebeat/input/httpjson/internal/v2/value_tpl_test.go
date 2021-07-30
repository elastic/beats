// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestValueTpl(t *testing.T) {
	cases := []struct {
		name          string
		value         string
		paramCtx      *transformContext
		paramTr       transformable
		paramDefVal   string
		expectedVal   string
		expectedError string
		setup         func()
		teardown      func()
	}{
		{
			name:  "can render values from ctx",
			value: "[[.last_response.body.param]]",
			paramCtx: &transformContext{
				firstEvent:   &common.MapStr{},
				lastEvent:    &common.MapStr{},
				lastResponse: newTestResponse(common.MapStr{"param": 25}, nil, ""),
			},
			paramTr:     transformable{},
			paramDefVal: "",
			expectedVal: "25",
		},
		{
			name:  "can render default value if execute fails",
			value: "[[.last_response.body.does_not_exist]]",
			paramCtx: &transformContext{
				lastEvent: &common.MapStr{},
			},
			paramTr:     transformable{},
			paramDefVal: "25",
			expectedVal: "25",
		},
		{
			name:        "can render default value if template is empty",
			value:       "",
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			paramDefVal: "25",
			expectedVal: "25",
		},
		{
			name:          "returns error if result is empty and no default is set",
			value:         "",
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			paramDefVal:   "",
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "can render default value if execute panics",
			value:       "[[.last_response.panic]]",
			paramDefVal: "25",
			expectedVal: "25",
		},
		{
			name:          "returns error if panics and no default is set",
			value:         "[[.last_response.panic]]",
			paramDefVal:   "",
			expectedVal:   "",
			expectedError: errExecutingTemplate.Error(),
		},
		{
			name:        "func parseDuration",
			value:       `[[ parseDuration "-1h" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "-1h0m0s",
		},
		{
			name:        "func now",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[ now ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 13:25:32 +0000 UTC",
		},
		{
			name:        "func now with duration",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[ now (parseDuration "-1h") ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 12:25:32 +0000 UTC",
		},
		{
			name:        "func parseDate",
			value:       `[[ parseDate "2020-11-05T12:25:32.1234567Z" "RFC3339Nano" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 12:25:32.1234567 +0000 UTC",
		},
		{
			name:        "func parseDate defaults to RFC3339",
			value:       `[[ parseDate "2020-11-05T12:25:32Z" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 12:25:32 +0000 UTC",
		},
		{
			name:        "func parseDate with custom layout",
			value:       `[[ (parseDate "Thu Nov  5 12:25:32 +0000 2020" "Mon Jan _2 15:04:05 -0700 2006") ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 12:25:32 +0000 UTC",
		},
		{
			name:        "func formatDate",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[ formatDate (now) "UnixDate" "America/New_York" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "Thu Nov  5 08:25:32 EST 2020",
		},
		{
			name:        "func formatDate defaults to UTC",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[ formatDate (now) "UnixDate" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "Thu Nov  5 13:25:32 UTC 2020",
		},
		{
			name:        "func formatDate falls back to UTC",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[ formatDate (now) "UnixDate" "wrong/tz"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "Thu Nov  5 13:25:32 UTC 2020",
		},
		{
			name:        "func parseTimestamp",
			value:       `[[ (parseTimestamp 1604582732) ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 13:25:32 +0000 UTC",
		},
		{
			name:        "func parseTimestampMilli",
			value:       `[[ (parseTimestampMilli 1604582732000) ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 13:25:32 +0000 UTC",
		},
		{
			name:        "func parseTimestampNano",
			value:       `[[ (parseTimestampNano 1604582732000000000) ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 13:25:32 +0000 UTC",
		},
		{
			name:  "func getRFC5988Link",
			value: `[[ getRFC5988Link "previous" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				firstEvent: &common.MapStr{},
				lastEvent:  &common.MapStr{},
				lastResponse: newTestResponse(
					nil,
					http.Header{"Link": []string{
						`<https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK>; title="Page 3"; rel="next"`,
						`<https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK>; title="Page 1"; rel="previous"`,
					}},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: "https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK",
		},
		{
			name:  "func getRFC5988Link does not match",
			value: `[[ getRFC5988Link "previous" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				lastResponse: newTestResponse(
					nil,
					http.Header{"Link": []string{
						`<https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK>`,
					}},
					"",
				),
			},
			paramTr:     transformable{},
			paramDefVal: "https://example.com/default",
			expectedVal: "https://example.com/default",
		},
		{
			name:        "func getRFC5988Link empty header",
			value:       `[[ getRFC5988Link "previous" .last_response.header.Empty ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			paramDefVal: "https://example.com/default",
			expectedVal: "https://example.com/default",
		},
		{
			name:        "can execute functions pipeline",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1604582732, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[ (parseDuration "-1h") | now | formatDate ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05T12:25:32Z",
		},
		{
			name:        "func toInt",
			value:       `[[(toInt "1")]] [[(toInt 1.0)]] [[(toInt "1,0")]] [[(toInt 2)]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "1 1 0 2",
		},
		{
			name:        "func add",
			value:       `[[add 1 2 3 4]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "10",
		},
		{
			name:        "func mul",
			value:       `[[mul 4 4]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "16",
		},
		{
			name:        "func div",
			value:       `[[div 16 4]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "4",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup()
			}
			if tc.teardown != nil {
				t.Cleanup(tc.teardown)
			}
			tpl := &valueTpl{}
			assert.NoError(t, tpl.Unpack(tc.value))

			var defTpl *valueTpl
			if tc.paramDefVal != "" {
				defTpl = &valueTpl{}
				assert.NoError(t, defTpl.Unpack(tc.paramDefVal))
			}

			got, err := tpl.Execute(tc.paramCtx, tc.paramTr, defTpl, logp.NewLogger(""))
			assert.Equal(t, tc.expectedVal, got)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}

func newTestResponse(body common.MapStr, header http.Header, url string) *response {
	resp := &response{
		header: http.Header{},
	}
	if len(body) > 0 {
		resp.body = body
	}
	if len(header) > 0 {
		resp.header = header
	}
	if url != "" {
		resp.url = newURL(url)
	}
	return resp
}
