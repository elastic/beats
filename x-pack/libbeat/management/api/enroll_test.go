// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestEnrollValid(t *testing.T) {
	beatUUID := uuid.NewV4()

	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		// Check correct path is used
		assert.Equal(t, "/api/beats/agent/"+beatUUID.String(), r.URL.Path)

		// Check enrollment token is correct
		assert.Equal(t, "thisismyenrollmenttoken", r.Header.Get("kbn-beats-enrollment-token"))

		request := struct {
			Hostname string `json:"host_name"`
			Type     string `json:"type"`
			Version  string `json:"version"`
		}{}
		if err := json.Unmarshal(body, &request); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "myhostname.lan", request.Hostname)
		assert.Equal(t, "metricbeat", request.Type)
		assert.Equal(t, "6.3.0", request.Version)

		fmt.Fprintf(w, `{"access_token": "fooo"}`)
	}))
	defer server.Close()

	accessToken, err := client.Enroll("metricbeat", "6.3.0", "myhostname.lan", beatUUID, "thisismyenrollmenttoken")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "fooo", accessToken)
}

func TestEnrollError(t *testing.T) {
	beatUUID := uuid.NewV4()

	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message": "Invalid enrollment token"}`, 400)
	}))
	defer server.Close()

	accessToken, err := client.Enroll("metricbeat", "6.3.0", "myhostname.lan", beatUUID, "thisismyenrollmenttoken")

	assert.NotNil(t, err)
	assert.Equal(t, "", accessToken)
}
