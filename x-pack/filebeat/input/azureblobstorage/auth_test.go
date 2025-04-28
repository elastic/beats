// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/mock"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// customTransporter implements the Transporter interface with a custom Do & RoundTrip method
type customTransporter struct {
	rt      http.RoundTripper
	servURL string
}

func (t *customTransporter) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.rt.RoundTrip(req)
}

// Do is responsible for the routing of the request to the appropriate handler based on the request URL
func (t *customTransporter) Do(req *http.Request) (*http.Response, error) {
	logp.L().Named("azure-blob-storage-test").Debug("request URL: ", req.URL)
	re := regexp.MustCompile(`^/([0-9a-fA-F-]+)/?(oauth2/v2\.0/token|v2\.0/\.well-known/openid-configuration)`)
	matches := re.FindStringSubmatch(req.URL.Path)

	if len(matches) == 3 {
		tenant_id := matches[1]
		action := matches[2]

		switch action {
		case "v2.0/.well-known/openid-configuration":
			return createJSONResponse(map[string]interface{}{
				"token_endpoint":         t.servURL + "/" + tenant_id + "/oauth2/v2.0/token",
				"authorization_endpoint": t.servURL + "/" + tenant_id + "/oauth2/v2.0/authorize",
				"issuer":                 t.servURL + "/" + tenant_id + "/oauth2/v2.0/issuer",
			}, 200)

		case "oauth2/v2.0/token":
			return createJSONResponse(map[string]interface{}{
				"token_type":   "Bearer",
				"expires_in":   3600,
				"access_token": "mock_access_token_123",
			}, 200)
		}
	}
	return t.rt.RoundTrip(req)
}

func createJSONResponse(data interface{}, statusCode int) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewBuffer(jsonData)),
		Header:     make(http.Header),
	}

	resp.Header.Set("Content-Type", "application/json")
	return resp, nil
}

func Test_OAuth2(t *testing.T) {
	tests := []struct {
		name        string
		baseConfig  map[string]interface{}
		mockHandler func() http.Handler
		expected    map[string]bool
	}{
		{
			name: "OAuth2TConfig",
			baseConfig: map[string]interface{}{
				"account_name": "beatsblobnew",
				"auth.oauth2": map[string]interface{}{
					"client_id":     "12345678-90ab-cdef-1234-567890abcdef",
					"client_secret": "abcdefg1234567890!@#$%^&*()-_=+",
					"tenant_id":     "87654321-abcd-ef90-1234-fedcba098765",
				},
				"max_workers":   2,
				"poll":          true,
				"poll_interval": "30s",
				"containers": []map[string]interface{}{
					{
						"name": beatsContainer,
					},
				},
			},
			mockHandler: mock.AzureStorageServer,
			expected: map[string]bool{
				mock.Beatscontainer_blob_ata_json:      true,
				mock.Beatscontainer_blob_data3_json:    true,
				mock.Beatscontainer_blob_docs_ata_json: true,
			},
		},
	}

	logp.TestingSetup()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := httptest.NewServer(tt.mockHandler())
			t.Cleanup(serv.Close)

			httpClient := &http.Client{
				Transport: &customTransporter{
					rt:      http.DefaultTransport,
					servURL: serv.URL,
				},
			}

			cfg := conf.MustNewConfigFrom(tt.baseConfig)
			conf := config{}
			err := cfg.Unpack(&conf)
			assert.NoError(t, err)

			// inject custom transport & client options
			conf.Auth.OAuth2.clientOptions = azcore.ClientOptions{
				InsecureAllowCredentialWithHTTP: true,
				Transport:                       httpClient.Transport.(*customTransporter),
			}

			input := newStatelessInput(conf, serv.URL+"/")

			assert.Equal(t, "azure-blob-storage-stateless", input.Name())
			assert.NoError(t, input.Test(v2.TestContext{}))

			chanClient := beattest.NewChanClient(len(tt.expected))
			t.Cleanup(func() { _ = chanClient.Close() })

			ctx, cancel := newV2Context(t)
			t.Cleanup(cancel)
			ctx.ID += tt.name

			var g errgroup.Group
			g.Go(func() error {
				return input.Run(ctx, chanClient)
			})

			var timeout *time.Timer
			if conf.PollInterval != nil {
				timeout = time.NewTimer(1*time.Second + *conf.PollInterval)
			} else {
				timeout = time.NewTimer(10 * time.Second)
			}
			t.Cleanup(func() { timeout.Stop() })

			var receivedCount int
		wait:
			for {
				select {
				case <-timeout.C:
					t.Errorf("timed out waiting for %d events", len(tt.expected))
					cancel()
					return
				case got := <-chanClient.Channel:
					var val interface{}
					var err error
					val, err = got.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.True(t, tt.expected[val.(string)])
					receivedCount += 1
					if receivedCount == len(tt.expected) {
						cancel()
						break wait
					}
				}
			}
		})
	}
}
