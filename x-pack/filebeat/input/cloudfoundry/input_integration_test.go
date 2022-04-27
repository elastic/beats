// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && cloudfoundry
// +build integration,cloudfoundry

package cloudfoundry

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	cftest "github.com/elastic/beats/v7/x-pack/libbeat/common/cloudfoundry/test"
)

func TestInput(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors("cloudfoundry"))

	t.Run("v1", func(t *testing.T) {
		testInput(t, "v1")
	})

	t.Run("v2", func(t *testing.T) {
		testInput(t, "v2")
	})
}

func testInput(t *testing.T, version string) {
	config := conf.MustNewConfigFrom(cftest.GetConfigFromEnv(t))
	config.SetString("version", -1, version)

	input, err := Plugin().Manager.Create(config)
	require.NoError(t, err)

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure that there is something happening in the firehose
	apiAddress, err := config.String("api_address", -1)
	require.NoError(t, err)
	go makeApiRequests(t, ctx, apiAddress)

	ch := make(chan beat.Event)
	client := &pubtest.FakeClient{
		PublishFunc: func(evt beat.Event) {
			if ctx.Err() != nil {
				return
			}

			select {
			case ch <- evt:
			case <-ctx.Done():
			}
		},
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		inputCtx := v2.Context{
			Logger:      logp.NewLogger("test"),
			Cancelation: ctx,
		}
		input.Run(inputCtx, pubtest.ConstClient(client))
	}()

	select {
	case e := <-ch:
		t.Logf("Event received: %+v", e)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for events")
	}
}

func makeApiRequests(t *testing.T, ctx context.Context, address string) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	for {
		req, err := http.NewRequest(http.MethodGet, address, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}
