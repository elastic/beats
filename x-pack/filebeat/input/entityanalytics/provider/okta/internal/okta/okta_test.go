// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package okta provide Okta user API support.
package okta

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/time/rate"
)

var logUsers = flag.Bool("log_user_response", false, "use to allow log users returned from the API")

func Test(t *testing.T) {
	// https://developer.okta.com/docs/reference/core-okta-api/
	host, ok := os.LookupEnv("OKTA_HOST")
	if !ok {
		t.Skip("okta tests require ${OKTA_HOST} to be set")
	}
	// https://help.okta.com/en-us/Content/Topics/Security/API.htm?cshid=Security_API#Security_API
	key, ok := os.LookupEnv("OKTA_TOKEN")
	if !ok {
		t.Skip("okta tests require ${OKTA_TOKEN} to be set")
	}

	// Make a global limiter with the capacity to proceed once.
	limiter := rate.NewLimiter(1, 1)

	// There are a variety of windows, the most conservative is one minute.
	// The rate limit will be adjusted on the second call to the API if
	// window is actually used to rate limit calculations.
	const window = time.Minute

	for _, omit := range []Response{
		OmitNone,
		OmitCredentials,
	} {
		name := "none"
		if omit != OmitNone {
			name = omit.String()
		}
		t.Run(name, func(t *testing.T) {
			var me User
			t.Run("me", func(t *testing.T) {
				query := make(url.Values)
				query.Set("limit", "200")
				users, _, err := GetUserDetails(context.Background(), http.DefaultClient, host, key, "me", query, omit, limiter, window)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(users) != 1 {
					t.Fatalf("unexpected len(users): got:%d want:1", len(users))
				}
				me = users[0]

				if omit&OmitCredentials != 0 && me.Credentials != nil {
					t.Errorf("unexpected credentials with %s: %#v", omit, me.Credentials)
				}

				if !*logUsers {
					return
				}
				b, err := json.Marshal(me)
				if err != nil {
					t.Errorf("failed to marshal user for logging: %v", err)
				}
				t.Logf("user: %s", b)
			})
			if t.Failed() {
				return
			}

			t.Run("user", func(t *testing.T) {
				if me.Profile.Login == "" {
					b, _ := json.Marshal(me)
					t.Skipf("cannot run user test without profile.login field set: %s", b)
				}

				query := make(url.Values)
				query.Set("limit", "200")
				users, _, err := GetUserDetails(context.Background(), http.DefaultClient, host, key, me.Profile.Login, query, omit, limiter, window)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(users) != 1 {
					t.Fatalf("unexpected len(users): got:%d want:1", len(users))
				}
				if !cmp.Equal(me, users[0]) {
					t.Errorf("unexpected result:\n-'me'\n+'%s'\n%s", me.Profile.Login, cmp.Diff(me, users[0]))
				}
			})

			t.Run("all", func(t *testing.T) {
				query := make(url.Values)
				query.Set("limit", "200")
				users, _, err := GetUserDetails(context.Background(), http.DefaultClient, host, key, "", query, omit, limiter, window)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				found := false
				for _, u := range users {
					if cmp.Equal(me, u, cmpopts.IgnoreFields(User{}, "Links")) {
						found = true
					}
				}
				if !found {
					t.Error("failed to find 'me' in user list")
				}

				if !*logUsers {
					return
				}
				b, err := json.Marshal(users)
				if err != nil {
					t.Errorf("failed to marshal users for logging: %v", err)
				}
				t.Logf("users: %s", b)
			})

			t.Run("error", func(t *testing.T) {
				query := make(url.Values)
				query.Set("limit", "200")
				query.Add("search", `not (status pr)`) // This cannot ever be true.
				_, _, err := GetUserDetails(context.Background(), http.DefaultClient, host, key, "", query, omit, limiter, window)
				oktaErr := &Error{}
				if !errors.As(err, &oktaErr) {
					// Don't test the value of the error since it was
					// determined by observation rather than documentation.
					// But log below.
					t.Fatalf("expected Okta API error got: %#v", err)
				}
				t.Logf("actual error: %v", err)
			})
		})
	}
}

func TestLocal(t *testing.T) {
	// Make a global limiter with more capacity than will be set by the mock API.
	// This will show the burst drop.
	limiter := rate.NewLimiter(10, 10)

	// There are a variety of windows, the most conservative is one minute.
	// The rate limit will be adjusted on the second call to the API if
	// window is actually used to rate limit calculations.
	const window = time.Minute

	const (
		key = "token"
		msg = `[{"id":"userid","status":"STATUS","created":"2023-05-14T13:37:20.000Z","activated":null,"statusChanged":"2023-05-15T01:50:30.000Z","lastLogin":"2023-05-15T01:59:20.000Z","lastUpdated":"2023-05-15T01:50:32.000Z","passwordChanged":"2023-05-15T01:50:32.000Z","type":{"id":"typeid"},"profile":{"firstName":"name","lastName":"surname","mobilePhone":null,"secondEmail":null,"login":"name.surname@example.com","email":"name.surname@example.com"},"credentials":{"password":{"value":"secret"},"emails":[{"value":"name.surname@example.com","status":"VERIFIED","type":"PRIMARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://localhost/api/v1/users/userid"}}}]`
	)
	var wantUsers []User
	err := json.Unmarshal([]byte(msg), &wantUsers)
	if err != nil {
		t.Fatalf("failed to unmarshal user data: %v", err)
	}

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := url.Parse(r.RequestURI)
		if err != nil {
			t.Errorf("unexpected error parsing request URI: %v", err)
		}
		if u.Path != "/api/v1/users" {
			t.Errorf("unexpected API endpoint: got:%s want:%s", u.Path, "/api/v1/users")
		}
		if got := r.Header.Get("accept"); got != "application/json" {
			t.Errorf("unexpected Accept header: got:%s want:%s", got, "application/json")
		}
		if got := r.Header.Get("authorization"); got != "SSWS "+key {
			t.Errorf("unexpected Authorization header: got:%s want:%s", got, "SSWS "+key)
		}

		// Leave 49 remaining, reset in one minute.
		w.Header().Add("x-rate-limit-limit", "50")
		w.Header().Add("x-rate-limit-remaining", "49")
		w.Header().Add("x-rate-limit-reset", fmt.Sprint(time.Now().Add(time.Minute).Unix()))

		// Set next link.
		w.Header().Add("link", `<https://localhost/api/v1/users?limit=200&after=opaquevalue>; rel="next"`)

		fmt.Fprintln(w, msg)
	}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Errorf("failed to parse server URL: %v", err)
	}
	host := u.Host

	query := make(url.Values)
	query.Set("limit", "200")
	users, h, err := GetUserDetails(context.Background(), ts.Client(), host, key, "", query, OmitNone, limiter, window)
	if err != nil {
		t.Fatalf("unexpected error from GetUserDetails: %v", err)
	}

	if !cmp.Equal(wantUsers, users) {
		t.Errorf("unexpected result:\n- want\n+ got\n%s", cmp.Diff(wantUsers, users))
	}

	lim := limiter.Limit()
	if lim < 49.0/60.0 || 50.0/60.0 < lim {
		t.Errorf("unexpected rate limit (outside [49/60, 50/60]: %f", lim)
	}
	if limiter.Burst() != 1 { // Set in GetUserDetails.
		t.Errorf("unexpected burst: got:%d want:1", limiter.Burst())
	}

	next, err := Next(h)
	if err != nil {
		t.Errorf("unexpected error from Next: %v", err)
	}
	if query := next.Encode(); query != "after=opaquevalue&limit=200" {
		t.Errorf("unexpected next query: got:%s want:%s", query, "after=opaquevalue&limit=200")
	}
}

var nextTests = []struct {
	header  http.Header
	want    string
	wantErr error
}{
	0: {
		header: http.Header{"Link": []string{
			`<https://yourOktaDomain/api/v1/logs?limit=20>; rel="self"`,
			`<https://yourOktaDomain/api/v1/logs?limit=20&after=1627500044869_1>; rel="next"`,
		}},
		want:    "after=1627500044869_1&limit=20",
		wantErr: nil,
	},
	1: {
		header: http.Header{"Link": []string{
			`<https://yourOktaDomain/api/v1/logs?limit=20>;rel="self"`,
			`<https://yourOktaDomain/api/v1/logs?limit=20&after=1627500044869_1>;rel="next"`,
		}},
		want:    "after=1627500044869_1&limit=20",
		wantErr: nil,
	},
	2: {
		header: http.Header{"Link": []string{
			`<https://yourOktaDomain/api/v1/logs?limit=20>; rel = "self"`,
			`<https://yourOktaDomain/api/v1/logs?limit=20&after=1627500044869_1>; rel = "next"`,
		}},
		want:    "after=1627500044869_1&limit=20",
		wantErr: nil,
	},
	3: {
		header: http.Header{"Link": []string{
			`<https://yourOktaDomain/api/v1/logs?limit=20>; rel="self"`,
		}},
		want:    "",
		wantErr: io.EOF,
	},
}

func TestNext(t *testing.T) {
	for i, test := range nextTests {
		got, err := Next(test.header)
		if err != test.wantErr {
			t.Errorf("unexpected ok result for %d: got:%v want:%v", i, err, test.wantErr)
		}
		if got.Encode() != test.want {
			t.Errorf("unexpected query result for %d: got:%q want:%q", i, got.Encode(), test.want)
		}
	}
}
