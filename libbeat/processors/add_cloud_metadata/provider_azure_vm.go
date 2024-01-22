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

// Azure VM Metadata Service
var azureVMMetadataFetcher = provider{
	Name: "azure-compute",

	Local: true,

	Create: func(_ string, config *conf.C) (metadataFetcher, error) {
		azMetadataURI := "/metadata/instance/compute?api-version=2021-02-01"
		azHeaders := map[string]string{"Metadata": "true"}
		azHttpSchema := func(m map[string]interface{}) mapstr.M {
			m["serviceName"] = "Virtual Machines"
			out, _ := s.Schema{
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
				"resourceGroup": s.Object{
					"name": c.Str("resourceGroupName"),
				},
			}.Apply(m)
			return mapstr.M{"cloud": out}
		}

		azGenSchema := func(m map[string]interface{}) mapstr.M {
			orchestrator := mapstr.M{
				"orchestrator": mapstr.M{},
			}

			orchestrator.DeepUpdate(m)
			return orchestrator
		}

		hfetcher, err := newMetadataFetcher(config, "azure", azHeaders, metadataHost, azHttpSchema, azMetadataURI)
		fetcher, err := newAzureMetadataFetcher("azure", hfetcher)
		gfetcher, err := newGenericMetadataFetcher(config, "azure", azGenSchema, fetcher.fetchAzureMetadata)
		fetcher.genericMetadataFetcher = gfetcher
		return fetcher, err
	},
}

type azureMetadataFetcher struct {
	provider               string
	httpMetadataFetcher    *httpMetadataFetcher
	genericMetadataFetcher *genericFetcher
	meta                   mapstr.M
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

func (az *azureMetadataFetcher) fetchMetadata(ctx context.Context, client http.Client) result {
	res := result{provider: az.provider, metadata: mapstr.M{}, err: nil}
	logger := logp.NewLogger("add_cloud_metadata")
	httpRes := az.httpMetadataFetcher.fetchMetadata(ctx, client)
	if httpRes.err != nil {
		res.err = httpRes.err
		return res
	}
	az.meta = httpRes.metadata
	gRes := az.genericMetadataFetcher.fetchMetadata(ctx, client)
	if gRes.err != nil {
		res.err = gRes.err
		return res
	}
	res.metadata = httpRes.metadata
	res.metadata.DeepUpdate(gRes.metadata)
	logger.Infof("Full result: %+v", res)
	return res
}

func getAzureCredentials(logger *logp.Logger) (azcore.TokenCredential, error) {
	if os.Getenv("TENANT_ID") != "" && os.Getenv("CLIENT_ID") != "" && os.Getenv("CLIENT_SECRET") != "" {
		return azidentity.NewClientSecretCredential(os.Getenv("TENANT_ID"), os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET"), nil)
	} else {
		logger.Debugf("No Client or Tenant configuration provided. Retrieving default Azure credentials")
		return azidentity.NewDefaultAzureCredential(nil)
	}
}

func getAKSClusterNameId(logger *logp.Logger, subscriptionId, resourceGroupName string) (string, string, error) {
	if subscriptionId == "" {
		subscriptionId = os.Getenv("SUBSCRIPTION_ID")
		if subscriptionId == "" {
			return "", "", fmt.Errorf("subscriptionId is required to create a new azure client")
		}
	}

	if resourceGroupName == "" {
		return "", "", fmt.Errorf("resourceGroupName is required to fetch cluster name and cluster Id")
	}
	cred, err := getAzureCredentials(logger)
	if err != nil {
		logger.Errorf("failed to obtain a credential: %v", err)
		return "", "", fmt.Errorf("failed to obtain a credential: %v", err)
	}
	ctx := context.Background()
	clientFactory, err := armcontainerservice.NewClientFactory(subscriptionId, cred, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create client: %v", err)
	}
	pager := clientFactory.NewManagedClustersClient().NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to advance page: %v", err)
		}
		for _, v := range page.Value {
			if *v.Properties.NodeResourceGroup == resourceGroupName {
				return *v.Name, *v.ID, nil
			}

		}
	}
	return "", "", nil
}

// fetchAzureMetadata queries raw metadata from a hosting provider's metadata service.
func (az *azureMetadataFetcher) fetchAzureMetadata(
	ctx context.Context,
	client http.Client,
	result *result,
) {
	logger := logp.NewLogger("add_cloud_metadata")
	subscriptionId, _ := az.meta.GetValue("cloud.account.id")
	resourceGroupName, _ := az.meta.GetValue("cloud.resourceGroup.name")
	strResourceGroupName := ""
	if val, ok := resourceGroupName.(string); ok {
		strResourceGroupName = fmt.Sprintf("%s", val)
	}
	strSubscriptionId := ""
	if val, ok := subscriptionId.(string); ok {
		strSubscriptionId = fmt.Sprintf("%s", val)
	}
	clusterName, clusterId, err := getAKSClusterNameId(logger, strSubscriptionId, strResourceGroupName)
	if err == nil {
		_, _ = result.metadata.Put("orchestrator.cluster.id", clusterId)
		_, _ = result.metadata.Put("orchestrator.cluster.name", clusterName)
	} else {
		logger.Debugf(fmt.Sprintf("failed to getAKSClusterNameId: %v", err))
	}
}
