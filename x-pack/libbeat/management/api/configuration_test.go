// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	beatUUID := uuid.NewV4()

	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check correct path is used
		assert.Equal(t, "/api/beats/agent/"+beatUUID.String()+"/configuration", r.URL.Path)

		// Check enrollment token is correct
		assert.Equal(t, "thisismyenrollmenttoken", r.Header.Get("kbn-beats-access-token"))

		fmt.Fprintf(w, `{"configuration_blocks":[{"type":"filebeat.modules","block_yml":"module: apache2\n"},{"type":"metricbeat.modules","block_yml":"module: nginx\n"}]}`)
	}))
	defer server.Close()

	configs, err := client.Configuration("thisismyenrollmenttoken", beatUUID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(configs))
}
