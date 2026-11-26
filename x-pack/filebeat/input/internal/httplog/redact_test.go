// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httplog

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

var maskHeaderTests = []struct {
	name      string
	in        http.Header
	sensitive []string
	want      http.Header
}{
	{
		name: "non_canonical_authorization",
		in:   http.Header{"authorization": {"Bearer abc"}},
		want: http.Header{"authorization": {"Bearer REDACTED"}},
	},
	{
		name: "basic_scheme_kept",
		in:   http.Header{"Authorization": {"Basic dXNlcjpwYXNz"}},
		want: http.Header{"Authorization": {"Basic REDACTED"}},
	},
	{
		name: "unknown_scheme_masked",
		in:   http.Header{"Authorization": {"Custom abc"}},
		want: http.Header{"Authorization": {"REDACTED"}},
	},
	{
		name: "no_scheme_masked",
		in:   http.Header{"Authorization": {"rawsecret"}},
		want: http.Header{"Authorization": {"REDACTED"}},
	},
	{
		name: "proxy_authorization",
		in:   http.Header{"Proxy-Authorization": {"Basic abc"}},
		want: http.Header{"Proxy-Authorization": {"Basic REDACTED"}},
	},
	{
		name: "deny_set",
		in: http.Header{
			"Cookie":    {"session=abc"},
			"X-Api-Key": {"k1"},
			"X-Apikeys": {"accessKey=a;secretKey=b"},
		},
		want: http.Header{
			"Cookie":    {"REDACTED"},
			"X-Api-Key": {"REDACTED"},
			"X-Apikeys": {"REDACTED"},
		},
	},
	{
		name: "every_value_of_a_header",
		in:   http.Header{"Set-Cookie": {"a=1", "b=2"}},
		want: http.Header{"Set-Cookie": {"REDACTED", "REDACTED"}},
	},
	{
		name:      "caller_list_ignores_case",
		in:        http.Header{"x-custom": {"v"}},
		sensitive: []string{"X-Custom"},
		want:      http.Header{"x-custom": {"REDACTED"}},
	},
	{
		name: "classifier",
		in:   http.Header{"X-Amz-Security-Token": {"t"}},
		want: http.Header{"X-Amz-Security-Token": {"REDACTED"}},
	},
	{
		name: "non_sensitive_untouched",
		in: http.Header{
			"Accept":       {"*/*"},
			"Content-Type": {"application/json"},
			"User-Agent":   {"test"},
		},
		want: http.Header{
			"Accept":       {"*/*"},
			"Content-Type": {"application/json"},
			"User-Agent":   {"test"},
		},
	},
}

func TestMaskerHeader(t *testing.T) {
	for _, test := range maskHeaderTests {
		t.Run(test.name, func(t *testing.T) {
			before := test.in.Clone()
			got := newMasker(test.sensitive).header(test.in)
			if !cmp.Equal(test.want, got) {
				t.Errorf("unexpected header:\n--- want:\n+++ got:\n%s", cmp.Diff(test.want, got))
			}
			if !cmp.Equal(before, test.in) {
				t.Errorf("input was modified:\n%s", cmp.Diff(before, test.in))
			}
		})
	}
}

var maskURLTests = []struct {
	name         string
	in           string
	wantOriginal string
	wantQuery    string
}{
	{
		name:         "listed_parameter_masked_cursor_kept",
		in:           "https://example.com/p?nextPageToken=abc&key=SECRET&limit=5",
		wantOriginal: "https://example.com/p?key=REDACTED&limit=5&nextPageToken=abc",
		wantQuery:    "key=REDACTED&limit=5&nextPageToken=abc",
	},
	{
		name:         "oauth_and_token_names",
		in:           "https://example.com/?token=a&refresh_token=b&id_token=c&secret=d&assertion=e&limit=1",
		wantOriginal: "https://example.com/?assertion=REDACTED&id_token=REDACTED&limit=1&refresh_token=REDACTED&secret=REDACTED&token=REDACTED",
		wantQuery:    "assertion=REDACTED&id_token=REDACTED&limit=1&refresh_token=REDACTED&secret=REDACTED&token=REDACTED",
	},
	{
		name:         "name_ignores_case",
		in:           "https://example.com/?Access_Token=abc",
		wantOriginal: "https://example.com/?Access_Token=REDACTED",
		wantQuery:    "Access_Token=REDACTED",
	},
	{
		name:         "unmasked_query_is_unchanged",
		in:           "https://example.com/?z=1&a=2",
		wantOriginal: "https://example.com/?z=1&a=2",
		wantQuery:    "z=1&a=2",
	},
	{
		name:         "no_query",
		in:           "https://example.com/p",
		wantOriginal: "https://example.com/p",
		wantQuery:    "",
	},
	{ //nolint:gosec // fake credentials in a test
		name:         "user_and_password",
		in:           "https://user:pass@example.com/x",
		wantOriginal: "https://REDACTED:REDACTED@example.com/x",
		wantQuery:    "",
	},
	{
		name:         "user_only",
		in:           "https://apikey123@example.com/x",
		wantOriginal: "https://REDACTED@example.com/x",
		wantQuery:    "",
	},
	{
		name:      "unparseable_query",
		in:        "https://example.com/?a=%zz",
		wantQuery: unparsedQuery,
	},
}

func TestMaskerURL(t *testing.T) {
	for _, test := range maskURLTests {
		t.Run(test.name, func(t *testing.T) {
			u, err := url.Parse(test.in)
			if err != nil {
				t.Fatalf("failed to parse test URL: %v", err)
			}
			original, query := maskURL(u)
			if test.wantOriginal != "" && original != test.wantOriginal {
				t.Errorf("unexpected original URL: got:%q want:%q", original, test.wantOriginal)
			}
			if query != test.wantQuery {
				t.Errorf("unexpected query: got:%q want:%q", query, test.wantQuery)
			}
			if u.String() != test.in {
				t.Errorf("input URL was modified: got:%q want:%q", u.String(), test.in)
			}
		})
	}
}

const (
	formType = "application/x-www-form-urlencoded"
	jsonType = "application/json"
)

var maskBodyTests = []struct {
	name        string
	contentType string
	body        string
	want        string
}{
	// Forms.
	{
		name:        "form_client_credentials",
		contentType: formType,
		body:        "client_id=id&client_secret=S3CRET&grant_type=client_credentials",
		want:        "client_id=id&client_secret=REDACTED&grant_type=client_credentials",
	},
	{
		name:        "form_password_grant",
		contentType: formType,
		body:        "grant_type=password&password=hunter2&username=u",
		want:        "grant_type=password&password=REDACTED&username=u",
	},
	{
		name:        "form_client_assertion_type_visible",
		contentType: formType,
		body:        "client_assertion=JWT&client_assertion_type=urn%3Aietf%3Aparams%3Aoauth%3Aclient-assertion-type%3Ajwt-bearer",
		want:        "client_assertion=REDACTED&client_assertion_type=urn%3Aietf%3Aparams%3Aoauth%3Aclient-assertion-type%3Ajwt-bearer",
	},
	{
		name:        "json_declared_as_form",
		contentType: formType,
		body:        `{"access_token":"A","ok":"b"}`,
		want:        `{"access_token":"REDACTED","ok":"b"}`,
	},
	{
		name:        "form_missing_content_type",
		contentType: "",
		body:        "client_secret=S&a=b",
		want:        "a=b&client_secret=REDACTED",
	},
	// JSON.
	{
		name:        "json_token_response",
		contentType: jsonType,
		body:        `{"access_token":"A","token_type":"Bearer","expires_in":3600}`,
		want:        `{"access_token":"REDACTED","expires_in":3600,"token_type":"Bearer"}`,
	},
	{
		name:        "json_charset_parameter",
		contentType: "application/json; charset=utf-8",
		body:        `{"access_token":"A"}`,
		want:        `{"access_token":"REDACTED"}`,
	},
	{
		name:        "json_vendor_type",
		contentType: "application/vnd.api+json",
		body:        `{"access_token":"A"}`,
		want:        `{"access_token":"REDACTED"}`,
	},
	{
		name:        "json_jamf_token_response",
		contentType: jsonType,
		body:        `{"token":"T","expires":"2026-01-01T00:00:00Z"}`,
		want:        `{"expires":"2026-01-01T00:00:00Z","token":"REDACTED"}`,
	},
	{
		name:        "json_nested_at_any_depth",
		contentType: jsonType,
		body:        `{"data":{"access_token":"A","ok":"b"},"items":[{"access_token":"C"}]}`,
		want:        `{"data":{"access_token":"REDACTED","ok":"b"},"items":[{"access_token":"REDACTED"}]}`,
	},
	{
		name:        "json_array_root",
		contentType: jsonType,
		body:        `[{"access_token":"A"},{"ok":1}]`,
		want:        `[{"access_token":"REDACTED"},{"ok":1}]`,
	},
	{
		name:        "json_below_a_listed_name",
		contentType: jsonType,
		body:        `{"password":{"value":"a","hint":"b"}}`,
		want:        `{"password":{"hint":"REDACTED","value":"REDACTED"}}`,
	},
	{
		name:        "json_numbers_and_html_kept",
		contentType: jsonType,
		body:        `{"password":"p","big":12345678901234567890,"u":"a<b&c"}`,
		want:        `{"big":12345678901234567890,"password":"REDACTED","u":"a<b&c"}`,
	},
	{
		name:        "json_number_under_listed_name",
		contentType: jsonType,
		body:        `{"password":123456789}`,
		want:        `{"password":"REDACTED"}`,
	},
	{
		name:        "ndjson",
		contentType: "application/x-ndjson",
		body:        "{\"token\":\"a\"}\n{\"n\":1}\n",
		want:        "{\"token\":\"REDACTED\"}\n{\"n\":1}",
	},
	{
		name:        "concatenated_json_with_json_type",
		contentType: jsonType,
		body:        `{"token":"a"}{"n":1}`,
		want:        "{\"token\":\"REDACTED\"}\n{\"n\":1}",
	},
	{
		name:        "json_in_text_plain",
		contentType: "text/plain",
		body:        `{"password":"p"}`,
		want:        `{"password":"REDACTED"}`,
	},
	{
		name:        "json_missing_content_type",
		contentType: "",
		body:        `{"password":"p"}`,
		want:        `{"password":"REDACTED"}`,
	},
	// Bodies that stay as they are.
	{
		name:        "json_error_body_with_word_token_unchanged",
		contentType: jsonType,
		body:        `{"error":"invalid_client","error_description":"Bad token provided","code":401}`,
		want:        `{"error":"invalid_client","error_description":"Bad token provided","code":401}`,
	},
	{
		name:        "json_token_type_unchanged",
		contentType: jsonType,
		body:        `{"token_type":"Bearer"}`,
		want:        `{"token_type":"Bearer"}`,
	},
	{
		name:        "json_code_field_unchanged",
		contentType: jsonType,
		body:        `{"error":{"code":"forbidden","message":"no"},"secretary":"x"}`,
		want:        `{"error":{"code":"forbidden","message":"no"},"secretary":"x"}`,
	},
	{
		name:        "html_with_listed_word_unchanged",
		contentType: "text/html",
		body:        "<p>Enter your password</p>",
		want:        "<p>Enter your password</p>",
	},
	{
		name:        "text_with_listed_word_unchanged",
		contentType: "text/plain",
		body:        "The access token is invalid",
		want:        "The access token is invalid",
	},
	{
		name:        "unparseable_json_without_listed_name_unchanged",
		contentType: jsonType,
		body:        `{"a":`,
		want:        `{"a":`,
	},
	{
		name:        "empty_body",
		contentType: jsonType,
		body:        "",
		want:        "",
	},
	// Bodies that are replaced.
	{
		name:        "unparseable_json_with_listed_name",
		contentType: jsonType,
		body:        `{"access_token":"A"`,
		want:        unparsedBody,
	},
	{
		name:        "unparseable_form_with_listed_name",
		contentType: formType,
		body:        "client_secret=S&a=%zz",
		want:        unparsedBody,
	},
}

func TestMaskBody(t *testing.T) {
	for _, test := range maskBodyTests {
		t.Run(test.name, func(t *testing.T) {
			got := string(maskBody(test.contentType, []byte(test.body)))
			if got != test.want {
				t.Errorf("maskBody(%q, %q) = %q; want %q", test.contentType, test.body, got, test.want)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newTestTracer(t *testing.T, maxBodyLen int, next http.RoundTripper) (*LoggingRoundTripper, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.DebugLevel,
	)
	log := logptest.NewTestingLogger(t, "test")
	return NewLoggingRoundTripper(next, zap.New(core), maxBodyLen, []string{"Authorization"}, log), &buf
}

func response(status int, contentType, body string, header http.Header) *http.Response {
	h := header.Clone()
	if h == nil {
		h = http.Header{}
	}
	h.Set("Content-Type", contentType)
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestLoggingRoundTripperMasksCredentials(t *testing.T) {
	const (
		secret = "FAKE-SECRET-7Q2" //nolint:gosec // fake credential in a test
		token  = "FAKE-TOKEN-9K1"  //nolint:gosec // fake credential in a test
	)
	tests := []struct {
		name        string
		req         func() *http.Request
		status      int
		contentType string
		body        string
		header      http.Header
		wantAbsent  []string
		wantPresent []string
	}{
		{
			name: "token_request_and_error_response",
			req: func() *http.Request {
				body := "client_id=id&client_secret=" + secret + "&grant_type=client_credentials"
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/token", strings.NewReader(body))
				req.Header.Set("Content-Type", formType)
				req.Header.Set("Authorization", "Basic "+secret)
				return req
			},
			status:      http.StatusUnauthorized,
			contentType: jsonType,
			body:        `{"error":"invalid_client"}`,
			header:      http.Header{"Set-Cookie": {"s=" + token}},
			wantAbsent:  []string{secret, token},
			wantPresent: []string{"client_secret=REDACTED", "invalid_client", `"http.response.status_code":401`, "Basic REDACTED"},
		},
		{
			name: "token_response",
			req: func() *http.Request {
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/token", strings.NewReader("grant_type=client_credentials"))
				req.Header.Set("Content-Type", formType)
				return req
			},
			status:      http.StatusOK,
			contentType: jsonType,
			body:        `{"access_token":"` + token + `","token_type":"Bearer","expires_in":3600}`,
			wantAbsent:  []string{token},
			wantPresent: []string{"token_type", "Bearer", "expires_in", `"http.response.status_code":200`},
		},
		{
			name: "query_parameter_and_user_info",
			req: func() *http.Request {
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://user:"+secret+"@example.com/api?key="+token+"&nextPageToken=cursor", nil)
				return req
			},
			status:      http.StatusOK,
			contentType: jsonType,
			body:        `{}`,
			wantAbsent:  []string{secret, token},
			wantPresent: []string{"nextPageToken=cursor", "key=REDACTED"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			next := roundTripFunc(func(*http.Request) (*http.Response, error) {
				return response(test.status, test.contentType, test.body, test.header), nil
			})
			rt, buf := newTestTracer(t, 1<<20, next)
			resp, err := rt.RoundTrip(test.req())
			if err != nil {
				t.Fatalf("unexpected error from RoundTrip: %v", err)
			}
			defer resp.Body.Close()
			_, err = io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response body: %v", err)
			}
			out := buf.String()
			for _, s := range test.wantAbsent {
				if strings.Contains(out, s) {
					t.Errorf("trace must not contain %q:\n%s", s, out)
				}
			}
			for _, s := range test.wantPresent {
				if !strings.Contains(out, s) {
					t.Errorf("trace must contain %q:\n%s", s, out)
				}
			}
		})
	}
}

func TestLoggingRoundTripperDoesNotChangeTraffic(t *testing.T) {
	const reqBody = "client_secret=S3CRET&grant_type=client_credentials"
	const respBody = `{"access_token":"TOKEN"}`
	var gotReq string
	next := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		gotReq = string(b)
		return response(http.StatusOK, jsonType, respBody, nil), nil
	})
	rt, _ := newTestTracer(t, 1<<20, next)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/token", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", formType)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error from RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if gotReq != reqBody {
		t.Errorf("request body sent = %q; want %q", gotReq, reqBody)
	}
	if string(got) != respBody {
		t.Errorf("response body received = %q; want %q", got, respBody)
	}
}

func TestLogRequestMasksBeforeTruncating(t *testing.T) {
	const secret = "FAKE-SECRET-7Q2" //nolint:gosec // fake credential in a test
	body := `{"pad":"` + strings.Repeat("x", 64) + `","client_secret":"` + secret + `"}`

	var buf bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.DebugLevel,
	)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/x", strings.NewReader(body))
	req.Header.Set("Content-Type", jsonType)
	LogRequest(zap.New(core), req, nil, 100)

	out := buf.String()
	if strings.Contains(out, secret) {
		t.Errorf("trace must not contain the secret:\n%s", out)
	}
	if !strings.Contains(out, `"http.request.body.truncated":true`) {
		t.Errorf("trace must report the body as truncated:\n%s", out)
	}
	if !strings.Contains(out, `"http.request.body.bytes":`+strconv.Itoa(len(body))) {
		t.Errorf("trace must report the original body length %d:\n%s", len(body), out)
	}
}

// TestAutoDetectRetryDoesNotTraceCredentials reproduces the case where the
// client credentials are sent in a Basic header, rejected, and then sent in
// the form body. Neither the secret nor the access token may be traced.
func TestAutoDetectRetryDoesNotTraceCredentials(t *testing.T) {
	const (
		clientID = "tracer-probe-id"
		secret   = "FAKE-SECRET-tracer-probe-7Q2" //nolint:gosec // fake credential in a test
		token    = "FAKE-ACCESS-TOKEN-probe-9K1"
	)

	var calls []string
	server := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/token" {
			calls = append(calls, "api: Authorization="+req.Header.Get("Authorization"))
			return response(http.StatusOK, jsonType, `{"ok":true}`, nil), nil
		}
		b, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(b))
		_ = req.ParseForm()
		if _, _, basic := req.BasicAuth(); basic {
			calls = append(calls, "token: Basic header -> 401")
			return response(http.StatusUnauthorized, jsonType, `{"error":"invalid_client"}`, nil), nil
		}
		if req.PostForm.Get("client_id") == clientID && req.PostForm.Get("client_secret") == secret {
			calls = append(calls, "token: form body credentials -> 200")
			return response(http.StatusOK, jsonType, `{"access_token":"`+token+`","token_type":"Bearer","expires_in":3600}`, nil), nil
		}
		calls = append(calls, "token: no credentials -> 401")
		return response(http.StatusUnauthorized, jsonType, `{"error":"invalid_client"}`, nil), nil
	})

	rt, buf := newTestTracer(t, 1<<20, server)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{Transport: rt})
	creds := clientcredentials.Config{ //nolint:gosec // fake credentials in a test
		ClientID:       clientID,
		ClientSecret:   secret,
		TokenURL:       "https://example.test/token",
		EndpointParams: map[string][]string{"grant_type": {"client_credentials"}},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.test/api", nil)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	resp, err := creds.Client(ctx).Do(req)
	if err != nil {
		t.Fatalf("unexpected error from request: %v", err)
	}
	resp.Body.Close()

	want := []string{
		"token: Basic header -> 401",
		"token: form body credentials -> 200",
		"api: Authorization=Bearer " + token,
	}
	if !cmp.Equal(want, calls) {
		t.Fatalf("unexpected server calls (the retry did not happen as expected):\n--- want:\n+++ got:\n%s", cmp.Diff(want, calls))
	}

	out := buf.String()
	for _, s := range []string{secret, token} {
		if strings.Contains(out, s) {
			t.Errorf("trace must not contain %q:\n%s", s, out)
		}
	}
	for _, s := range []string{"client_secret=REDACTED", "invalid_client", `"http.response.status_code":401`, "Bearer REDACTED"} {
		if !strings.Contains(out, s) {
			t.Errorf("trace must contain %q:\n%s", s, out)
		}
	}
}

func TestLogRequestTruncationIsAboutTheMaskedBody(t *testing.T) {
	// Masking shrinks this body from well over the limit to under it.
	body := `{"client_secret":"` + strings.Repeat("s", 200) + `"}`

	var buf bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.DebugLevel,
	)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://example.com/x", strings.NewReader(body))
	req.Header.Set("Content-Type", jsonType)
	LogRequest(zap.New(core), req, nil, 100)

	out := buf.String()
	if !strings.Contains(out, `"http.request.body.truncated":false`) {
		t.Errorf("trace must report the masked body as not truncated:\n%s", out)
	}
	if !strings.Contains(out, `"http.request.body.bytes":`+strconv.Itoa(len(body))) {
		t.Errorf("trace must report the original body length %d:\n%s", len(body), out)
	}
	if !strings.Contains(out, `{\"client_secret\":\"REDACTED\"}`) {
		t.Errorf("trace must contain the masked body:\n%s", out)
	}
}
