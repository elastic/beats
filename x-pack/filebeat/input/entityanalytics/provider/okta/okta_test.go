// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta/internal/okta"
	"github.com/elastic/elastic-agent-libs/logp"
)

var trace = flag.Bool("request_trace", false, "enable request tracing during tests")

func TestOktaDoFetch(t *testing.T) {
	logp.TestingSetup()

	tests := []struct {
		dataset     string
		enrichWith  []string
		wantUsers   bool
		wantDevices bool
	}{
		{dataset: "", enrichWith: []string{"groups"}, wantUsers: true, wantDevices: true},
		{dataset: "all", enrichWith: []string{"groups"}, wantUsers: true, wantDevices: true},
		{dataset: "users", enrichWith: []string{"groups", "roles", "factors"}, wantUsers: true, wantDevices: false},
		{dataset: "devices", enrichWith: []string{"groups"}, wantUsers: false, wantDevices: true},
	}

	for _, test := range tests {
		t.Run(test.dataset, func(t *testing.T) {
			suffix := test.dataset
			if suffix != "" {
				suffix = "_" + suffix
			}
			dbFilename := fmt.Sprintf("TestOktaDoFetch%s.db", suffix)
			store := testSetupStore(t, dbFilename)
			t.Cleanup(func() {
				testCleanupStore(store, dbFilename)
			})

			const (
				window  = time.Minute
				key     = "token"
				users   = `[{"id":"USERID","status":"STATUS","created":"2023-05-14T13:37:20.000Z","activated":null,"statusChanged":"2023-05-15T01:50:30.000Z","lastLogin":"2023-05-15T01:59:20.000Z","lastUpdated":"2023-05-15T01:50:32.000Z","passwordChanged":"2023-05-15T01:50:32.000Z","type":{"id":"typeid"},"profile":{"firstName":"name","lastName":"surname","mobilePhone":null,"secondEmail":null,"login":"name.surname@example.com","email":"name.surname@example.com"},"credentials":{"password":{"value":"secret"},"emails":[{"value":"name.surname@example.com","status":"VERIFIED","type":"PRIMARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://localhost/api/v1/users/USERID"}}}]`
				roles   = `[{"id":"IFIFAX2BIRGUSTQ","label":"Application administrator","type":"APP_ADMIN","status":"ACTIVE","created":"2019-02-06T16:17:40.000Z","lastUpdated":"2019-02-06T16:17:40.000Z","assignmentType":"USER"},{"id":"JBCUYUC7IRCVGS27IFCE2SKO","label":"Help Desk administrator","type":"HELP_DESK_ADMIN","status":"ACTIVE","created":"2019-02-06T16:17:40.000Z","lastUpdated":"2019-02-06T16:17:40.000Z","assignmentType":"USER"},{"id":"ra125eqBFpETrMwu80g4","label":"Organization administrator","type":"ORG_ADMIN","status":"ACTIVE","created":"2019-02-06T16:17:40.000Z","lastUpdated":"2019-02-06T16:17:40.000Z","assignmentType":"USER"},{"id":"gra25fapn1prGTBKV0g4","label":"API Access Management administrator","type":"API_ACCESS_MANAGEMENT_ADMIN","status":"ACTIVE","created\"":"2019-02-06T16:20:57.000Z","lastUpdated\"":"2019-02-06T16:20:57.000Z","assignmentType\"":"GROUP"}]`
				groups  = `[{"id":"USERID","profile":{"description":"All users in your organization","name":"Everyone"}}]`
				factors = `[{"id":"ufs2bysphxKODSZKWVCT","factorType":"question","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2014-04-15T18:10:06.000Z","lastUpdated":"2014-04-15T18:10:06.000Z","profile":{"question":"favorite_art_piece","questionText":"What is your favorite piece of art?"}},{"id":"ostf2gsyictRQDSGTDZE","factorType":"token:software:totp","provider":"OKTA","status":"PENDING_ACTIVATION","created":"2014-06-27T20:27:33.000Z","lastUpdated":"2014-06-27T20:27:33.000Z","profile":{"credentialId":"dade.murphy@example.com"}},{"id":"sms2gt8gzgEBPUWBIFHN","factorType":"sms","provider":"OKTA","status":"ACTIVE","created":"2014-06-27T20:27:26.000Z","lastUpdated":"2014-06-27T20:27:26.000Z","profile":{"phoneNumber":"+1-555-415-1337"}}]`
				devices = `[{"id":"DEVICEID","status":"STATUS","created":"2019-10-02T18:03:07.000Z","lastUpdated":"2019-10-02T18:03:07.000Z","profile":{"displayName":"Example Device name 1","platform":"WINDOWS","serialNumber":"XXDDRFCFRGF3M8MD6D","sid":"S-1-11-111","registered":true,"secureHardwarePresent":false,"diskEncryptionType":"ALL_INTERNAL_VOLUMES"},"resourceType":"UDDevice","resourceDisplayName":{"value":"Example Device name 1","sensitive":false},"resourceAlternateId":null,"resourceId":"DEVICEID","_links":{"activate":{"href":"https://localhost/api/v1/devices/DEVICEID/lifecycle/activate","hints":{"allow":["POST"]}},"self":{"href":"https://localhost/api/v1/devices/DEVICEID","hints":{"allow":["GET","PATCH","PUT"]}},"users":{"href":"https://localhost/api/v1/devices/DEVICEID/users","hints":{"allow":["GET"]}}}}]`
			)

			data := map[string]string{
				"users":   users,
				"roles":   roles,
				"groups":  groups,
				"devices": devices,
				"factors": factors,
			}

			var wantUsers []User
			if test.wantUsers {
				err := json.Unmarshal([]byte(users), &wantUsers)
				if err != nil {
					t.Fatalf("failed to unmarshal user data: %v", err)
				}
				var wantGroups []okta.Group
				err = json.Unmarshal([]byte(groups), &wantGroups)
				if err != nil {
					t.Fatalf("failed to unmarshal user data: %v", err)
				}
				for i, u := range wantUsers {
					wantUsers[i].Groups = append(u.Groups, wantGroups...)
				}
			}
			var wantDevices []Device
			if test.wantDevices {
				err := json.Unmarshal([]byte(devices), &wantDevices)
				if err != nil {
					t.Fatalf("failed to unmarshal device data: %v", err)
				}
			}
			var wantFactors []okta.Factor
			if slices.Contains(test.enrichWith, "factors") {
				err := json.Unmarshal([]byte(factors), &wantFactors)
				if err != nil {
					t.Fatalf("failed to unmarshal factor data: %v", err)
				}
			}
			var wantRoles []okta.Role
			if slices.Contains(test.enrichWith, "roles") {
				err := json.Unmarshal([]byte(roles), &wantRoles)
				if err != nil {
					t.Fatalf("failed to unmarshal role data: %v", err)
				}
			}

			wantStates := make(map[string]State)

			// Set the number of repeats.
			const repeats = 3
			var n int
			setHeaders := func(w http.ResponseWriter) {
				// Leave 49 remaining, reset in one minute.
				w.Header().Add("x-rate-limit-limit", "50")
				w.Header().Add("x-rate-limit-remaining", "49")
				w.Header().Add("x-rate-limit-reset", fmt.Sprint(time.Now().Add(time.Minute).Unix()))
			}
			mux := http.NewServeMux()
			mux.Handle("/api/v1/users/{userid}/{metadata}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setHeaders(w)
				attr := r.PathValue("metadata")
				if attr != "groups" {
					fmt.Fprintln(w, data[attr])
					return
				}
				// Give the groups if this is a get user groups request.
				userid := r.PathValue("userid")
				fmt.Fprintln(w, strings.ReplaceAll(data[attr], "USERID", userid))
			}))
			mux.Handle("/api/v1/devices/{deviceid}/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setHeaders(w)
				fmt.Fprintln(w, data["users"])
			}))
			mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setHeaders(w)

				base := path.Base(r.URL.Path)

				// Set next link if we can still repeat.
				n++
				if n < repeats {
					w.Header().Add("link", fmt.Sprintf(`<https://localhost/api/v1/%s?limit=200&after=opaquevalue>; rel="next"`, base))
				}

				prefix := strings.TrimRight(base, "s") // endpoints are plural.
				id := fmt.Sprintf("%sid%d", prefix, n)

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
				wantStates[id] = state

				replacer := strings.NewReplacer(
					strings.ToUpper(prefix+"id"), id,
					"STATUS", status,
				)
				fmt.Fprintln(w, replacer.Replace(data[base]))
			}))
			ts := httptest.NewTLSServer(mux)
			defer ts.Close()

			u, err := url.Parse(ts.URL)
			if err != nil {
				t.Errorf("failed to parse server URL: %v", err)
			}
			rateLimiter := okta.NewRateLimiter(window, nil)
			a := oktaInput{
				cfg: conf{
					OktaDomain: u.Host,
					OktaToken:  key,
					Dataset:    test.dataset,
					EnrichWith: test.enrichWith,
				},
				client: ts.Client(),
				lim:    rateLimiter,
				logger: logp.L(),
			}
			if *trace {
				name := test.dataset
				if name == "" {
					name = "default"
				}
				// Use legacy behaviour; nil enabled setting.
				a.cfg.Tracer = &tracerConfig{Logger: lumberjack.Logger{
					Filename: fmt.Sprintf("test_trace_%s.ndjson", name),
				}}
			}
			a.client = requestTrace(context.Background(), a.client, a.cfg, a.logger)

			ss, err := newStateStore(store)
			if err != nil {
				t.Fatalf("unexpected error making state store: %v", err)
			}
			defer ss.close(false)

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			t.Run("users", func(t *testing.T) {
				n = 0
				var got []*User
				published := make(map[string]struct{})

				err := a.doFetchUsers(ctx, ss, false, func(u *User) {
					got = append(got, u)
					published[u.ID] = struct{}{}
				})
				if err != nil {
					t.Fatalf("unexpected error from doFetch: %v", err)
				}

				if len(got) != wantCount(repeats, test.wantUsers) {
					t.Errorf("unexpected number of results: got:%d want:%d", len(got), wantCount(repeats, test.wantUsers))
				}
				if len(published) != len(got) {
					t.Errorf("unexpected number of distinct users published: got:%d want:%d", len(published), len(got))
				}
				for i, g := range got {
					wantID := fmt.Sprintf("userid%d", i+1)
					if g.ID != wantID {
						t.Errorf("unexpected user ID for user %d: got:%s want:%s", i, g.ID, wantID)
					}
					if len(g.Factors) != len(wantFactors) {
						t.Errorf("number of factors for user %d: got:%d want:%d", i, len(g.Factors), len(wantFactors))
					}
					if len(g.Roles) != len(wantRoles) {
						t.Errorf("number of roles for user %d: got:%d want:%d", i, len(g.Roles), len(wantRoles))
					}
					for j, gg := range g.Groups {
						if gg.ID != wantID {
							t.Errorf("unexpected used ID for user group %d in %d: got:%s want:%s", j, i, gg.ID, wantID)
						}
					}
					if g.State != wantStates[g.ID] {
						t.Errorf("unexpected user state for user %s: got:%s want:%s", g.ID, g.State, wantStates[g.ID])
					}
				}
			})

			t.Run("devices", func(t *testing.T) {
				n = 0
				var got []*Device
				published := make(map[string]struct{})

				err := a.doFetchDevices(ctx, ss, false, func(d *Device) {
					got = append(got, d)
					published[d.ID] = struct{}{}
				})
				if err != nil {
					t.Fatalf("unexpected error from doFetch: %v", err)
				}

				if len(got) != wantCount(repeats, test.wantDevices) {
					t.Errorf("unexpected number of results: got:%d want:%d", len(got), wantCount(repeats, test.wantDevices))
				}
				if len(published) != len(got) {
					t.Errorf("unexpected number of distinct devices published: got:%d want:%d", len(published), len(got))
				}
				for i, g := range got {
					if wantID := fmt.Sprintf("deviceid%d", i+1); g.ID != wantID {
						t.Errorf("unexpected device ID for device %d: got:%s want:%s", i, g.ID, wantID)
					}
					if g.State != wantStates[g.ID] {
						t.Errorf("unexpected device state for device %s: got:%s want:%s", g.ID, g.State, wantStates[g.ID])
					}
					if g.Users == nil {
						t.Errorf("expected users for device %s", g.ID)
					}
				}

				if t.Failed() {
					b, _ := json.MarshalIndent(got, "", "\t")
					t.Logf("document:\n%s", b)
				}
			})
		})
	}
}

func wantCount(n int, want bool) int {
	if !want {
		return 0
	}
	return n
}
