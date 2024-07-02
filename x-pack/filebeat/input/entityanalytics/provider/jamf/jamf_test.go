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
	"testing"
	"time"

	_ "embed"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/jamf/internal/jamf"
	"github.com/elastic/elastic-agent-libs/logp"
)

var trace = flag.Bool("request_trace", false, "enable request tracing during tests")

//go:embed internal/jamf/testdata/computers.json
var computers []byte

func TestJamfDoFetch(t *testing.T) {
	dbFilename := t.Name() + ".db"
	store := testSetupStore(t, dbFilename)
	t.Cleanup(func() {
		testCleanupStore(store, dbFilename)
	})

	var (
		wantComputers []*Computer
		rawComputers  jamf.Computers
	)
	err := json.Unmarshal(computers, &rawComputers)
	if err != nil {
		t.Fatalf("failed to unmarshal device data: %v", err)
	}
	for _, c := range rawComputers.Results {
		wantComputers = append(wantComputers, &Computer{
			Computer: c,
			State:    Discovered,
		})
	}

	// Set the number of repeats.
	tenant, username, password, client, cleanup, err := testContext()
	if err != nil {
		t.Fatalf("unexpected error getting env context: %v", err)
	}
	defer cleanup()

	a := jamfInput{
		cfg: conf{
			JamfTenant:   tenant,
			JamfUsername: username,
			JamfPassword: password,
		},
		client: client,
		logger: logp.L(),
	}
	if *trace {
		// Use legacy behaviour; nil enabled setting.
		a.cfg.Tracer = &tracerConfig{Logger: lumberjack.Logger{
			Filename: "test_trace.ndjson",
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

	t.Run("devices", func(t *testing.T) {
		got, err := a.doFetchComputers(ctx, ss, false)
		if err != nil {
			t.Fatalf("unexpected error from doFetch: %v", err)
		}

		if wantComputers != nil && !cmp.Equal(wantComputers, got) {
			t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(wantComputers, got))
		}
	})
}

func testContext() (tenant string, username string, password string, client *http.Client, cleanup func(), err error) {
	username = "testuser"
	password = "testuser_password"

	var tok jamf.Token
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
	mux.Handle("/api/preview/computers", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+tok.Token || !tok.IsValidFor(0) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("content-type", "application/json;charset=UTF-8")
			w.Write([]byte("{\n  \"httpStatus\" : 401,\n  \"errors\" : [ {\n    \"code\" : \"INVALID_TOKEN\",\n    \"description\" : \"Unauthorized\",\n    \"id\" : \"0\",\n    \"field\" : null\n  } ]\n}"))
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Header().Set("content-type", "application/json;charset=UTF-8")
			w.Write([]byte("{\n  \"httpStatus\" : 405,\n  \"errors\" : [ ]\n}"))
			return
		}
		w.Write(computers)
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
}
