// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"encoding/json"
	"flag"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
)

var logResponses = flag.Bool("log_response", false, "use to log users/groups returned from the API")

// Invoke test with something like this:
//
//	AD_BASE=CN=Users,DC=<servername>,DC=local AD_URL=ldap://<ip> AD_USER=CN=Administrator,CN=Users,DC=<servername>,DC=local AD_PASS=<passwort> go test -v -log_response
func Test(t *testing.T) {
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

	var times []time.Time
	t.Run("full", func(t *testing.T) {
		users, err := GetDetails(url, user, pass, base, time.Time{}, 0, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error from GetDetails: %v", err)
		}

		if len(users) == 0 {
			t.Error("expected non-empty result from query")
		}
		found := false
		var gotUsers []any
		for _, e := range users {
			dn := e.User["distinguishedName"]
			gotUsers = append(gotUsers, dn)
			if dn == user {
				found = true
			}

			when, ok := e.User["whenChanged"].(time.Time)
			if ok {
				times = append(times, when)
			}
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

	t.Run("update", func(t *testing.T) {
		sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
		since := times[0].Add(time.Second) // Step past first entry by a small amount within LDAP resolution.
		var want int
		// ... and count all entries since then.
		for _, when := range times[1:] {
			if !since.After(when) {
				want++
			}
		}
		users, err := GetDetails(url, user, pass, base, since, 0, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error from GetDetails: %v", err)
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
