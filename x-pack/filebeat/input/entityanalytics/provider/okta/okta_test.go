// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestOktaDoFetch(t *testing.T) {
	dbFilename := "TestOktaDoFetch.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	const (
		window = time.Minute
		key    = "token"
		msg    = `[{"id":"userid","status":"STATUS","created":"2023-05-14T13:37:20.000Z","activated":null,"statusChanged":"2023-05-15T01:50:30.000Z","lastLogin":"2023-05-15T01:59:20.000Z","lastUpdated":"2023-05-15T01:50:32.000Z","passwordChanged":"2023-05-15T01:50:32.000Z","type":{"id":"typeid"},"profile":{"firstName":"name","lastName":"surname","mobilePhone":null,"secondEmail":null,"login":"name.surname@example.com","email":"name.surname@example.com"},"credentials":{"password":{"value":"secret"},"emails":[{"value":"name.surname@example.com","status":"VERIFIED","type":"PRIMARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://localhost/api/v1/users/userid"}}}]`
	)

	var wantUsers []User
	err := json.Unmarshal([]byte(msg), &wantUsers)
	if err != nil {
		t.Fatalf("failed to unmarshal user data: %v", err)
	}

	wantUserStates := make(map[string]State)

	// Set the number of repeats.
	const repeats = 3
	var n int
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Leave 49 remaining, reset in one minute.
		w.Header().Add("x-rate-limit-limit", "50")
		w.Header().Add("x-rate-limit-remaining", "49")
		w.Header().Add("x-rate-limit-reset", fmt.Sprint(time.Now().Add(time.Minute).Unix()))

		// Set next link if we can still repeat.
		n++
		if n < repeats {
			w.Header().Add("link", `<https://localhost/api/v1/users?limit=200&after=opaquevalue>; rel="next"`)
		}

		userid := fmt.Sprintf("userid%d", n)

		// Store expected states. The State values are all Discovered
		// unless the user is deleted since they are all first appearance.
		states := []string{
			"ACTIVE",
			"RECOVERY",
			"DEPROVISIONED",
		}
		status := states[n%len(states)]
		state := Discovered
		if status == "DEPROVISIONED" {
			state = Deleted
		}
		wantUserStates[userid] = state

		replacer := strings.NewReplacer(
			"userid", userid,
			"STATUS", status,
		)
		fmt.Fprintln(w, replacer.Replace(msg))
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Errorf("failed to parse server URL: %v", err)
	}
	a := oktaInput{
		cfg: conf{
			OktaDomain: u.Host,
			OktaToken:  key,
		},
		client: ts.Client(),
		lim:    rate.NewLimiter(1, 1),
		logger: logp.L(),
	}

	ss, err := newStateStore(store)
	if err != nil {
		t.Fatalf("unexpected error making state store: %v", err)
	}
	defer ss.close(false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := a.doFetch(ctx, ss, false)
	if err != nil {
		t.Fatalf("unexpected error from doFetch: %v", err)
	}

	if len(got) != repeats {
		t.Errorf("unexpected number of results: got:%d want:%d", len(got), repeats)
	}
	for i, g := range got {
		if wantID := fmt.Sprintf("userid%d", i+1); g.ID != wantID {
			t.Errorf("unexpected user ID for user %d: got:%s want:%s", i, g.ID, wantID)
		}
		if g.State != wantUserStates[g.ID] {
			t.Errorf("unexpected user ID for user %s: got:%s want:%s", g.ID, g.State, wantUserStates[g.ID])
		}
	}
}
