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
		{dataset: "users", enrichWith: []string{"perms"}, wantUsers: true, wantDevices: false},
		{dataset: "users", enrichWith: []string{"groups", "devices"}, wantUsers: true, wantDevices: false},
		{dataset: "users", enrichWith: []string{"supervises"}, wantUsers: true, wantDevices: false},
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
				window      = time.Minute
				key         = "token"
				users       = `[{"id":"USERID","status":"STATUS","created":"2023-05-14T13:37:20.000Z","activated":null,"statusChanged":"2023-05-15T01:50:30.000Z","lastLogin":"2023-05-15T01:59:20.000Z","lastUpdated":"2023-05-15T01:50:32.000Z","passwordChanged":"2023-05-15T01:50:32.000Z","type":{"id":"typeid"},"profile":{"firstName":"name","lastName":"surname","mobilePhone":null,"secondEmail":null,"login":"name.surname@example.com","email":"name.surname@example.com"},"credentials":{"password":{"value":"secret"},"emails":[{"value":"name.surname@example.com","status":"VERIFIED","type":"PRIMARY"}],"provider":{"type":"OKTA","name":"OKTA"}},"_links":{"self":{"href":"https://localhost/api/v1/users/USERID"}}}]`
				roles       = `[{"id":"IFIFAX2BIRGUSTQ","label":"Application administrator","type":"APP_ADMIN","status":"ACTIVE","created":"2019-02-06T16:17:40.000Z","lastUpdated":"2019-02-06T16:17:40.000Z","assignmentType":"USER"},{"id":"JBCUYUC7IRCVGS27IFCE2SKO","label":"Help Desk administrator","type":"HELP_DESK_ADMIN","status":"ACTIVE","created":"2019-02-06T16:17:40.000Z","lastUpdated":"2019-02-06T16:17:40.000Z","assignmentType":"USER"},{"id":"ra125eqBFpETrMwu80g4","label":"Organization administrator","type":"ORG_ADMIN","status":"ACTIVE","created":"2019-02-06T16:17:40.000Z","lastUpdated":"2019-02-06T16:17:40.000Z","assignmentType":"USER"},{"id":"gra25fapn1prGTBKV0g4","label":"API Access Management administrator","type":"API_ACCESS_MANAGEMENT_ADMIN","status":"ACTIVE","created":"2019-02-06T16:20:57.000Z","lastUpdated":"2019-02-06T16:20:57.000Z","assignmentType":"GROUP"},{"id":"ra0Yq6IJxGIr0ouum0g3","role":"cr0Yq6IJxGIr0ouum0g3","label":"Custom role","type":"CUSTOM","status":"ACTIVE","created":"2019-02-06T16:20:57.000Z","lastUpdated":"2019-02-06T16:20:57.000Z","assignmentType":"USER"}]`
				groups      = `[{"id":"USERID","profile":{"description":"All users in your organization","name":"Everyone"}}]`
				factors     = `[{"id":"ufs2bysphxKODSZKWVCT","factorType":"question","provider":"OKTA","vendorName":"OKTA","status":"ACTIVE","created":"2014-04-15T18:10:06.000Z","lastUpdated":"2014-04-15T18:10:06.000Z","profile":{"question":"favorite_art_piece","questionText":"What is your favorite piece of art?"}},{"id":"ostf2gsyictRQDSGTDZE","factorType":"token:software:totp","provider":"OKTA","status":"PENDING_ACTIVATION","created":"2014-06-27T20:27:33.000Z","lastUpdated":"2014-06-27T20:27:33.000Z","profile":{"credentialId":"dade.murphy@example.com"}},{"id":"sms2gt8gzgEBPUWBIFHN","factorType":"sms","provider":"OKTA","status":"ACTIVE","created":"2014-06-27T20:27:26.000Z","lastUpdated":"2014-06-27T20:27:26.000Z","profile":{"phoneNumber":"+1-555-415-1337"}}]`
				devices     = `[{"id":"DEVICEID","status":"STATUS","created":"2019-10-02T18:03:07.000Z","lastUpdated":"2019-10-02T18:03:07.000Z","profile":{"displayName":"Example Device name 1","platform":"WINDOWS","serialNumber":"XXDDRFCFRGF3M8MD6D","sid":"S-1-11-111","registered":true,"secureHardwarePresent":false,"diskEncryptionType":"ALL_INTERNAL_VOLUMES"},"resourceType":"UDDevice","resourceDisplayName":{"value":"Example Device name 1","sensitive":false},"resourceAlternateId":null,"resourceId":"DEVICEID","_links":{"activate":{"href":"https://localhost/api/v1/devices/DEVICEID/lifecycle/activate","hints":{"allow":["POST"]}},"self":{"href":"https://localhost/api/v1/devices/DEVICEID","hints":{"allow":["GET","PATCH","PUT"]}},"users":{"href":"https://localhost/api/v1/devices/DEVICEID/users","hints":{"allow":["GET"]}}}}]`
				permissions = `{"permissions":[{"label":"okta.users.read","created":"2021-02-06T16:17:40.000Z","lastUpdated":"2021-02-06T16:17:40.000Z"},{"label":"okta.apps.read","created":"2021-02-06T16:17:40.000Z","lastUpdated":"2021-02-06T16:17:40.000Z"}]}`
				// userDevices is sample data from https://developer.okta.com/docs/api/openapi/okta-management/management/tags/userresources/other/listuserdevices
				userDevices = `[{"id":"guo4a5uyerdpvAiJT0h7","status":"ACTIVE","created":"2022-05-14T13:37:20.000Z","lastUpdated":"2022-05-14T13:37:20.000Z","profile":{"displayName":"DESKTOP-XXXX","platform":"WINDOWS","manufacturer":"LENOVO","model":"20BH002DUS","osVersion":"10.0.19043","serialNumber":"1XXXX0X0X","registered":true,"secureHardwarePresent":false,"diskEncryptionType":"ALL_INTERNAL_VOLUMES"},"resourceType":"UDDevice","resourceDisplayName":{"value":"DESKTOP-XXXX","sensitive":false},"resourceAlternateId":null,"resourceId":"guo4a5uyerdpvAiJT0h7","_links":{"activate":{"href":"https://localhost/api/v1/devices/guo4a5uyerdpvAiJT0h7/lifecycle/activate","hints":{"allow":["POST"]}},"self":{"href":"https://localhost/api/v1/devices/guo4a5uyerdpvAiJT0h7","hints":{"allow":["GET","PATCH","PUT"]}}}}]`
			)

			data := map[string]string{
				"users":        users,
				"roles":        roles,
				"groups":       groups,
				"devices":      devices,
				"factors":      factors,
				"permissions":  permissions,
				"user_devices": userDevices,
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
			if slices.Contains(test.enrichWith, "roles") || slices.Contains(test.enrichWith, "perms") {
				err := json.Unmarshal([]byte(roles), &wantRoles)
				if err != nil {
					t.Fatalf("failed to unmarshal role data: %v", err)
				}
			}
			var wantPerms []okta.Permission
			if slices.Contains(test.enrichWith, "perms") {
				var result struct {
					Permissions []okta.Permission `json:"permissions"`
				}
				err := json.Unmarshal([]byte(permissions), &result)
				if err != nil {
					t.Fatalf("failed to unmarshal permissions data: %v", err)
				}
				wantPerms = result.Permissions
			}
			var wantUserDevices []okta.Device
			if slices.Contains(test.enrichWith, "devices") {
				err := json.Unmarshal([]byte(userDevices), &wantUserDevices)
				if err != nil {
					t.Fatalf("failed to unmarshal user device data: %v", err)
				}
			}
			// supervises is computed from profile.managerId in state — test users
			// have no managerId, so the expected list is always empty.
			var wantSupervises []okta.SupervisedUser

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
			permCalls := map[string]int{}

			mux := http.NewServeMux()
			mux.Handle("/api/v1/users/{userid}/{metadata}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setHeaders(w)
				attr := r.PathValue("metadata")
				switch attr {
				case "groups":
					// Replace USERID placeholder with the actual user ID.
					userid := r.PathValue("userid")
					fmt.Fprintln(w, strings.ReplaceAll(data[attr], "USERID", userid))
				case "devices":
					// User-enrolled devices are served from a separate data key.
					fmt.Fprintln(w, data["user_devices"])
				default:
					fmt.Fprintln(w, data[attr])
				}
			}))
			mux.Handle("/api/v1/iam/roles/{roleId}/permissions", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				setHeaders(w)
				roleID := r.PathValue("roleId")
				permCalls[roleID]++
				if roleID != "cr0Yq6IJxGIr0ouum0g3" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				fmt.Fprintln(w, data["permissions"])
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
					if len(g.Devices) != len(wantUserDevices) {
						t.Errorf("number of enrolled devices for user %d: got:%d want:%d", i, len(g.Devices), len(wantUserDevices))
					}
					if slices.Contains(test.enrichWith, "perms") {
						for j, role := range g.Roles {
							want := 0
							if role.Type == "CUSTOM" {
								want = len(wantPerms)
							}
							if len(role.Permissions) != want {
								t.Errorf("number of permissions for role %d (type %s) of user %d: got:%d want:%d", j, role.Type, i, len(role.Permissions), want)
							}
						}
					}
					if len(g.Supervises) != len(wantSupervises) {
						t.Errorf("number of supervised users for user %d: got:%d want:%d", i, len(g.Supervises), len(wantSupervises))
					}
					for j, su := range g.Supervises {
						w := wantSupervises[j]
						if su.ID != w.ID {
							t.Errorf("unexpected supervised user ID %d for user %d: got:%s want:%s", j, i, su.ID, w.ID)
						}
						if su.Email != w.Email {
							t.Errorf("unexpected supervised user email %d for user %d: got:%s want:%s", j, i, su.Email, w.Email)
						}
						if su.Username != w.Username {
							t.Errorf("unexpected supervised user username %d for user %d: got:%s want:%s", j, i, su.Username, w.Username)
						}
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
				if slices.Contains(test.enrichWith, "perms") {
					// The cache must deduplicate calls: one custom role definition across
					// repeats users means exactly one permissions API call.
					if permCalls["cr0Yq6IJxGIr0ouum0g3"] != 1 {
						t.Errorf("permissions endpoint call count for custom role: got:%d want:1", permCalls["cr0Yq6IJxGIr0ouum0g3"])
					}
					if len(permCalls) != 1 {
						t.Errorf("permissions endpoint called for unexpected role IDs: %v", permCalls)
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

// TestOktaDoFetchSupervisesEnrichment exercises the full doFetchUsers path when
// supervises enrichment is enabled. It verifies that supervises relationships are
// correctly derived from profile.managerId without any additional API calls.
func TestOktaDoFetchSupervisesEnrichment(t *testing.T) {
	logp.TestingSetup()

	dbFilename := "TestOktaDoFetchSupervisesEnrichment.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() { testCleanupStore(store, dbFilename) })

	const (
		window = time.Minute
		key    = "token"
		// manager has no managerId; sub1 and sub2 report to manager.
		manager = `{"id":"manager-id","status":"ACTIVE","created":"2023-05-14T13:37:20.000Z","activated":"2023-05-14T13:37:20.000Z","lastUpdated":"2023-05-15T01:50:32.000Z","type":{},"profile":{"email":"manager@example.com","login":"manager@example.com"}}`
		sub1    = `{"id":"sub1-id","status":"ACTIVE","created":"2023-05-14T13:37:20.000Z","activated":"2023-05-14T13:37:20.000Z","lastUpdated":"2023-05-15T01:50:32.000Z","type":{},"profile":{"email":"sub1@example.com","login":"sub1@example.com","managerId":"manager-id"}}`
		sub2    = `{"id":"sub2-id","status":"ACTIVE","created":"2023-05-14T13:37:20.000Z","activated":"2023-05-14T13:37:20.000Z","lastUpdated":"2023-05-15T01:50:32.000Z","type":{},"profile":{"email":"sub2@example.com","login":"sub2@example.com","managerId":"manager-id"}}`
	)

	allUsers := "[" + manager + "," + sub1 + "," + sub2 + "]"

	setHeaders := func(w http.ResponseWriter) {
		w.Header().Add("x-rate-limit-limit", "50")
		w.Header().Add("x-rate-limit-remaining", "49")
		w.Header().Add("x-rate-limit-reset", fmt.Sprint(time.Now().Add(time.Minute).Unix()))
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setHeaders(w)
		fmt.Fprintln(w, allUsers)
	}))
	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("unexpected error parsing server URL: %v", err)
	}

	a := oktaInput{
		cfg: conf{
			OktaDomain: u.Host,
			OktaToken:  key,
			Dataset:    "users",
			EnrichWith: []string{"supervises"},
		},
		client: ts.Client(),
		lim:    okta.NewRateLimiter(window, nil),
		logger: logp.L(),
	}

	ss, err := newStateStore(store)
	if err != nil {
		t.Fatalf("unexpected error making state store: %v", err)
	}
	defer ss.close(false)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var got []*User
	err = a.doFetchUsers(ctx, ss, true, func(u *User) {
		got = append(got, u)
	})
	if err != nil {
		t.Fatalf("unexpected error from doFetchUsers: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 users, got %d", len(got))
	}

	gotByID := make(map[string]*User, len(got))
	for _, u := range got {
		gotByID[u.ID] = u
	}

	// Manager must have both subordinates in Supervises.
	managerUser, ok := gotByID["manager-id"]
	if !ok {
		t.Fatal("manager user not found in published results")
	}
	if len(managerUser.Supervises) != 2 {
		t.Fatalf("expected 2 supervised users for manager, got %d: %v", len(managerUser.Supervises), managerUser.Supervises)
	}
	supervisesByID := make(map[string]okta.SupervisedUser, len(managerUser.Supervises))
	for _, su := range managerUser.Supervises {
		supervisesByID[su.ID] = su
	}
	for _, want := range []okta.SupervisedUser{
		{ID: "sub1-id", Email: "sub1@example.com", Username: "sub1@example.com"},
		{ID: "sub2-id", Email: "sub2@example.com", Username: "sub2@example.com"},
	} {
		got, ok := supervisesByID[want.ID]
		if !ok {
			t.Errorf("supervised user %s not found in manager's Supervises", want.ID)
			continue
		}
		if got != want {
			t.Errorf("supervised user %s: got %+v, want %+v", want.ID, got, want)
		}
	}

	// Subordinates must not supervise anyone.
	for _, id := range []string{"sub1-id", "sub2-id"} {
		u, ok := gotByID[id]
		if !ok {
			t.Errorf("user %s not found in published results", id)
			continue
		}
		if len(u.Supervises) != 0 {
			t.Errorf("expected no supervised users for %s, got %d", id, len(u.Supervises))
		}
	}
}

func TestAssignSupervises(t *testing.T) {
	logp.TestingSetup()

	dbFilename := "TestAssignSupervises.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() { testCleanupStore(store, dbFilename) })

	ss, err := newStateStore(store)
	if err != nil {
		t.Fatalf("unexpected error making state store: %v", err)
	}
	defer ss.close(false)

	// Populate state with a manager and two subordinates. The subordinates
	// carry profile.managerId pointing to the manager's ID, matching what
	// Okta returns in the standard user profile.
	ss.storeUser(okta.User{
		ID: "manager-id",
		Profile: map[string]any{
			"email": "manager@example.com",
			"login": "manager@example.com",
		},
	})
	ss.storeUser(okta.User{
		ID: "sub1-id",
		Profile: map[string]any{
			"email":     "sub1@example.com",
			"login":     "sub1@example.com",
			"managerId": "manager-id",
		},
	})
	ss.storeUser(okta.User{
		ID: "sub2-id",
		Profile: map[string]any{
			"email":     "sub2@example.com",
			"login":     "sub2@example.com",
			"managerId": "manager-id",
		},
	})

	a := oktaInput{logger: logp.L()}
	a.assignSupervises(ss)

	manager := ss.users["manager-id"]
	if len(manager.Supervises) != 2 {
		t.Fatalf("expected 2 supervised users for manager, got %d", len(manager.Supervises))
	}
	gotIDs := map[string]okta.SupervisedUser{}
	for _, su := range manager.Supervises {
		gotIDs[su.ID] = su
	}
	for _, want := range []okta.SupervisedUser{
		{ID: "sub1-id", Email: "sub1@example.com", Username: "sub1@example.com"},
		{ID: "sub2-id", Email: "sub2@example.com", Username: "sub2@example.com"},
	} {
		got, ok := gotIDs[want.ID]
		if !ok {
			t.Errorf("supervised user %s not found", want.ID)
			continue
		}
		if got != want {
			t.Errorf("supervised user %s: got %+v want %+v", want.ID, got, want)
		}
	}

	// Subordinates should not supervise anyone.
	for _, id := range []string{"sub1-id", "sub2-id"} {
		if len(ss.users[id].Supervises) != 0 {
			t.Errorf("expected no supervised users for %s, got %d", id, len(ss.users[id].Supervises))
		}
	}
}

// TestAssignSupervisesOrdering proves that assignSupervises always produces a
// Supervises slice sorted by subordinate ID, regardless of map iteration order.
// Without sorting, Go's non-deterministic map iteration would produce a
// different ordering on each call, causing supervisesEqual to treat identical
// membership as a change and trigger spurious republishes on every incremental sync.
func TestAssignSupervisesOrdering(t *testing.T) {
	logp.TestingSetup()

	dbFilename := "TestAssignSupervisesOrdering.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() { testCleanupStore(store, dbFilename) })

	ss, err := newStateStore(store)
	if err != nil {
		t.Fatalf("unexpected error making state store: %v", err)
	}
	defer ss.close(false)

	// Store a manager and three subordinates with IDs chosen so that insertion
	// order differs from lexicographic order, maximising the chance of catching
	// an unsorted result.
	ss.storeUser(okta.User{ID: "mgr", Profile: map[string]any{"email": "mgr@example.com", "login": "mgr@example.com"}})
	for _, sub := range []struct{ id, email string }{
		{"zzz-sub", "zzz@example.com"},
		{"aaa-sub", "aaa@example.com"},
		{"mmm-sub", "mmm@example.com"},
	} {
		ss.storeUser(okta.User{
			ID: sub.id,
			Profile: map[string]any{
				"email":     sub.email,
				"login":     sub.email,
				"managerId": "mgr",
			},
		})
	}

	a := oktaInput{logger: logp.L()}

	// Call assignSupervises multiple times. If the slice were unsorted, the
	// ordering would vary across calls due to map iteration randomness.
	want := []okta.SupervisedUser{
		{ID: "aaa-sub", Email: "aaa@example.com", Username: "aaa@example.com"},
		{ID: "mmm-sub", Email: "mmm@example.com", Username: "mmm@example.com"},
		{ID: "zzz-sub", Email: "zzz@example.com", Username: "zzz@example.com"},
	}
	for i := range 10 {
		a.assignSupervises(ss)
		got := ss.users["mgr"].Supervises
		if len(got) != len(want) {
			t.Fatalf("iteration %d: expected %d supervised users, got %d", i, len(want), len(got))
		}
		for j := range want {
			if got[j] != want[j] {
				t.Errorf("iteration %d: position %d: got %+v, want %+v", i, j, got[j], want[j])
			}
		}
	}
}

// TestStoreUserExistingPointer proves that storeUser returns the pointer that
// lives in state.users for an existing user. This is required so that
// assignSupervises (which iterates state.users) and the published *User pointer
// are the same object — otherwise Supervises set via state.users would never
// reach the published document or the persisted store.
func TestStoreUserExistingPointer(t *testing.T) {
	logp.TestingSetup()

	dbFilename := "TestStoreUserExistingPointer.db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() { testCleanupStore(store, dbFilename) })

	ss, err := newStateStore(store)
	if err != nil {
		t.Fatalf("unexpected error making state store: %v", err)
	}

	u := okta.User{ID: "user-1", Profile: map[string]any{"email": "a@example.com"}}

	// First store: user is new (Discovered).
	ptr1 := ss.storeUser(u)
	if ptr1 != ss.users["user-1"] {
		t.Fatal("first storeUser: returned pointer not the same as state.users entry")
	}

	// Second store: user already exists (Modified). The returned pointer must
	// still be the same object that lives in state.users so that any subsequent
	// mutation (e.g. setting Supervises) is visible in the map and will be
	// persisted on close.
	ptr2 := ss.storeUser(u)
	if ptr2 != ss.users["user-1"] {
		t.Fatal("second storeUser: returned pointer not the same as state.users entry for existing user")
	}

	// Mutating via the returned pointer must be visible through state.users.
	ptr2.Supervises = []okta.SupervisedUser{{ID: "sub-1", Email: "sub@example.com", Username: "sub@example.com"}}
	if len(ss.users["user-1"].Supervises) != 1 {
		t.Fatal("mutation via returned pointer not visible through state.users")
	}

	// Commit and reopen; Supervises must survive the round-trip.
	if err := ss.close(true); err != nil {
		t.Fatalf("close with commit failed: %v", err)
	}

	ss2, err := newStateStore(store)
	if err != nil {
		t.Fatalf("unexpected error reopening state store: %v", err)
	}
	defer ss2.close(false)

	reloaded, ok := ss2.users["user-1"]
	if !ok {
		t.Fatal("user-1 not found after reopen")
	}
	if len(reloaded.Supervises) != 1 || reloaded.Supervises[0].ID != "sub-1" {
		t.Errorf("Supervises not persisted: got %+v", reloaded.Supervises)
	}
}
