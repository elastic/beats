// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/mock"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	beatsContainer              = "beatscontainer"
	beatsContainer2             = "beatscontainer2"
	beatsMultilineJSONContainer = "beatsmultilinejsoncontainer"
	beatsJSONContainer          = "beatsjsoncontainer"
	beatsNdJSONContainer        = "beatsndjsoncontainer"
	beatsGzJSONContainer        = "beatsgzjsoncontainer"
)

func Test_StorageClient(t *testing.T) {
	t.Skip("Flaky test: issue -  https://github.com/elastic/beats/issues/34332")
	tests := []struct {
		name            string
		baseConfig      map[string]interface{}
		mockHandler     func() http.Handler
		expected        map[string]bool
		isError         error
		unexpectedError error
	}{
		{
			name: "SingleContainerWithPoll_NoErr",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
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
			unexpectedError: context.Canceled,
		},
		{
			name: "SingleContainerWithoutPoll_NoErr",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                false,
				"poll_interval":                       "10s",
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
			unexpectedError: nil,
		},
		{
			name: "TwoContainersWithPoll_NoErr",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsContainer,
					},
					{
						"name": beatsContainer2,
					},
				},
			},
			mockHandler: mock.AzureStorageServer,
			expected: map[string]bool{
				mock.Beatscontainer_blob_ata_json:      true,
				mock.Beatscontainer_blob_data3_json:    true,
				mock.Beatscontainer_blob_docs_ata_json: true,
				mock.Beatscontainer_2_blob_ata_json:    true,
				mock.Beatscontainer_2_blob_data3_json:  true,
			},
			unexpectedError: context.Canceled,
		},
		{
			name: "TwoContainersWithoutPoll_NoErr",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                false,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsContainer,
					},
					{
						"name": beatsContainer2,
					},
				},
			},
			mockHandler: mock.AzureStorageServer,
			expected: map[string]bool{
				mock.Beatscontainer_blob_ata_json:      true,
				mock.Beatscontainer_blob_data3_json:    true,
				mock.Beatscontainer_blob_docs_ata_json: true,
				mock.Beatscontainer_2_blob_ata_json:    true,
				mock.Beatscontainer_2_blob_data3_json:  true,
			},
			unexpectedError: context.Canceled,
		},
		{
			name: "SingleContainerPoll_InvalidContainerErr",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": "azuretest",
					},
				},
			},
			mockHandler:     mock.AzureStorageServer,
			expected:        map[string]bool{},
			isError:         mock.NotFoundErr,
			unexpectedError: nil,
		},
		{
			name: "SingleContainerWithoutPoll_InvalidBucketErr",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                false,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": "azuretest",
					},
				},
			},
			mockHandler:     mock.AzureStorageServer,
			expected:        map[string]bool{},
			isError:         mock.NotFoundErr,
			unexpectedError: nil,
		},
		{
			name: "TwoContainersWithPoll_InvalidBucketErr",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": "azurenew",
					},
					{
						"name": "azurelatest",
					},
				},
			},
			mockHandler:     mock.AzureStorageServer,
			expected:        map[string]bool{},
			isError:         mock.NotFoundErr,
			unexpectedError: nil,
		},
		{
			name: "SingleBucketWithPoll_InvalidConfigValue",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         5100,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsContainer,
					},
				},
			},
			mockHandler:     mock.AzureStorageServer,
			expected:        map[string]bool{},
			isError:         errors.New("requires value <= 5000 accessing 'max_workers'"),
			unexpectedError: nil,
		},
		{
			name: "TwoBucketWithPoll_InvalidConfigValue",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         5100,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsContainer,
					},
					{
						"name": beatsContainer2,
					},
				},
			},
			mockHandler:     mock.AzureStorageServer,
			expected:        map[string]bool{},
			isError:         errors.New("requires value <= 5000 accessing 'max_workers'"),
			unexpectedError: nil,
		},
		{
			name: "ReadJSON",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsJSONContainer,
					},
				},
			},
			mockHandler: mock.AzureStorageFileServer,
			expected: map[string]bool{
				mock.BeatsFilesContainer_log_json[0]: true,
				mock.BeatsFilesContainer_log_json[1]: true,
				mock.BeatsFilesContainer_log_json[2]: true,
			},
			unexpectedError: context.Canceled,
		},
		{
			name: "ReadOctetStreamJSON",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsMultilineJSONContainer,
					},
				},
			},
			mockHandler: mock.AzureStorageFileServer,
			expected: map[string]bool{
				mock.BeatsFilesContainer_multiline_json[0]: true,
				mock.BeatsFilesContainer_multiline_json[1]: true,
			},
			unexpectedError: context.Canceled,
		},
		{
			name: "ReadNdJSON",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsNdJSONContainer,
					},
				},
			},
			mockHandler: mock.AzureStorageFileServer,
			expected: map[string]bool{
				mock.BeatsFilesContainer_log_ndjson[0]: true,
				mock.BeatsFilesContainer_log_ndjson[1]: true,
			},
			unexpectedError: context.Canceled,
		},
		{
			name: "ReadMultilineGzJSON",
			baseConfig: map[string]interface{}{
				"account_name":                        "beatsblobnew",
				"auth.shared_credentials.account_key": "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
				"max_workers":                         2,
				"poll":                                true,
				"poll_interval":                       "10s",
				"containers": []map[string]interface{}{
					{
						"name": beatsGzJSONContainer,
					},
				},
			},
			mockHandler: mock.AzureStorageFileServer,
			expected: map[string]bool{
				mock.BeatsFilesContainer_multiline_json_gz[0]: true,
				mock.BeatsFilesContainer_multiline_json_gz[1]: true,
			},
			unexpectedError: context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := httptest.NewServer(tt.mockHandler())
			t.Cleanup(serv.Close)

			cfg := conf.MustNewConfigFrom(tt.baseConfig)
			conf := config{}
			err := cfg.Unpack(&conf)
			if err != nil {
				assert.EqualError(t, err, tt.isError.Error())
				return
			}
			input := newStatelessInput(conf, serv.URL+"/")

			assert.Equal(t, "azure-blob-storage-stateless", input.Name())
			assert.NoError(t, input.Test(v2.TestContext{}))

			chanClient := beattest.NewChanClient(len(tt.expected))
			t.Cleanup(func() { _ = chanClient.Close() })

			ctx, cancel := newV2Context()
			t.Cleanup(cancel)

			var g errgroup.Group
			g.Go(func() error {
				return input.Run(ctx, chanClient)
			})

			var timeout *time.Timer
			if conf.PollInterval != nil {
				timeout = time.NewTimer(1*time.Second + *conf.PollInterval)
			} else {
				timeout = time.NewTimer(5 * time.Second)
			}
			t.Cleanup(func() { timeout.Stop() })

			if len(tt.expected) == 0 {
				if tt.isError != nil && g.Wait() != nil {
					//nolint:errorlint // This will never be a wrapped error
					if tt.isError == mock.NotFoundErr {
						arr := strings.Split(g.Wait().Error(), "\n")
						errStr := strings.Join(arr[1:], "\n")
						assert.Equal(t, tt.isError.Error(), errStr)
					} else {
						assert.EqualError(t, g.Wait(), tt.isError.Error())
					}
					cancel()
				} else {
					assert.NoError(t, g.Wait())
					cancel()
				}
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
					var val interface{}
					var err error
					val, err = got.Fields.GetValue("message")
					assert.NoError(t, err)
					assert.True(t, tt.expected[val.(string)])
					assert.Equal(t, tt.isError, err)
					receivedCount += 1
					if receivedCount == len(tt.expected) {
						cancel()
						break wait
					}
				}
			}
			assert.ErrorIs(t, g.Wait(), tt.unexpectedError)
		})
	}
}

func newV2Context() (v2.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	return v2.Context{
		Logger:      logp.NewLogger("azure-blob-storage_test"),
		ID:          "test_id",
		Cancelation: ctx,
	}, cancel
}
