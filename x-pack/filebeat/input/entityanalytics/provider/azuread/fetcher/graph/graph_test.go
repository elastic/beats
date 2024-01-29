// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/authenticator/mock"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var usersResponse1 = apiUserResponse{
	Users: []userAPI{
		{
			"id":                "5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc",
			"userPrincipalName": "user.one@example.com",
			"mail":              "user.one@example.com",
			"displayName":       "User One",
			"givenName":         "User",
			"surname":           "One",
			"jobTitle":          "Software Engineer",
			"mobilePhone":       "123-555-1000",
			"businessPhones": []string{
				"123-555-0122",
			},
		},
	},
}

var usersResponse2 = apiUserResponse{
	Users: []userAPI{
		{
			"id":                "d897d560-3d17-4dae-81b3-c898fe82bf84",
			"userPrincipalName": "user.two@example.com",
			"mail":              "user.two@example.com",
			"displayName":       "User Two",
			"givenName":         "User",
			"surname":           "Two",
			"jobTitle":          "Accountant",
			"mobilePhone":       "205-555-2000",
			"businessPhones": []string{
				"205-555-5488",
				"205-555-7724",
			},
		},
	},
}

var devicesResponse1 = apiDeviceResponse{
	Devices: []deviceAPI{
		{
			"id":                     "6a59ea83-02bd-468f-a40b-f2c3d1821983",
			"accountEnabled":         true,
			"deviceId":               "eab73519-780d-4d43-be6d-a4a89af2a348",
			"displayName":            "DESKTOP-LK3PESR",
			"operatingSystem":        "Windows",
			"operatingSystemVersion": "10.0.19043.1237",
			"physicalIds":            []interface{}{},
			"extensionAttributes": map[string]interface{}{
				"extensionAttribute1": "BYOD-Device",
				"extensionAttribute2": nil,
				"extensionAttribute3": nil,
				"extensionAttribute4": nil,
			},
			"alternativeSecurityIds": []interface{}{
				map[string]interface{}{
					"type":             "2", // Rendered as string to avoid in-flight conversion to float.
					"identityProvider": nil,
					"key":              "WAA1ADAAOQA6AD...QBnAD0A",
				},
			},
		},
	},
}

var devicesResponse2 = apiDeviceResponse{
	Devices: []deviceAPI{
		{
			"id":                     "adbbe40a-0627-4328-89f1-88cac84dbc7f",
			"accountEnabled":         true,
			"deviceId":               "2fbbb8f9-ff67-4a21-b867-a344d18a4198",
			"displayName":            "DESKTOP-LETW452G",
			"operatingSystem":        "Windows",
			"operatingSystemVersion": "10.0.19043.1337",
			"physicalIds":            []interface{}{},
			"extensionAttributes": map[string]interface{}{
				"extensionAttribute1": "BYOD-Device",
				"extensionAttribute2": nil,
				"extensionAttribute3": nil,
				"extensionAttribute4": nil,
			},
			"alternativeSecurityIds": []interface{}{
				map[string]interface{}{
					"type":             "2", // Rendered as string to avoid in-flight conversion to float.
					"identityProvider": nil,
					"key":              "DGFSGHSGGTH345A...35DSFH0A",
				},
			},
		},
	},
}

var deviceOwnerResponses = map[string]apiUserResponse{
	"6a59ea83-02bd-468f-a40b-f2c3d1821983": {
		Users: []userAPI{{"id": "5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"}},
	},
	"adbbe40a-0627-4328-89f1-88cac84dbc7f": {
		Users: []userAPI{{"id": "5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"}},
	},
}

var deviceUserResponses = map[string]apiUserResponse{
	"6a59ea83-02bd-468f-a40b-f2c3d1821983": {
		Users: []userAPI{{"id": "d897d560-3d17-4dae-81b3-c898fe82bf84"}, {"id": "5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"}},
	},
	"adbbe40a-0627-4328-89f1-88cac84dbc7f": {
		Users: []userAPI{{"id": "5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"}},
	},
}

var groupsResponse1 = apiGroupResponse{
	Groups: []groupAPI{
		{
			ID:          uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80"),
			DisplayName: "group1",
			MembersDelta: []memberAPI{
				{
					ID:   uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
					Type: apiUserType,
				},
			},
		},
	},
}

var groupsResponse2 = apiGroupResponse{
	Groups: []groupAPI{
		{
			ID:          uuid.MustParse("d140978f-d641-4f01-802f-4ecc1acf8935"),
			DisplayName: "group2",
			MembersDelta: []memberAPI{
				{
					ID:   uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80"),
					Type: apiGroupType,
				},
				{
					ID:      uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
					Type:    apiGroupType,
					Removed: &removed{Reason: "changed"},
				},
			},
		},
	},
}

type testServer struct {
	srv  *httptest.Server
	addr string
}

func (s *testServer) setup(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/users/delta", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		var data []byte
		var err error

		skipToken := r.URL.Query().Get("$skiptoken")
		switch skipToken {
		case "":
			usersResponse1.NextLink = "http://" + s.addr + "/users/delta?$skiptoken=test"
			data, err = json.Marshal(&usersResponse1)
		case "test":
			usersResponse2.DeltaLink = "http://" + s.addr + "/users/delta?$deltatoken=test"
			data, err = json.Marshal(&usersResponse2)
		default:
			err = fmt.Errorf("unknown skipToken value: %q", skipToken)
		}
		require.NoError(t, err)

		_, err = w.Write(data)
		require.NoError(t, err)
	})

	mux.HandleFunc("/devices/delta", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		var data []byte
		var err error

		skipToken := r.URL.Query().Get("$skiptoken")
		switch skipToken {
		case "":
			devicesResponse1.NextLink = "http://" + s.addr + "/devices/delta?$skiptoken=test"
			data, err = json.Marshal(&devicesResponse1)
		case "test":
			devicesResponse2.DeltaLink = "http://" + s.addr + "/devices/delta?$deltatoken=test"
			data, err = json.Marshal(&devicesResponse2)
		default:
			err = fmt.Errorf("unknown skipToken value: %q", skipToken)
		}
		require.NoError(t, err)

		_, err = w.Write(data)
		require.NoError(t, err)
	})

	mux.HandleFunc("/devices/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		var data []byte
		var err error

		switch path.Base(r.URL.Path) {
		case "registeredOwners":
			data, err = json.Marshal(deviceOwnerResponses[path.Base(path.Dir(r.URL.Path))])
		case "registeredUsers":
			data, err = json.Marshal(deviceUserResponses[path.Base(path.Dir(r.URL.Path))])
		default:
			err = fmt.Errorf("unknown endpoint: %s", r.URL)
		}
		require.NoError(t, err)

		_, err = w.Write(data)
		require.NoError(t, err)
	})

	mux.HandleFunc("/groups/delta", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		var data []byte
		var err error

		skipToken := r.URL.Query().Get("$skiptoken")
		switch skipToken {
		case "":
			groupsResponse1.NextLink = "http://" + s.addr + "/groups/delta?$skiptoken=test"
			data, err = json.Marshal(&groupsResponse1)
		case "test":
			groupsResponse2.DeltaLink = "http://" + s.addr + "/groups/delta?$deltatoken=test"
			data, err = json.Marshal(&groupsResponse2)
		default:
			err = fmt.Errorf("unknown skipToken value: %q", skipToken)
		}
		require.NoError(t, err)

		_, err = w.Write(data)
		require.NoError(t, err)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		require.Fail(t, "Matched unknown route")
	})

	s.srv = httptest.NewServer(mux)
	s.addr = s.srv.Listener.Addr().String()
}

func TestGraph_Groups(t *testing.T) {
	var testSrv testServer
	testSrv.setup(t)
	defer testSrv.srv.Close()

	wantDeltaLink := "http://" + testSrv.addr + "/groups/delta?$deltatoken=test"
	wantGroups := []*fetcher.Group{
		{
			ID:   uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80"),
			Name: "group1",
			Members: []fetcher.Member{
				{
					ID:   uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
					Type: fetcher.MemberUser,
				},
			},
		},
		{
			ID:   uuid.MustParse("d140978f-d641-4f01-802f-4ecc1acf8935"),
			Name: "group2",
			Members: []fetcher.Member{
				{
					ID:   uuid.MustParse("331676df-b8fd-4492-82ed-02b927f8dd80"),
					Type: fetcher.MemberGroup,
				},
				{
					ID:      uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
					Type:    fetcher.MemberGroup,
					Deleted: true,
				},
			},
		},
	}

	rawConf := graphConf{
		APIEndpoint: "http://" + testSrv.addr,
	}
	c, err := config.NewConfigFrom(&rawConf)
	require.NoError(t, err)
	auth := mock.New(mock.DefaultTokenValue)

	f, err := New(c, logp.L(), auth)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gotGroups, gotDeltaLink, gotErr := f.Groups(ctx, "")

	require.NoError(t, gotErr)
	require.EqualValues(t, wantGroups, gotGroups)
	require.Equal(t, wantDeltaLink, gotDeltaLink)
}

func TestGraph_Users(t *testing.T) {
	var testSrv testServer
	testSrv.setup(t)
	defer testSrv.srv.Close()

	wantDeltaLink := "http://" + testSrv.addr + "/users/delta?$deltatoken=test"
	wantUsers := []*fetcher.User{
		{
			ID: uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
			Fields: map[string]interface{}{
				"userPrincipalName": "user.one@example.com",
				"mail":              "user.one@example.com",
				"displayName":       "User One",
				"givenName":         "User",
				"surname":           "One",
				"jobTitle":          "Software Engineer",
				"mobilePhone":       "123-555-1000",
				"businessPhones": []any{
					"123-555-0122",
				},
			},
		},
		{
			ID: uuid.MustParse("d897d560-3d17-4dae-81b3-c898fe82bf84"),
			Fields: map[string]interface{}{
				"userPrincipalName": "user.two@example.com",
				"mail":              "user.two@example.com",
				"displayName":       "User Two",
				"givenName":         "User",
				"surname":           "Two",
				"jobTitle":          "Accountant",
				"mobilePhone":       "205-555-2000",
				"businessPhones": []any{
					"205-555-5488",
					"205-555-7724",
				},
			},
		},
	}

	rawConf := graphConf{
		APIEndpoint: "http://" + testSrv.addr,
	}
	c, err := config.NewConfigFrom(&rawConf)
	require.NoError(t, err)
	auth := mock.New(mock.DefaultTokenValue)

	f, err := New(c, logp.L(), auth)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gotUsers, gotDeltaLink, gotErr := f.Users(ctx, "")

	require.NoError(t, gotErr)
	require.EqualValues(t, wantUsers, gotUsers)
	require.Equal(t, wantDeltaLink, gotDeltaLink)
}

func TestGraph_Devices(t *testing.T) {
	var testSrv testServer
	testSrv.setup(t)
	defer testSrv.srv.Close()

	wantDeltaLink := "http://" + testSrv.addr + "/devices/delta?$deltatoken=test"
	wantDevices := []*fetcher.Device{
		{
			ID: uuid.MustParse("6a59ea83-02bd-468f-a40b-f2c3d1821983"),
			Fields: map[string]interface{}{
				"accountEnabled":         true,
				"deviceId":               "eab73519-780d-4d43-be6d-a4a89af2a348",
				"displayName":            "DESKTOP-LK3PESR",
				"operatingSystem":        "Windows",
				"operatingSystemVersion": "10.0.19043.1237",
				"physicalIds":            []interface{}{},
				"extensionAttributes": map[string]interface{}{
					"extensionAttribute1": "BYOD-Device",
					"extensionAttribute2": nil,
					"extensionAttribute3": nil,
					"extensionAttribute4": nil,
				},
				"alternativeSecurityIds": []interface{}{
					map[string]interface{}{
						"type":             "2", // Rendered as string to avoid in-flight conversion to float.
						"identityProvider": nil,
						"key":              "WAA1ADAAOQA6AD...QBnAD0A",
					},
				},
			},
			RegisteredOwners: collections.NewUUIDSet(
				uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
			),
			RegisteredUsers: collections.NewUUIDSet(
				uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
				uuid.MustParse("d897d560-3d17-4dae-81b3-c898fe82bf84"),
			),
		},
		{
			ID: uuid.MustParse("adbbe40a-0627-4328-89f1-88cac84dbc7f"),
			Fields: map[string]interface{}{
				"accountEnabled":         true,
				"deviceId":               "2fbbb8f9-ff67-4a21-b867-a344d18a4198",
				"displayName":            "DESKTOP-LETW452G",
				"operatingSystem":        "Windows",
				"operatingSystemVersion": "10.0.19043.1337",
				"physicalIds":            []interface{}{},
				"extensionAttributes": map[string]interface{}{
					"extensionAttribute1": "BYOD-Device",
					"extensionAttribute2": nil,
					"extensionAttribute3": nil,
					"extensionAttribute4": nil,
				},
				"alternativeSecurityIds": []interface{}{
					map[string]interface{}{
						"type":             "2", // Rendered as string to avoid in-flight conversion to float.
						"identityProvider": nil,
						"key":              "DGFSGHSGGTH345A...35DSFH0A",
					},
				},
			},
			RegisteredOwners: collections.NewUUIDSet(
				uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
			),
			RegisteredUsers: collections.NewUUIDSet(
				uuid.MustParse("5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"),
			),
		},
	}

	for _, test := range []struct {
		name      string
		selection selection
	}{
		{name: "default_selection"},
		{
			name: "user_selection",
			selection: selection{
				UserQuery:   strings.Split(strings.TrimPrefix(defaultUsersQuery, "$select="), ","),
				GroupQuery:  strings.Split(strings.TrimPrefix(defaultGroupsQuery, "$select="), ","),
				DeviceQuery: strings.Split(strings.TrimPrefix(defaultDevicesQuery, "$select="), ","),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			rawConf := graphConf{
				APIEndpoint: "http://" + testSrv.addr,
				Select:      test.selection,
			}
			c, err := config.NewConfigFrom(&rawConf)
			require.NoError(t, err)
			auth := mock.New(mock.DefaultTokenValue)

			f, err := New(c, logp.L(), auth)
			require.NoError(t, err)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			gotDevices, gotDeltaLink, gotErr := f.Devices(ctx, "")

			require.NoError(t, gotErr)
			// Using go-cmp because testify is too weak for this comparison.
			// reflect.DeepEqual works, but won't show a reasonable diff.
			exporter := cmp.Exporter(func(t reflect.Type) bool {
				return t == reflect.TypeOf(collections.UUIDSet{})
			})
			if !cmp.Equal(wantDevices, gotDevices, exporter) {
				t.Errorf("unexpected result:\n--- got\n--- want\n%s", cmp.Diff(wantDevices, gotDevices, exporter))
			}
			require.Equal(t, wantDeltaLink, gotDeltaLink)
		})
	}
}
