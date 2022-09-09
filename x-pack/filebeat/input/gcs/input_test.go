// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/test/mock"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func Test_StorageClient(t *testing.T) {
	tests := []struct {
		name            string
		baseConfig      map[string]interface{}
		mockHandler     func() http.Handler
		expected        map[string]bool
		wantErr         bool
		unexpectedError error
	}{
		{
			name: "singleBucketWithPollNoErr",
			baseConfig: map[string]interface{}{
				"project_id":                 "elastic-sa",
				"auth.credentials_file.path": "/gcs_creds.json",
				"max_workers":                2,
				"poll":                       true,
				"poll_interval":              "5s",
				"parse_json":                 false,
				"buckets": []map[string]interface{}{
					{
						"name": "gcs-test-new",
					},
				},
			},
			mockHandler: mock.GCSServer,
			expected: map[string]bool{
				mock.Gcs_test_new_object_ata_json:      true,
				mock.Gcs_test_new_object_data3_json:    true,
				mock.Gcs_test_new_object_docs_ata_json: true,
			},
			wantErr:         false,
			unexpectedError: context.Canceled,
		},
		{
			name: "singleBucketWithoutPollNoErr",
			baseConfig: map[string]interface{}{
				"project_id":                 "elastic-sa",
				"auth.credentials_file.path": "/gcs_creds.json",
				"max_workers":                2,
				"poll":                       false,
				"poll_interval":              "10s",
				"parse_json":                 false,
				"buckets": []map[string]interface{}{
					{
						"name": "gcs-test-new",
					},
				},
			},
			mockHandler: mock.GCSServer,
			expected: map[string]bool{
				mock.Gcs_test_new_object_ata_json:      true,
				mock.Gcs_test_new_object_data3_json:    true,
				mock.Gcs_test_new_object_docs_ata_json: true,
			},
			wantErr:         false,
			unexpectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := httptest.NewTLSServer(tt.mockHandler())
			httpclient := http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
			client, _ := storage.NewClient(context.Background(), option.WithEndpoint(serv.URL), option.WithoutAuthentication(), option.WithHTTPClient(&httpclient))
			cfg := conf.MustNewConfigFrom(tt.baseConfig)
			conf := config{}
			assert.NoError(t, cfg.Unpack(&conf))
			input := newStatelessInput(conf)

			assert.Equal(t, "gcs-stateless", input.Name())
			assert.NoError(t, input.Test(v2.TestContext{}))

			chanClient := beattest.NewChanClient(len(tt.expected))
			t.Cleanup(func() { _ = chanClient.Close() })

			ctx, cancel := newV2Context()
			t.Cleanup(cancel)

			var g errgroup.Group
			g.Go(func() error {
				return input.Run(ctx, chanClient, client)
			})

			timeout := time.NewTimer(1*time.Second + *conf.PollInterval)
			t.Cleanup(func() { _ = timeout.Stop() })

			if len(tt.expected) == 0 {
				cancel()
				assert.NoError(t, g.Wait())
				return
			}

			var receivedCount int
		wait:
			for {
				select {
				case <-timeout.C:
					t.Errorf("timed out waiting for %d events", len(tt.expected))
					cancel()
					return
				case got := <-chanClient.Channel:
					val, err := got.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.True(t, tt.expected[val.(string)])
					assert.Equal(t, tt.wantErr, !(err == nil))
					receivedCount += 1
					if receivedCount == len(tt.expected) {
						cancel()
						break wait
					}
				}
			}
			assert.ErrorIs(t, tt.unexpectedError, g.Wait())
		})
	}
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("gcs_test"),
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
}
