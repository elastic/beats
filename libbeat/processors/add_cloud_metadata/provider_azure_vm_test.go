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
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v4/fake"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azfake "github.com/Azure/azure-sdk-for-go/sdk/azcore/fake"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var cluster1Name = "testcluster1Name"
var cluster1Id = "testcluster1Id"

var cluster1 = armcontainerservice.ManagedCluster{
	ID:         to.Ptr(cluster1Id),
	Name:       to.Ptr(cluster1Name),
	Properties: &armcontainerservice.ManagedClusterProperties{NodeResourceGroup: to.Ptr("MC_myname_group_myname_eastus")},
}

const azInstanceIdentityDocument = `{
	"azEnvironment": "AzurePublicCloud",
	"customData": "",
	"evictionPolicy": "",
	"isHostCompatibilityLayerVm": "false",
	"licenseType": "",
	"location": "eastus",
	"name": "aks-agentpool-12628255-vmss_2",
	"offer": "",
	"osProfile": {
		"adminUsername": "azureuser",
		"computerName": "aks-agentpool-12628255-vmss000002",
		"disablePasswordAuthentication": "true"
	},
	"osType": "Linux",
	"placementGroupId": "43e6bd16-b9ae-4a0c-a7e3-6c9ab23482a7",
	"plan": {
		"name": "",
		"product": "",
		"publisher": ""
	},
	"platformFaultDomain": "0",
	"platformUpdateDomain": "0",
	"priority": "",
	"provider": "Microsoft.Compute",
	"publicKeys": [],
	"publisher": "",
	"resourceGroupName": "MC_myname_group_myname_eastus",
	"resourceId": "/subscriptions/0e073ec1-c22f-4488-adde-da35ed609ccd/resourceGroups/MC_myname_group_myname_eastus/providers/Microsoft.Compute/virtualMachineScaleSets/aks-agentpool-12628255-vmss/virtualMachines/2",
	"securityProfile": {
		"secureBootEnabled": "false",
		"virtualTpmEnabled": "false"
	},
	"sku": "",
	"storageProfile": {},
	"subscriptionId": "0e073ec1-c22f-4488-adde-da35ed609ccd",
	"tags": "aks-managed-coordination:true;aks-managed-createOperationID:29bea4bf-8a24-4dcb-aecb-c2fb07d5bd29;aks-managed-creationSource:vmssclient-aks-agentpool-12628255-vmss;aks-managed-kubeletIdentityClientID:64efabb1-53aa-4868-8a07-9e385f00e527;aks-managed-orchestrator:Kubernetes:1.25.6;aks-managed-poolName:agentpool;aks-managed-resourceNameSuffix:80220090",
	"tagsList": [],
	"userData": "",
	"version": "",
	"vmId": "220e2b43-0913-492f-ada0-b6b2795bdb9b",
	"vmScaleSetName": "aks-agentpool-12628255-vmss",
	"vmSize": "Standard_DS2_v2",
	"zone": "3"
}`

func initAzureTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/metadata/instance/compute?api-version=2021-02-01" && r.Header.Get("Metadata") == "true" {
			_, _ = w.Write([]byte(azInstanceIdentityDocument))
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
}

// NewTokenCredential creates an instance of the TokenCredential type.
func newTokenCredential() *TokenCredential {
	return &TokenCredential{}
}

// TokenCredential is a fake credential that implements the azcore.TokenCredential interface.
type TokenCredential struct {
	err error
}

// SetError sets the specified error to be returned from GetToken().
// Use this to simulate an error during authentication.
func (t *TokenCredential) SetError(err error) {
	t.err = fmt.Errorf("Token cannot be created")
}

// GetToken implements the azcore.TokenCredential for the TokenCredential type.
func (t *TokenCredential) GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error) {
	if t.err != nil {
		return azcore.AccessToken{}, t.err
	}
	return azcore.AccessToken{Token: "fake_token", ExpiresOn: time.Now().Add(24 * time.Hour)}, nil
}

func TestRetrieveAzureMetadata(t *testing.T) {

	fakeMCServer := fake.ManagedClustersServer{
		NewListPager: func(options *armcontainerservice.ManagedClustersClientListOptions) (resp azfake.PagerResponder[armcontainerservice.ManagedClustersClientListResponse]) {

			page := armcontainerservice.ManagedClustersClientListResponse{
				ManagedClusterListResult: armcontainerservice.ManagedClusterListResult{
					Value: []*armcontainerservice.ManagedCluster{
						&cluster1,
					},
				},
			}
			resp.AddPage(http.StatusOK, page, nil)
			return resp
		},
	}
	cred := newTokenCredential()
	clusterClient, _ := armcontainerservice.NewManagedClustersClient("subscriptionID", cred, &arm.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: fake.NewManagedClustersServerTransport(&fakeMCServer),
		},
	})

	NewClusterClient = func(clientFactory *armcontainerservice.ClientFactory) *armcontainerservice.ManagedClustersClient {
		return clusterClient
	}
	defer func() {
		NewClusterClient = func(clientFactory *armcontainerservice.ClientFactory) *armcontainerservice.ManagedClustersClient {
			return clientFactory.NewManagedClustersClient()
		}
	}()

	logp.TestingSetup()

	server := initAzureTestServer()
	defer server.Close()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"host": server.Listener.Addr().String(),
	})
	if err != nil {
		t.Fatal(err)
	}

	p, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	actual, err := p.Run(&beat.Event{Fields: mapstr.M{}})
	if err != nil {
		t.Fatal(err)
	}

	expected := mapstr.M{
		"cloud": mapstr.M{
			"provider": "azure",
			"instance": mapstr.M{
				"id":   "220e2b43-0913-492f-ada0-b6b2795bdb9b",
				"name": "aks-agentpool-12628255-vmss_2",
			},
			"machine": mapstr.M{
				"type": "Standard_DS2_v2",
			},
			"account": mapstr.M{
				"id": "0e073ec1-c22f-4488-adde-da35ed609ccd",
			},
			"service": mapstr.M{
				"name": "Virtual Machines",
			},
			"resource_group": mapstr.M{
				"name": "MC_myname_group_myname_eastus",
			},
			"region": "eastus",
		},
		"orchestrator": mapstr.M{
			"cluster": mapstr.M{
				"id":   "testcluster1Id",
				"name": "testcluster1Name",
			},
		},
	}
	assert.Equal(t, expected, actual.Fields)
}
