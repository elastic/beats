// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/elastic/beats/v7/libbeat/statestore"
)

func (input *azurebsInput) collect(ctx context.Context, persistentStore *statestore.Store) error {
	containers := input.fetchContainerList(ctx)

	for _, v := range containers {
		blobClientArr, err := input.fetchBlobsFromContainer(ctx, v)
		if err != nil {
			return err
		}

		for _, x := range blobClientArr {
			dataBuffer, err := x.extractData(ctx)
			if err != nil {
				return err
			}

			input.log.Infof("data from container %s and blob %s is : %s ", *v.Name, *x.blob.Name, strings.TrimSpace(dataBuffer.String()))
		}
	}

	return nil
}

func (input *azurebsInput) fetchContainerList(ctx context.Context) []*azblob.ContainerItem {
	var containers []*azblob.ContainerItem
	containerPager := input.client.ListContainers(nil)
	for containerPager.NextPage(ctx) {
		resp := containerPager.PageResponse()
		containers = append(containers, resp.ListContainersSegmentResponse.ContainerItems...)
	}

	return containers
}

func (input *azurebsInput) fetchBlobsFromContainer(ctx context.Context, container *azblob.ContainerItem) ([]*blobClientObj, error) {
	var blobClientArr []*blobClientObj

	containerClient, err := input.client.NewContainerClient(*container.Name)
	if err != nil {
		input.log.Errorf("Error fetching blob client object for container : %s, error : %v", container.Name, err)
		return nil, err
	}
	blobPager := containerClient.ListBlobsFlat(nil)

	for blobPager.NextPage(ctx) {
		resp := blobPager.PageResponse()

		for _, v := range resp.Segment.BlobItems {
			blobURL := fmt.Sprintf("%s%s/%s", input.serviceURL, *container.Name, *v.Name)

			blobClient, err := fetchBlobClients(blobURL, input.credential, input.log)
			if err != nil {
				return nil, err
			}

			blobClientArr = append(blobClientArr, &blobClientObj{
				client: blobClient,
				blob:   v,
			})
		}

	}

	return blobClientArr, nil
}

func (bc *blobClientObj) extractData(ctx context.Context) (*bytes.Buffer, error) {
	get, err := bc.client.Download(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download data from blob with error : %v", err)
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(&azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from blob with error : %v", err)
	}

	err = reader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close blob reader with error : %v", err)
	}

	return downloadedData, nil
}
