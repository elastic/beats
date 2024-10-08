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
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/okta/internal/okta"
	"github.com/elastic/elastic-agent-libs/logp"
)

var trace = flag.Bool("request_trace", false, "enable request tracing during tests")

func TestOktaDoFetch(t *testing.T) {
	logp.TestingSetup()

	tests := []struct {
		dataset     string
		wantUsers   bool
		wantDevices bool
	}{
		{dataset: "", wantUsers: true, wantDevices: true},
		{dataset: "all", wantUsers: true, wantDevices: true},
		{dataset: "users", wantUsers: true, wantDevices: false},
		{dataset: "devices", wantUsers: false, wantDevices: true},
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
				groups  = `[{"id":"USERID","profile":{"description":"All users in your organization","name":"Everyone"}}]`
				devices = `[{"id":"DEVICEID","status":"STATUS","created":"2019-10-02T18:03:07.000Z","lastUpdated":"2019-10-02T18:03:07.000Z","profile":{"displayName":"Example Device name 1","platform":"WINDOWS","serialNumber":"XXDDRFCFRGF3M8MD6D","sid":"S-1-11-111","registered":true,"secureHardwarePresent":false,"diskEncryptionType":"ALL_INTERNAL_VOLUMES"},"resourceType":"UDDevice","resourceDisplayName":{"value":"Example Device name 1","sensitive":false},"resourceAlternateId":null,"resourceId":"DEVICEID","_links":{"activate":{"href":"https://localhost/api/v1/devices/DEVICEID/lifecycle/activate","hints":{"allow":["POST"]}},"self":{"href":"https://localhost/api/v1/devices/DEVICEID","hints":{"allow":["GET","PATCH","PUT"]}},"users":{"href":"https://localhost/api/v1/devices/DEVICEID/users","hints":{"allow":["GET"]}}}}]`
			)

			data := map[string]string{
				"users":   users,
				"groups":  groups,
				"devices": devices,
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

			wantStates := make(map[string]State)

			// Set the number of repeats.
			const repeats = 3
			var n int
			ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Leave 49 remaining, reset in one minute.
				w.Header().Add("x-rate-limit-limit", "50")
				w.Header().Add("x-rate-limit-remaining", "49")
				w.Header().Add("x-rate-limit-reset", fmt.Sprint(time.Now().Add(time.Minute).Unix()))

				if strings.HasPrefix(r.URL.Path, "/api/v1/users") && strings.HasSuffix(r.URL.Path, "groups") {
					// Give the groups if this is a get user groups request.
					userid := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/users/"), "/groups")
					fmt.Fprintln(w, strings.ReplaceAll(data["groups"], "USERID", userid))
					return
				}
				if strings.HasPrefix(r.URL.Path, "/api/v1/device") && strings.HasSuffix(r.URL.Path, "users") {
					// Give one user if this is a get device users request.
					fmt.Fprintln(w, data["users"])
					return
				}

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
			defer ts.Close()

			u, err := url.Parse(ts.URL)
			if err != nil {
				t.Errorf("failed to parse server URL: %v", err)
			}
			a := oktaInput{
				cfg: conf{
					OktaDomain: u.Host,
					OktaToken:  key,
					Dataset:    test.dataset,
				},
				client: ts.Client(),
				lim:    rate.NewLimiter(1, 1),
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

				got, err := a.doFetchUsers(ctx, ss, false)
				if err != nil {
					t.Fatalf("unexpected error from doFetch: %v", err)
				}

				if len(got) != wantCount(repeats, test.wantUsers) {
					t.Errorf("unexpected number of results: got:%d want:%d", len(got), wantCount(repeats, test.wantUsers))
				}
				for i, g := range got {
					wantID := fmt.Sprintf("userid%d", i+1)
					if g.ID != wantID {
						t.Errorf("unexpected user ID for user %d: got:%s want:%s", i, g.ID, wantID)
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

				got, err := a.doFetchDevices(ctx, ss, false)
				if err != nil {
					t.Fatalf("unexpected error from doFetch: %v", err)
				}

				if len(got) != wantCount(repeats, test.wantDevices) {
					t.Errorf("unexpected number of results: got:%d want:%d", len(got), wantCount(repeats, test.wantDevices))
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
