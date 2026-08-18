// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package testazuread provides httptest mocks for Azure AD entity-analytics tests.
package testazuread

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/beats/v9/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher"
	mockfetcher "github.com/elastic/beats/v9/x-pack/filebeat/input/entityanalytics/provider/azuread/fetcher/mock"
)

// UserID is the ID of the first user in mockfetcher.UserResponse.
const UserID = "5ebc6a0f-05b7-4f42-9c8a-682bbc75d0fc"

// StartGraphServer starts a plain HTTP mock of Entra ID token + Graph
// endpoints backed by azuread/fetcher/mock data (same fixtures as the azure-ad
// provider equivalence and e2e tests).
func StartGraphServer(t *testing.T) *httptest.Server {
	t.Helper()

	var srvURL string
	mux := http.NewServeMux()

	deltaLink := func(path string) string {
		return srvURL + path
	}

	mux.HandleFunc("POST /test-tenant/oauth2/v2.0/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	mux.HandleFunc("GET /v1.0/users/delta", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"@odata.deltaLink": deltaLink("/v1.0/users/delta?$deltatoken=full"),
			"value":            usersAsGraphJSON(),
		})
	})

	mux.HandleFunc("GET /v1.0/devices/delta", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"@odata.deltaLink": deltaLink("/v1.0/devices/delta?$deltatoken=full"),
			"value":            devicesAsGraphJSON(),
		})
	})

	groups := mockfetcher.GroupResponse
	mux.HandleFunc("GET /v1.0/groups", func(w http.ResponseWriter, _ *http.Request) {
		var groupList []map[string]any
		for _, g := range groups {
			groupList = append(groupList, map[string]any{
				"id":          g.ID.String(),
				"displayName": g.Name,
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": groupList})
	})

	groupMembers := groupMembersMap()
	mux.HandleFunc("GET /v1.0/groups/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 || parts[4] != "members" {
			http.NotFound(w, r)
			return
		}
		groupID := parts[3]
		members := groupMembers[groupID]
		_ = json.NewEncoder(w).Encode(map[string]any{"value": members})
	})

	mux.HandleFunc("GET /v1.0/devices/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.NotFound(w, r)
			return
		}
		relation := parts[4]
		if relation != "registeredOwners" && relation != "registeredUsers" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": []any{}})
	})

	mux.HandleFunc("GET /v1.0/reports/authenticationMethods/userRegistrationDetails", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": mfaAsGraphJSON()})
	})

	mux.HandleFunc("GET /v1.0/users", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("$select") != "id,signInActivity" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": signInActivityAsGraphJSON()})
	})

	srv := httptest.NewServer(mux)
	srvURL = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func usersAsGraphJSON() []map[string]any {
	var result []map[string]any
	for _, u := range mockfetcher.UserResponse {
		entry := make(map[string]any, len(u.Fields)+1)
		entry["id"] = u.ID.String()
		maps.Copy(entry, u.Fields)
		result = append(result, entry)
	}
	return result
}

func devicesAsGraphJSON() []map[string]any {
	var result []map[string]any
	for _, d := range mockfetcher.DeviceResponse {
		entry := make(map[string]any, len(d.Fields)+1)
		entry["id"] = d.ID.String()
		maps.Copy(entry, d.Fields)
		result = append(result, entry)
	}
	return result
}

func groupMembersMap() map[string][]map[string]any {
	result := make(map[string][]map[string]any)
	for _, g := range mockfetcher.GroupResponse {
		var members []map[string]any
		for _, m := range g.Members {
			var odataType string
			switch m.Type {
			case fetcher.MemberUser:
				odataType = "#microsoft.graph.user"
			case fetcher.MemberDevice:
				odataType = "#microsoft.graph.device"
			case fetcher.MemberGroup:
				odataType = "#microsoft.graph.group"
			}
			members = append(members, map[string]any{
				"id":          m.ID.String(),
				"@odata.type": odataType,
			})
		}
		result[g.ID.String()] = members
	}
	return result
}

func mfaAsGraphJSON() []map[string]any {
	var result []map[string]any
	for userID, mfa := range mockfetcher.MFAResponse {
		result = append(result, map[string]any{
			"id":                    userID.String(),
			"isMfaCapable":          mfa.IsMFACapable,
			"isMfaRegistered":       mfa.IsMFARegistered,
			"isPasswordlessCapable": mfa.IsPasswordlessCapable,
			"isSsprCapable":         mfa.IsSsprCapable,
			"isSsprEnabled":         mfa.IsSsprEnabled,
			"isSsprRegistered":      mfa.IsSsprRegistered,
			"methodsRegistered":     mfa.MethodsRegistered,
			"userPreferredMethodForSecondaryAuthentication": mfa.UserPreferredMethodForSecondaryAuthentication,
			"userType": mfa.UserType,
		})
	}
	return result
}

func signInActivityAsGraphJSON() []map[string]any {
	var result []map[string]any
	for userID, sia := range mockfetcher.SignInActivityResponse {
		result = append(result, map[string]any{
			"id": userID.String(),
			"signInActivity": map[string]any{
				"lastSignInDateTime":                sia.LastSignInDateTime,
				"lastSignInRequestId":               sia.LastSignInRequestId,
				"lastNonInteractiveSignInDateTime":  sia.LastNonInteractiveSignInDateTime,
				"lastNonInteractiveSignInRequestId": sia.LastNonInteractiveSignInRequestId,
				"lastSuccessfulSignInDateTime":      sia.LastSuccessfulSignInDateTime,
				"lastSuccessfulSignInRequestId":     sia.LastSuccessfulSignInRequestId,
			},
		})
	}
	return result
}
