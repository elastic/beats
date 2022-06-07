package azureblobstorage

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/elastic/beats/v7/libbeat/statestore"
)

func (input *azurebsInput) collect(ctx context.Context, persistentStore *statestore.Store) {
	containers := input.fetchContainerList(ctx)

	for _, v := range containers {
		input.fetchBlobsFromContainer(ctx, v)
	}

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

func (input *azurebsInput) fetchBlobsFromContainer(ctx context.Context, container *azblob.ContainerItem) (*blobClientObj, error) {
	var blobs []*azblob.BlobItemInternal

	blobClient, err := fetchBlobClients(input.serviceURL, input.credential, input.log)
	if err != nil {
		return nil, err
	}
	containerClient, err := input.client.NewContainerClient(*container.Name)
	if err != nil {
		input.log.Errorf("Error fetching blob client object for container : %s, error : %v", container.Name, err)
		return nil, err
	}
	blobPager := containerClient.ListBlobsFlat(nil)

	for blobPager.NextPage(ctx) {
		resp := blobPager.PageResponse()

		blobs = append(blobs, resp.Segment.BlobItems...)
	}

	return &blobClientObj{
		client: blobClient,
		blobs:  blobs,
	}, nil
}
