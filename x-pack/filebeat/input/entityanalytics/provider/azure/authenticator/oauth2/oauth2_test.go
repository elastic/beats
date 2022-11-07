// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func testSetupServer(tokenValue string, expiresIn int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := authResponse{
			TokenType:    "Bearer",
			AccessToken:  tokenValue,
			ExpiresIn:    expiresIn,
			ExtExpiresIn: expiresIn,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			panic(err)
		}

		_, err = w.Write(data)
		if err != nil {
			panic(err)
		}

		w.Header().Add("Content-Type", "application/json")
	}))
}

func TestRenew(t *testing.T) {
	t.Run("new-token", func(t *testing.T) {
		value := "test-value"
		expiresIn := 1000

		srv := testSetupServer(value, expiresIn)
		defer srv.Close()

		cfg, err := config.NewConfigFrom(&conf{
			Endpoint: "http://" + srv.Listener.Addr().String(),
			Secret:   "value",
			ClientID: "client-id",
			TenantID: "tenant-id",
		})
		assert.NoError(t, err)

		auth, err := New(cfg, logp.L())
		assert.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		gotToken, err := auth.Token(ctx)
		assert.NoError(t, err)

		assert.WithinDuration(t, time.Now().Add(time.Duration(expiresIn)*time.Second), auth.(*oauth2).expires, 5*time.Second)
		assert.Equal(t, value, gotToken)
	})

	t.Run("cached-token", func(t *testing.T) {
		cachedToken := "cached-value"
		expireTime := time.Now().Add(1000 * time.Second)

		srv := testSetupServer(cachedToken, 1000)
		defer srv.Close()

		cfg, err := config.NewConfigFrom(&conf{
			Endpoint: "http://" + srv.Listener.Addr().String(),
			Secret:   "value",
			ClientID: "client-id",
			TenantID: "tenant-id",
		})
		assert.NoError(t, err)

		auth, err := New(cfg, logp.L())
		assert.NoError(t, err)

		auth.(*oauth2).expires = expireTime
		auth.(*oauth2).token = cachedToken

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		gotToken, err := auth.Token(ctx)
		assert.NoError(t, err)

		assert.Equal(t, expireTime, auth.(*oauth2).expires)
		assert.Equal(t, cachedToken, gotToken)
	})
}
