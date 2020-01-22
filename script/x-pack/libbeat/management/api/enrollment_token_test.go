// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnrollmentToken(t *testing.T) {
	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check correct path is used
		assert.Equal(t, "/api/beats/enrollment_tokens", r.URL.Path)
		fmt.Fprintf(w, `{"results": [{"item":"65074ff8639a4661ba7e1bd5ccc209ed"}]}`)
	}))
	defer server.Close()

	token, err := client.CreateEnrollmentToken()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "65074ff8639a4661ba7e1bd5ccc209ed", token)
}
