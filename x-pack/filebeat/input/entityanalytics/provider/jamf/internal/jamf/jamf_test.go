// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jamf

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	_ "embed"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

var logResponses = flag.Bool("log_response", false, "use to log users/devices returned from the API")

var (
	//go:embed testdata/computers.json
	computers []byte

	//go:embed testdata/users.json
	users []byte
)

var jamfTests = []struct {
	name          string
	context       func() (tenant, username, password string, client *http.Client, cleanup func(), err error)
	wantComputers *Computers
	wantUsers     []User
}{
	{
		name: "jamf",
		context: func() (tenant string, username string, password string, client *http.Client, cleanup func(), err error) {
			tenant, ok := os.LookupEnv("JAMF_TENANT")
			if !ok {
				return "", "", "", nil, nil, skipError("jamf test requires ${JAMF_TENANT} to be set")
			}
			username, ok = os.LookupEnv("JAMF_USERNAME")
			if !ok {
				return "", "", "", nil, nil, skipError("jamf test requires ${JAMF_USERNAME} to be set")
			}
			password, ok = os.LookupEnv("JAMF_PASSWORD")
			if !ok {
				return "", "", "", nil, nil, skipError("jamf test requires ${JAMF_PASSWORD} to be set")
			}
			return tenant, username, password, http.DefaultClient, func() {}, nil
		},
	},
	{
		name: "local",
		context: func() (tenant string, username string, password string, client *http.Client, cleanup func(), err error) {
			username = "testuser"
			password = "testuser_password"

			var tok Token
			mux := http.NewServeMux()
			mux.Handle("/api/v1/auth/token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != username || pass != password {
					w.WriteHeader(http.StatusUnauthorized)
					w.Header().Set("content-type", "application/json;charset=UTF-8")
					w.Write([]byte("{\n  \"httpStatus\" : 401,\n  \"errors\" : [ ]\n}"))
					return
				}
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					w.Header().Set("content-type", "application/json;charset=UTF-8")
					w.Write([]byte("{\n  \"httpStatus\" : 405,\n  \"errors\" : [ ]\n}"))
					return
				}
				tok.Token = uuid.New().String()
				tok.Expires = time.Now().In(time.UTC).Add(time.Hour)
				fmt.Fprintf(w, "{\n  \"token\" : \"%s\",\n  \"expires\" : \"%s\"\n}", tok.Token, tok.Expires.Format(time.RFC3339))
			}))
			isValidRequest := func(w http.ResponseWriter, r *http.Request) bool {
				if r.Header.Get("Authorization") != "Bearer "+tok.Token || !tok.IsValidFor(0) {
					w.WriteHeader(http.StatusUnauthorized)
					w.Header().Set("content-type", "application/json;charset=UTF-8")
					w.Write([]byte("{\n  \"httpStatus\" : 401,\n  \"errors\" : [ {\n    \"code\" : \"INVALID_TOKEN\",\n    \"description\" : \"Unauthorized\",\n    \"id\" : \"0\",\n    \"field\" : null\n  } ]\n}"))
					return false
				}
				if r.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					w.Header().Set("content-type", "application/json;charset=UTF-8")
					w.Write([]byte("{\n  \"httpStatus\" : 405,\n  \"errors\" : [ ]\n}"))
					return false
				}
				return true
			}
			mux.Handle("/api/preview/computers", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if isValidRequest(w, r) {
					w.Write(computers)
				}
			}))
			mux.Handle("/JSSResource/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if isValidRequest(w, r) {
					w.Write(users)
				}
			}))

			srv := httptest.NewTLSServer(mux)
			u, err := url.Parse(srv.URL)
			if err != nil {
				srv.Close()
				return "", "", "", nil, func() {}, err
			}
			tenant = u.Host

			cli := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}

			return tenant, username, password, cli, srv.Close, nil
		},
		wantComputers: mustParseJSON[*Computers]("testdata/computers.json", computers),
		wantUsers:     mustParseJSON[*Users]("testdata/users.json", users).Users,
	},
}

func mustParseJSON[T any](name string, data []byte) T {
	var v T
	err := json.Unmarshal(data, &v)
	if err != nil {
		panic("invalid test data: " + name)
	}
	return v
}

func TestJamf(t *testing.T) {
	ctx := context.Background()

	for _, test := range jamfTests {
		t.Run(test.name, func(t *testing.T) {
			tenant, username, password, client, cleanup, err := test.context()
			switch err := err.(type) {
			case nil:
			case skipError:
				t.Skip(err)
			default:
				t.Fatalf("unexpected error getting env context: %v", err)
			}
			defer cleanup()
			tok, err := GetToken(ctx, client, tenant, username, password)
			if err != nil {
				t.Fatalf("unexpected error getting bearer token: %v", err)
			}

			t.Run("users", func(t *testing.T) {
				query := make(url.Values)
				query.Set("page-size", "10")
				got, err := GetUsers(ctx, client, tenant, tok, query)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if test.wantUsers != nil && !cmp.Equal(test.wantUsers, got) {
					t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(test.wantUsers, got))
				}
				if *logResponses {
					b, err := json.Marshal(got)
					if err != nil {
						t.Errorf("failed to marshal devices for logging: %v", err)
					}
					t.Logf("users: %s", b)
				}
			})

			t.Run("computers", func(t *testing.T) {
				query := make(url.Values)
				query.Set("page-size", "10")
				got, err := GetComputers(ctx, client, tenant, tok, query)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if test.wantComputers != nil && !cmp.Equal(*test.wantComputers, got) {
					t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(*test.wantComputers, got))
				}
				if *logResponses {
					b, err := json.Marshal(got)
					if err != nil {
						t.Errorf("failed to marshal devices for logging: %v", err)
					}
					t.Logf("devices: %s", b)
				}
			})
		})
	}
}

type skipError string

func (e skipError) Error() string {
	return string(e)
}
