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

func TestParseBaseDN(t *testing.T) {
	// The ldap library normalizes attribute types to lowercase when
	// serializing DNs, so expected values use lowercase cn, ou, dc.
	tests := []struct {
		name                  string
		baseDN                string
		wantContainerBaseDN   string
		wantPotentialGroupDNs []string
		wantOriginalBaseDN    string
	}{
		{
			name:                  "OU only - no potential groups",
			baseDN:                "OU=Users,DC=example,DC=com",
			wantContainerBaseDN:   "ou=Users,dc=example,dc=com",
			wantPotentialGroupDNs: nil,
			wantOriginalBaseDN:    "ou=Users,dc=example,dc=com",
		},
		{
			name:                  "DC only - no potential groups",
			baseDN:                "DC=example,DC=com",
			wantContainerBaseDN:   "dc=example,dc=com",
			wantPotentialGroupDNs: nil,
			wantOriginalBaseDN:    "dc=example,dc=com",
		},
		{
			name:                  "CN before OU - extracts potential group",
			baseDN:                "CN=Admin Users,OU=Groups,DC=example,DC=com",
			wantContainerBaseDN:   "ou=Groups,dc=example,dc=com",
			wantPotentialGroupDNs: []string{"cn=Admin Users,ou=Groups,dc=example,dc=com"},
			wantOriginalBaseDN:    "cn=Admin Users,ou=Groups,dc=example,dc=com",
		},
		{
			name:                  "CN before DC - extracts potential group",
			baseDN:                "CN=Domain Admins,DC=example,DC=com",
			wantContainerBaseDN:   "dc=example,dc=com",
			wantPotentialGroupDNs: []string{"cn=Domain Admins,dc=example,dc=com"},
			wantOriginalBaseDN:    "cn=Domain Admins,dc=example,dc=com",
		},
		{
			name:                  "nested OU - no potential groups",
			baseDN:                "OU=IT,OU=Departments,DC=example,DC=com",
			wantContainerBaseDN:   "ou=IT,ou=Departments,dc=example,dc=com",
			wantPotentialGroupDNs: nil,
			wantOriginalBaseDN:    "ou=IT,ou=Departments,dc=example,dc=com",
		},
		{
			name:                  "CN Users container - extracts as potential group",
			baseDN:                "CN=Users,DC=example,DC=com",
			wantContainerBaseDN:   "dc=example,dc=com",
			wantPotentialGroupDNs: []string{"cn=Users,dc=example,dc=com"},
			wantOriginalBaseDN:    "cn=Users,dc=example,dc=com",
		},
		{
			name:                  "complex path with CN",
			baseDN:                "CN=Security Team,OU=IT Groups,OU=Groups,DC=corp,DC=example,DC=com",
			wantContainerBaseDN:   "ou=IT Groups,ou=Groups,dc=corp,dc=example,dc=com",
			wantPotentialGroupDNs: []string{"cn=Security Team,ou=IT Groups,ou=Groups,dc=corp,dc=example,dc=com"},
			wantOriginalBaseDN:    "cn=Security Team,ou=IT Groups,ou=Groups,dc=corp,dc=example,dc=com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, err := ldap.ParseDN(tt.baseDN)
			if err != nil {
				t.Fatalf("failed to parse test DN %q: %v", tt.baseDN, err)
			}

			got := parseBaseDN(base)

			if got.containerBaseDN != tt.wantContainerBaseDN {
				t.Errorf("parseBaseDN() containerBaseDN = %q, want %q", got.containerBaseDN, tt.wantContainerBaseDN)
			}
			if got.originalBaseDN != tt.wantOriginalBaseDN {
				t.Errorf("parseBaseDN() originalBaseDN = %q, want %q", got.originalBaseDN, tt.wantOriginalBaseDN)
			}
			if len(got.potentialGroupDNs) != len(tt.wantPotentialGroupDNs) {
				t.Errorf("parseBaseDN() potentialGroupDNs length = %d, want %d", len(got.potentialGroupDNs), len(tt.wantPotentialGroupDNs))
			} else {
				for i, dn := range got.potentialGroupDNs {
					if dn != tt.wantPotentialGroupDNs[i] {
						t.Errorf("parseBaseDN() potentialGroupDNs[%d] = %q, want %q", i, dn, tt.wantPotentialGroupDNs[i])
					}
				}
			}
		})
	}
}

func TestParseBaseDNNil(t *testing.T) {
	got := parseBaseDN(nil)
	if got.containerBaseDN != "" || got.originalBaseDN != "" || len(got.potentialGroupDNs) != 0 {
		t.Errorf("parseBaseDN(nil) = %+v, want empty struct", got)
	}

	emptyDN := &ldap.DN{}
	got = parseBaseDN(emptyDN)
	if got.containerBaseDN != "" || got.originalBaseDN != "" || len(got.potentialGroupDNs) != 0 {
		t.Errorf("parseBaseDN(empty) = %+v, want empty struct", got)
	}
}

func TestBuildMemberOfFilter(t *testing.T) {
	tests := []struct {
		name     string
		groupDNs []string
		want     string
	}{
		{
			name:     "empty",
			groupDNs: nil,
			want:     "",
		},
		{
			name:     "single group",
			groupDNs: []string{"cn=Admin Users,ou=Groups,dc=example,dc=com"},
			want:     "(memberOf:1.2.840.113556.1.4.1941:=cn=Admin Users,ou=Groups,dc=example,dc=com)",
		},
		{
			name:     "multiple groups",
			groupDNs: []string{"cn=Admins,dc=example,dc=com", "cn=Users,dc=example,dc=com"},
			want:     "(|(memberOf:1.2.840.113556.1.4.1941:=cn=Admins,dc=example,dc=com)(memberOf:1.2.840.113556.1.4.1941:=cn=Users,dc=example,dc=com))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMemberOfFilter(tt.groupDNs)
			if got != tt.want {
				t.Errorf("buildMemberOfFilter() = %q, want %q", got, tt.want)
			}
		})
	}
}

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
		users, err := GetDetails("(&(objectCategory=person)(objectClass=user))", url, user, pass, base, time.Time{}, nil, nil, 0, nil, nil)
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
		users, err := GetDetails("(&(objectCategory=person)(objectClass=user))", url, user, pass, base, since, nil, nil, 0, nil, nil)
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
