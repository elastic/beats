// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/elastic/elastic-agent-libs/logp"
)

var logResponses = flag.Bool("log_response", false, "use to log users/groups returned from the API")

func TestActiveDirectoryDoFetch(t *testing.T) {
	url, ok := os.LookupEnv("AD_URL")
	if !ok {
		t.Skip("activedirectory tests require ${AD_URL} to be set")
	}
	baseDN, ok := os.LookupEnv("AD_BASE")
	if !ok {
		t.Skip("activedirectory tests require ${AD_BASE} to be set")
	}
	user, ok := os.LookupEnv("AD_USER")
	if !ok {
		t.Skip("activedirectory tests require ${AD_USER} to be set")
	}
	pass, ok := os.LookupEnv("AD_PASS")
	if !ok {
		t.Skip("activedirectory tests require ${AD_PASS} to be set")
	}

	base, err := ldap.ParseDN(baseDN)
	if err != nil {
		t.Fatalf("invalid base distinguished name: %v", err)
	}

	const dbFilename = "TestActiveDirectoryDoFetch.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})
	a := adInput{
		cfg: conf{
			BaseDN:   baseDN,
			URL:      url,
			User:     user,
			Password: pass,
		},
		baseDN: base,
		logger: logp.L(),
	}

	ss, err := newStateStore(store)
	if err != nil {
		t.Fatalf("unexpected error making state store: %v", err)
	}
	defer ss.close(false)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var times []time.Time
	t.Run("full", func(t *testing.T) {
		ss.whenChanged = time.Time{} // Reach back to the start of time.

		users, err := a.doFetchUsers(ctx, ss, false) // We are lying about fullSync since we are not getting users via the store.
		if err != nil {
			t.Fatalf("unexpected error from doFetch: %v", err)
		}

		if len(users) == 0 {
			t.Error("expected non-empty result from query")
		}
		found := false
		var gotUsers []string
		for _, e := range users {
			gotUsers = append(gotUsers, e.ID)
			if e.ID == user {
				found = true
			}

			times = append(times, e.WhenChanged)
		}
		if !found {
			t.Errorf("expected login user to be found in directory: got:%q", gotUsers)
		}

		if !*logResponses {
			return
		}
		b, err := json.MarshalIndent(users, "", "\t")
		if err != nil {
			t.Errorf("failed to marshal users for logging: %v", err)
		}
		t.Logf("user: %s", b)
	})

	// Find the time of the first changed entry for later.
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	since := times[0].Add(time.Second) // Step past first entry by a small amount within LDAP resolution.
	var want int
	// ... and count all entries since then.
	for _, when := range times[1:] {
		if !since.After(when) {
			want++
		}
	}

	t.Run("update", func(t *testing.T) {
		ss.whenChanged = since // Reach back until after the first entry.

		users, err := a.doFetchUsers(ctx, ss, false)
		if err != nil {
			t.Fatalf("unexpected error from doFetchUsers: %v", err)
		}

		if len(users) != want {
			t.Errorf("unexpected number of results from query since %v: got:%d want:%d", since, len(users), want)
		}

		if !*logResponses && !t.Failed() {
			return
		}
		b, err := json.MarshalIndent(users, "", "\t")
		if err != nil {
			t.Errorf("failed to marshal users for logging: %v", err)
		}
		t.Logf("user: %s", b)
	})
}
