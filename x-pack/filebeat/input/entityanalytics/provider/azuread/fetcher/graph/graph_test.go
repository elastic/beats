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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

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
