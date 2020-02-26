// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/x-pack/agent/pkg/release"
)

func TestAddingHeaders(t *testing.T) {
	msg := []byte("OK")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, fmt.Sprintf("Beat agent v%s", release.Version()), req.Header.Get("User-Agent"))
		w.Write(msg)
	}))
	defer server.Close()

	c := server.Client()
	rtt := withHeaders(c.Transport, headers)

	c.Transport = rtt
	resp, err := c.Get(server.URL)
	require.NoError(t, err)
	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, b, msg)
}
