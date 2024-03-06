// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package add_cloud_metadata

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type azureMetadataFetcher struct {
	provider               string
	httpMetadataFetcher    *httpMetadataFetcher
	genericMetadataFetcher *genericFetcher
	httpMeta               mapstr.M
}

func newAzureMetadataFetcher(
	provider string,
	httpMetadataFetcher *httpMetadataFetcher,
) (*azureMetadataFetcher, error) {

	azFetcher := &azureMetadataFetcher{
		provider:            provider,
		httpMetadataFetcher: httpMetadataFetcher,
	}
	return azFetcher, nil
}

// NewClusterClient returns a NewManagedClustersClient
var NewClusterClient func(clientFactory *armcontainerservice.ClientFactory) *armcontainerservice.ManagedClustersClient = func(clientFactory *armcontainerservice.ClientFactory) *armcontainerservice.ManagedClustersClient {
	return clientFactory.NewManagedClustersClient()
}

// Azure VM Metadata Service
var azureVMMetadataFetcher = provider{
	Name: "azure-compute",

	Local: true,

	Create: func(_ string, config *conf.C) (metadataFetcher, error) {
		azMetadataURI := "/metadata/instance/compute?api-version=2021-02-01"
		azHeaders := map[string]string{"Metadata": "true"}
		azHttpSchema := func(m map[string]interface{}) mapstr.M {
			m["serviceName"] = "Virtual Machines"
			cloud, _ := s.Schema{
				"account": s.Object{
					"id": c.Str("subscriptionId"),
				},
				"instance": s.Object{
					"id":   c.Str("vmId"),
					"name": c.Str("name"),
				},
				"machine": s.Object{
					"type": c.Str("vmSize"),
				},
				"service": s.Object{
					"name": c.Str("serviceName"),
				},
				"region": c.Str("location"),
			}.Apply(m)

			azure, _ := s.Schema{
				"resourcegroup": s.Object{
					"name": c.Str("resourceGroupName"),
				},
			}.Apply(m)
			return mapstr.M{"cloud": cloud, "azure": azure}
		}

		azGenSchema := func(m map[string]interface{}) mapstr.M {
			orchestrator := mapstr.M{
				"orchestrator": mapstr.M{},
			}

			orchestrator.DeepUpdate(m)
			return orchestrator
		}

		// hfetcher represents an http fetcher to retrieve metadata from azure metadata endpoint
		hfetcher, err := newMetadataFetcher(config, "azure", azHeaders, metadataHost, azHttpSchema, azMetadataURI)
		if err != nil {
			return hfetcher, fmt.Errorf("failed to create new http metadata fetcher: %w", err)
		}
		// fetcher represents an azure metadata fetcher. The struct includes two type of fetchers.
		// 1. An http fetcher(hfetcher) which retrieves metadata from azure metadata endpoint and
		// 2. A generic fetcher(gfetcher) which uses azure sdk to retrieve metadata of azure managed clusters.
		fetcher, err := newAzureMetadataFetcher("azure", hfetcher)
		if err != nil {
			return fetcher, fmt.Errorf("failed to create new azure metadata fetcher: %w", err)
		}
		// gfetcher is created and assinged to fetcher after the fetcher is created in order the
		// fetchAzureClusterMeta to be a method of fetcher. This is needed so that the generic fetcher
		// can use the results/metadata that are already retrieved from http fetcher. SubscriptionId and
		// resourceGroupName are then used to filter azure managed clusters results.
		gfetcher, err := newGenericMetadataFetcher(config, "azure", azGenSchema, fetcher.fetchAzureClusterMeta)
		if err != nil {
			return fetcher, fmt.Errorf("failed to create new generic metadata fetcher: %w", err)
		}
		fetcher.genericMetadataFetcher = gfetcher
		return fetcher, nil
	},
}

// fetchMetadata fetches azure vm metadata from
// 1. Azure metadata endpoint with httpMetadataFetcher
// 2. Azure Managed Clusters using azure sdk  with genericMetadataFetcher
func (az *azureMetadataFetcher) fetchMetadata(ctx context.Context, client http.Client) result {
	res := result{provider: az.provider, metadata: mapstr.M{}, err: nil}
	logger := logp.NewLogger("add_cloud_metadata")
	httpRes := az.httpMetadataFetcher.fetchMetadata(ctx, client)
	if httpRes.err != nil {
		res.err = httpRes.err
		return res
	}
	res.metadata = httpRes.metadata
	az.httpMeta = httpRes.metadata
	gRes := az.genericMetadataFetcher.fetchMetadata(ctx, client)
	if gRes.err != nil {
		logger.Warnf("Failed to get additional AKS Cluster meta: %+v", gRes.err)
		return res
	}

	res.metadata.DeepUpdate(gRes.metadata)
	return res
}

// getAzureCredentials returns credentials to connect to Azure
// env vars TENANT_ID, CLIENT_ID and CLIENT_SECRET are required
// if not set, NewDefaultAzureCredential method will be used
func getAzureCredentials(logger *logp.Logger) (azcore.TokenCredential, error) {
	if os.Getenv("TENANT_ID") != "" && os.Getenv("CLIENT_ID") != "" && os.Getenv("CLIENT_SECRET") != "" {
		return azidentity.NewClientSecretCredential(os.Getenv("TENANT_ID"), os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"), nil)
	} else {
		logger.Debugf("No Client or Tenant configuration provided. Retrieving default Azure credentials")
		return azidentity.NewDefaultAzureCredential(nil)
	}
}

// getAKSClusterNameId returns the AKS cluster name and Id for a given resourceGroup
func getAKSClusterNameId(ctx context.Context, logger *logp.Logger, clusterClient *armcontainerservice.ManagedClustersClient, resourceGroupName string) (string, string, error) {
	pager := clusterClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to advance page: %w", err)
		}
		for _, v := range page.Value {
			if *v.Properties.NodeResourceGroup == resourceGroupName {
				return *v.Name, *v.ID, nil
			}

		}
	}
	return "", "", nil
}

// fetchAzureClusterMeta fetches metadata of Azure Managed Clusters using azure sdk.
func (az *azureMetadataFetcher) fetchAzureClusterMeta(
	ctx context.Context,
	client http.Client,
	result *result,
) {
	logger := logp.NewLogger("add_cloud_metadata")
	subscriptionId, _ := az.httpMeta.GetValue("cloud.account.id")
	resourceGroupName, _ := az.httpMeta.GetValue("azure.resourcegroup.name")
	strResourceGroupName := ""
	if val, ok := resourceGroupName.(string); ok {
		strResourceGroupName = val
	}
	strSubscriptionId := ""
	if val, ok := subscriptionId.(string); ok {
		strSubscriptionId = val
	}
	// if subscriptionId cannot be retrieved from metadata endpoint return an error
	if strSubscriptionId == "" {
		logger.Debugf("subscriptionId cannot be retrieved from metadata endpoint")
		result.err = fmt.Errorf("subscriptionId is required to create a new azure client")
		return
	}

	if strResourceGroupName == "" {
		result.err = fmt.Errorf("resourceGroupName is required to fetch AKS cluster name and cluster Id")
		return
	}
	cred, err := getAzureCredentials(logger)
	if err != nil {
		result.err = fmt.Errorf("failed to obtain azure credentials: %w", err)
		return
	}
	clientFactory, err := armcontainerservice.NewClientFactory(strSubscriptionId, cred, nil)
	if err != nil {
		result.err = fmt.Errorf("failed to create new armcontainerservice client factory: %w", err)
		return
	}

	clusterClient := NewClusterClient(clientFactory)
	clusterName, clusterId, err := getAKSClusterNameId(ctx, logger, clusterClient, strResourceGroupName)
	if err == nil {
		_, _ = result.metadata.Put("orchestrator.cluster.id", clusterId)
		_, _ = result.metadata.Put("orchestrator.cluster.name", clusterName)
	} else {
		result.err = fmt.Errorf("failed to get AKS cluster name and Id: %w", err)
	}
}
