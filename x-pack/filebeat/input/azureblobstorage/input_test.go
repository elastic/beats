// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/mock"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"gotest.tools/gotestsum/log"
)

const (
	bucketGcsTestNew    = "gcs-test-new"
	bucketGcsTestLatest = "gcs-test-latest"
)

func Test_StorageClient(t *testing.T) {
	tests := []struct {
		name            string
		baseConfig      map[string]interface{}
		mockHandler     func() http.Handler
		expected        map[string]bool
		checkJSON       bool
		isError         error
		unexpectedError error
	}{
		{
			name: "Test1_SingleBucketWithPoll_NoErr",
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
			unexpectedError: context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serv := httptest.NewServer(tt.mockHandler())
			// httpclient := http.Client{
			// 	Transport: &http.Transport{
			// 		TLSClientConfig: &tls.Config{
			// 			InsecureSkipVerify: true, //nolint:gosec // We can ignore as this is just for testing
			// 		},
			// 	},
			// }
			t.Cleanup(serv.Close)
			fetchdata(serv)
			// 	client, _ := storage.NewClient(context.Background(), option.WithEndpoint(serv.URL), option.WithoutAuthentication(), option.WithHTTPClient(&httpclient))
			// 	cfg := conf.MustNewConfigFrom(tt.baseConfig)
			// 	conf := config{}
			// 	err := cfg.Unpack(&conf)
			// 	if err != nil {
			// 		assert.EqualError(t, err, tt.isError.Error())
			// 		return
			// 	}
			// 	input := newStatelessInput(conf)

			// 	assert.Equal(t, "gcs-stateless", input.Name())
			// 	assert.NoError(t, input.Test(v2.TestContext{}))

			// 	chanClient := beattest.NewChanClient(len(tt.expected))
			// 	t.Cleanup(func() { _ = chanClient.Close() })

			// 	ctx, cancel := newV2Context()
			// 	t.Cleanup(cancel)

			// 	var g errgroup.Group
			// 	g.Go(func() error {
			// 		return input.Run(ctx, chanClient, client)
			// 	})

			// 	var timeout *time.Timer
			// 	if conf.PollInterval != nil {
			// 		timeout = time.NewTimer(1*time.Second + *conf.PollInterval)
			// 	} else {
			// 		timeout = time.NewTimer(5 * time.Second)
			// 	}
			// 	t.Cleanup(func() { _ = timeout.Stop() })

			// 	if len(tt.expected) == 0 {
			// 		if tt.isError != nil && g.Wait() != nil {
			// 			assert.EqualError(t, g.Wait(), tt.isError.Error())
			// 			cancel()
			// 		} else {
			// 			assert.NoError(t, g.Wait())
			// 			cancel()
			// 		}
			// 		return
			// 	}

			// 	var receivedCount int
			// wait:
			// 	for {
			// 		select {
			// 		case <-timeout.C:
			// 			t.Errorf("timed out waiting for %d events", len(tt.expected))
			// 			cancel()
			// 			return
			// 		case got := <-chanClient.Channel:
			// 			var val interface{}
			// 			var err error
			// 			if !tt.checkJSON {
			// 				val, err = got.Fields.GetValue("message")
			// 				assert.NoError(t, err)
			// 				assert.True(t, tt.expected[val.(string)])
			// 			} else {
			// 				val, err = got.Fields.GetValue("gcs.storage.object.json_data")
			// 				fVal := fmt.Sprintf("%v", val)
			// 				assert.NoError(t, err)
			// 				assert.True(t, tt.expected[fVal])
			// 			}
			// 			assert.Equal(t, tt.isError, err)
			// 			receivedCount += 1
			// 			if receivedCount == len(tt.expected) {
			// 				cancel()
			// 				break wait
			// 			}
			// 		}
			// 	}
			// 	assert.ErrorIs(t, tt.unexpectedError, g.Wait())
		})
	}
}

func fetchdata(serv *httptest.Server) {
	credential, err := azblob.NewSharedKeyCredential("xyz", "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==")
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
	}

	client, err := azblob.NewServiceClientWithSharedKey(serv.URL, credential, nil)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
	}

	containerClient, err := client.NewContainerClient("beatscontainer")
	if err != nil {
		log.Errorf("Error fetching container client for container : %s, error : %v", "beatscontainer", err)
	}

	pager := containerClient.ListBlobsFlat(&azblob.ContainerListBlobsFlatOptions{
		Include: []azblob.ListBlobsIncludeItem{
			azblob.ListBlobsIncludeItemMetadata,
			azblob.ListBlobsIncludeItemTags,
		},
	})
	ctx := context.Background()
	for pager.NextPage(ctx) {
		for _, v := range pager.PageResponse().Segment.BlobItems {
			blobURL := serv.URL + "beatscontainer" + "/" + *v.Name
			blobCreds := &types.BlobCredentials{
				ServiceCreds:  ais.credential,
				BlobName:      *v.Name,
				ContainerName: ais.src.ContainerName,
			}

			fmt.Println("NAME : ", *v.Name)

			// blobClient, err := fetchBlobClient(blobURL, blobCreds, ais.log)
			// if err != nil {
			// 	return nil, err
			// }

		}
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
