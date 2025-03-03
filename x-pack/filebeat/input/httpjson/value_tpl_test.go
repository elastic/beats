// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/useragent"
)

func TestValueTpl(t *testing.T) {
	logp.TestingSetup()

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
			name:  "can access Go types in context",
			value: `[[.last_response.header.Get "foo"]] [[.last_response.url.params.Get "foo"]] [[.url.Host]] [[.url.Query.Get "bar"]]`,
			paramCtx: &transformContext{
				firstEvent:   &mapstr.M{},
				lastEvent:    &mapstr.M{},
				lastResponse: newTestResponse(mapstr.M{"param": 25}, http.Header{"Foo": []string{"bar"}}, "http://localhost?foo=bar"),
			},
			paramTr:     transformable{"url": newURL("http://localhost?bar=bazz")},
			paramDefVal: "",
			expectedVal: "bar bar localhost bazz",
		},
		{
			name:  "can render values from ctx",
			value: "[[.last_response.body.param]]",
			paramCtx: &transformContext{
				firstEvent:   &mapstr.M{},
				lastEvent:    &mapstr.M{},
				lastResponse: newTestResponse(mapstr.M{"param": 25}, nil, ""),
			},
			paramTr:     transformable{},
			paramDefVal: "",
			expectedVal: "25",
		},
		{
			name:  "can render default value if execute fails",
			value: "[[.last_response.body.does_not_exist]]",
			paramCtx: &transformContext{
				lastEvent: &mapstr.M{},
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
			name:        "func parseDateInTZ with RFC3339Nano and timezone offset",
			value:       `[[ parseDateInTZ "2020-11-05T12:25:32.1234567Z" "-0700" "RFC3339Nano" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 19:25:32.1234567 +0000 UTC",
		},
		{
			name:        "func parseDateInTZ defaults to RFC3339 with implicit offset and timezone",
			value:       `[[ parseDateInTZ "2020-11-05T12:25:32+04:00" "-0700" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 19:25:32 +0000 UTC",
		},
		{
			name:        "func parseDateInTZ defaults to RFC3339 with IANA timezone",
			value:       `[[ parseDateInTZ "2020-11-05T12:25:32Z" "America/New_York" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 17:25:32 +0000 UTC",
		},
		{
			name:        "func parseDateInTZ with custom layout and timezone name",
			value:       `[[ parseDateInTZ "Thu Nov  5 12:25:32 2020" "Europe/Paris" "Mon Jan _2 15:04:05 2006" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020-11-05 11:25:32 +0000 UTC",
		},
		{
			name:        "func parseDateInTZ with invalid timezone",
			value:       `[[ parseDateInTZ "2020-11-05T12:25:32Z" "Invalid/Timezone" ]]`,
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
			name:  "func getRFC5988Link single rel matches",
			value: `[[ getRFC5988Link "next" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
				lastResponse: newTestResponse(
					nil,
					http.Header{"Link": []string{
						`<https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK>; title="Page 3"; rel="next"`,
					}},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: "https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK",
		},
		{
			name:  "func getRFC5988Link multiple rel as separate strings matches",
			value: `[[ getRFC5988Link "previous" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
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
			name:  "func getRFC5988Link multiple rel as separate strings in random order matches",
			value: `[[ getRFC5988Link "previous" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
				lastResponse: newTestResponse(
					nil,
					http.Header{"Link": []string{
						`<https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK>; title="Page 1"; rel="previous"`,
						`<https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK>; title="Page 3"; rel="next"`,
					}},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: "https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK",
		},
		{
			name:  "func getRFC5988Link multiple rel as single string matches",
			value: `[[ getRFC5988Link "previous" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
				lastResponse: newTestResponse(
					nil,
					http.Header{"Link": []string{
						`<https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK>; title="Page 1"; rel="previous",
						<https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK>; title="Page 3"; rel="next"`,
					}},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: "https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK",
		},
		{
			name:  "func getRFC5988Link multiple rel as single string in random order matches",
			value: `[[ getRFC5988Link "next" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
				lastResponse: newTestResponse(
					nil,
					http.Header{"Link": []string{
						`<https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK>; title="Page 1"; rel="previous",
						<https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK>; title="Page 3"; rel="next"`,
					}},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: "https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK",
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
			name:  "func getRFC5988Link no space link parameters",
			value: `[[ getRFC5988Link "next" .last_response.header.Link ]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
				lastResponse: newTestResponse(
					nil,
					http.Header{"Link": []string{
						`<https://example.com/api/v1/users?before=00ubfjQEMYBLRUWIEDKK>;title="Page 1";rel="previous",
						<https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK>;title="Page 3";rel="next"`,
					}},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: "https://example.com/api/v1/users?after=00ubfjQEMYBLRUWIEDKK",
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
		{
			name:        "func min int",
			value:       `[[min 4 1]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "1",
		},
		{
			name:        "func max int",
			value:       `[[max 4 1]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "4",
		},
		{
			name:        "func max float",
			value:       `[[max 1.23 4.666]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "4.666",
		},
		{
			name:        "func min float",
			value:       `[[min 1.23 4.666]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "1.23",
		},
		{
			name:        "func min string",
			value:       `[[min "a" "b"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "a",
		},
		{
			name:        "func max string",
			value:       `[[max "a" "b"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "b",
		},
		{
			name:        "func min int64 unix seconds",
			value:       `[[ min (now.Unix) 1689771139 ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "1689771139",
		},
		{
			name:        "func min int year",
			value:       `[[ min (now.Year) 2020 ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020",
		},
		{
			name:        "func max duration",
			value:       `[[ max (parseDuration "59m") (parseDuration "1h") ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "1h0m0s",
		},
		{
			name:        "func min int ",
			value:       `[[ min (now.Year) 2020 ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2020",
		},
		{
			name:        "func sha1 hmac Hex",
			value:       `[[hmac "sha1" "secret" "string1" "string2"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "87eca1e7cba012b2dd4a907c2ad4345a252a38f4",
		},
		{
			name:        "func sha256 hmac Hex",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1627697597, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[hmac "sha256" "secret" "string1" "string2" (formatDate (now) "RFC1123")]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "adc61cd206e146f2d1337504e760ea70f3d2e34bedf28d07802e0e776568a06b",
		},
		{
			name:          "func invalid hmac Hex",
			value:         `[[hmac "md5" "secret" "string1" "string2"]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "func sha1 hash Hex empty",
			value:       `[[hash "sha1"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:        "func sha1 hash Hex",
			value:       `[[hash "sha1" "string1" "string2"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "6b3966bea9fe56d1f9708517fd22b70c682b8a3d",
		},
		{
			name:        "func sha256 hash Hex",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1627697597, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[hash "sha256" "string1" "string2" (formatDate (now) "RFC1123")]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "b0a92a08a9b4883aa3aa2d0957be12a678cbdbb32dc5db09fe68239a09872f96",
		},
		{
			name:          "func invalid hash Hex",
			value:         `[[hash "md5" "string1" "string2"]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "func base64Encode 2 strings",
			value:       `[[base64Encode "string1" "string2"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "c3RyaW5nMXN0cmluZzI=",
		},
		{
			name:          "func base64Encode no value",
			value:         `[[base64Encode ""]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "func hexDecode string",
			value:       `[[hexDecode "b0a92a08a9b4883aa3aa2d0957be12a678cbdbb32dc5db09fe68239a09872f96"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "\xb0\xa9*\b\xa9\xb4\x88:\xa3\xaa-\tW\xbe\x12\xa6x\xcb۳-\xc5\xdb\t\xfeh#\x9a\t\x87/\x96",
		},
		{
			name:          "func hexDecode empty string",
			value:         `[[hexDecode ""]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:          "func invalid hexDecode string",
			value:         `[[hexDecode "abcdefghijklmnopqrstuvwxyz"]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name: "func join",
			value: `[[join .last_response.body.strarr ","]] [[join .last_response.body.iarr ","]] ` +
				`[[join .last_response.body.narr ","]] [[join .last_response.body.singlevalstr ","]] ` +
				`[[join .last_response.body.singlevalint ","]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
				lastResponse: newTestResponse(
					mapstr.M{
						"strarr": []string{
							"foo",
							"bar",
						},
						"iarr": []interface{}{
							"foo",
							2,
						},
						"narr": []int{
							1,
							2,
						},
						"singlevalstr": "foo",
						"singlevalint": 2,
					},
					http.Header{},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: "foo,bar foo,2 1,2 foo 2",
		},
		{
			name:  "func sprintf",
			value: `[[sprintf "%q:%d" (join .last_response.body.arr ",") 1]]`,
			paramCtx: &transformContext{
				firstEvent: &mapstr.M{},
				lastEvent:  &mapstr.M{},
				lastResponse: newTestResponse(
					mapstr.M{
						"arr": []string{
							"foo",
							"bar",
						},
					},
					http.Header{},
					"",
				),
			},
			paramTr:     transformable{},
			expectedVal: `"foo,bar":1`,
		},
		{
			name:        "func sha1 hmac Base64",
			value:       `[[hmacBase64 "sha1" "secret" "string1" "string2"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "h+yh58ugErLdSpB8KtQ0WiUqOPQ=",
		},
		{
			name:        "func sha256 hmac Base64",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1627697597, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[hmacBase64 "sha256" "secret" "string1" "string2"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "HlglO6yRZs0Ts3MjmgnRKtTJk3fr9nt8LmeliVKZyAA=",
		},
		{
			name:          "func invalid hmac Base64",
			value:         `[[hmacBase64 "md5" "secret" "string1" "string2"]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "func sha1 empty",
			value:       `[[hashBase64 "sha1"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2jmj7l5rSw0yVb/vlWAYkK/YBwk=",
		},
		{
			name:        "func sha1 hash Base64",
			value:       `[[hashBase64 "sha1" "string1" "string2"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "azlmvqn+VtH5cIUX/SK3DGgrij0=",
		},
		{
			name:        "func sha256 hash Base64",
			setup:       func() { timeNow = func() time.Time { return time.Unix(1627697597, 0).UTC() } },
			teardown:    func() { timeNow = time.Now },
			value:       `[[hashBase64 "sha256" "string1" "string2"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "usCapy5jLnbDbmwcTlArc8Paf8poxHUnPcVReBVYfMQ=",
		},
		{
			name:          "func invalid hmac Base64",
			value:         `[[hmacBase64 "md5" "string1" "string2"]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "func base64Decode 2 strings",
			value:       `[[base64Decode "c3RyaW5nMXN0cmluZzI="]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "string1string2",
		},
		{
			name:          "func base64Decode no value",
			value:         `[[base64Decode ""]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "func userAgent no values",
			value:       `[[userAgent]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String()),
		},
		{
			name:        "func userAgent blank value",
			value:       `[[userAgent ""]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String()),
		},
		{
			name:        "func userAgent 1 value",
			value:       `[[userAgent "integration_name/1.2.3"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String(), "integration_name/1.2.3"),
		},
		{
			name:        "func userAgent 2 value",
			value:       `[[userAgent "integration_name/1.2.3" "test"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: useragent.UserAgent("Filebeat", version.GetDefaultVersion(), version.Commit(), version.BuildTime().String(), "integration_name/1.2.3", "test"),
		},
		{
			name:        "func beatInfo GOOS",
			value:       `[[beatInfo.goos]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: runtime.GOOS,
		},
		{
			name:        "func beatInfo Arch",
			value:       `[[beatInfo.goarch]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: runtime.GOARCH,
		},
		{
			name:        "func beatInfo Commit",
			value:       `[[beatInfo.commit]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: version.Commit(),
		},
		{
			name:        "func beatInfo Build Time",
			value:       `[[beatInfo.buildtime]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: version.BuildTime().String(),
		},
		{
			name:        "func beatInfo Version",
			value:       `[[beatInfo.version]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: version.GetDefaultVersion(),
		},
		{
			name:          "func urlEncode blank",
			value:         `[[urlEncode ""]]`,
			paramCtx:      emptyTransformContext(),
			paramTr:       transformable{},
			expectedVal:   "",
			expectedError: errEmptyTemplateResult.Error(),
		},
		{
			name:        "func urlEncode URL Safe",
			value:       `[[urlEncode "asdf"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "asdf",
		},
		{
			name:        "func urlEncode URL Safe",
			value:       `[[urlEncode "2022-02-17T04:37:10.406+0000"]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "2022-02-17T04%3A37%3A10.406%2B0000",
		},
		{
			name:        "func replaceAll",
			value:       `[[ "some value" | replaceAll "some" "my" ]]`,
			paramCtx:    emptyTransformContext(),
			paramTr:     transformable{},
			expectedVal: "my value",
		},
		{
			name:  "func toJSON",
			value: "[[ toJSON .first_event.events ]]",
			paramCtx: &transformContext{
				firstEvent:   &mapstr.M{"events": []interface{}{map[string]interface{}{"id": 1234}}},
				lastEvent:    &mapstr.M{},
				lastResponse: newTestResponse(nil, nil, ""),
			},
			paramTr:     transformable{},
			expectedVal: `[{"id":1234}]`,
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

			got, err := tpl.Execute(tc.paramCtx, tc.paramTr, tc.name, defTpl, logp.NewLogger(""))
			assert.Equal(t, tc.expectedVal, got)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}

func newTestResponse(body mapstr.M, header http.Header, url string) *response {
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
		resp.url = *(newURL(url))
	}
	return resp
}
