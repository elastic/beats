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

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	conf "github.com/elastic/elastic-agent-libs/config"
)

// AWS EC2 Metadata Service
var ec2MetadataFetcher = provider{
	Name: "aws-ec2",

	Local: true,

	Create: func(_ string, config *conf.C) (metadataFetcher, error) {
		ec2Schema := func(m map[string]interface{}) mapstr.M {
			m["service"] = mapstr.M{
				"name": "EC2",
			}
			return mapstr.M{"cloud": m}
		}

		fetcher, err := newGenericMetadataFetcher(config, "aws", ec2Schema, fetchRawProviderMetadata)
		return fetcher, err
	},
}

// fetchRaw queries raw metadata from a hosting provider's metadata service.
func fetchRawProviderMetadata(
	ctx context.Context,
	client http.Client,
	result *result,
) {
	logger := logp.NewLogger("add_cloud_metadata")

	// LoadDefaultConfig loads the EC2 role credentials
	awsConfig, err := awscfg.LoadDefaultConfig(context.TODO(), awscfg.WithHTTPClient(&client))
	if err != nil {
		logger.Debugf("error loading AWS default configuration: %s.", err)
		return
	}

	awsClient := imds.NewFromConfig(awsConfig)

	instanceIdentity, err := awsClient.GetInstanceIdentityDocument(context.TODO(), &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		logger.Warnf("error fetching EC2 Identity Document: %s.", err)
	}

	// AWS Region must be set to be able to get EC2 Tags
	awsRegion := instanceIdentity.InstanceIdentityDocument.Region
	awsConfig.Region = awsRegion

	clusterName, err := fetchEC2ClusterNameTag(awsConfig, instanceIdentity.InstanceIdentityDocument.InstanceID)
	if err != nil {
		logger.Debugf("error fetching cluster name metadata: %s.", err)
	}

	accountID := instanceIdentity.InstanceIdentityDocument.AccountID

	// for AWS cluster ID is used cluster ARN: arn:partition:service:region:account-id:resource-type/resource-id, example:
	// arn:aws:eks:us-east-2:627286350134:cluster/cluster-name
	if *clusterName != "" {
		clusterARN := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", awsRegion, accountID, *clusterName)

		result.metadata.Put("cloud.orchestrator.cluster.name", clusterName)
		result.metadata.Put("cloud.orchestrator.cluster.id", clusterARN)
	}

	result.metadata.Put("cloud.instance.id", instanceIdentity.InstanceIdentityDocument.InstanceID)
	result.metadata.Put("cloud.machine.type", instanceIdentity.InstanceIdentityDocument.InstanceType)
	result.metadata.Put("cloud.region", awsRegion)
	result.metadata.Put("cloud.availability_zone", instanceIdentity.InstanceIdentityDocument.AvailabilityZone)
	result.metadata.Put("cloud.account.id", accountID)
	result.metadata.Put("cloud.image.id", instanceIdentity.InstanceIdentityDocument.ImageID)
}

func fetchEC2ClusterNameTag(awsConfig awssdk.Config, instanceID string) (*string, error) {
	svc := ec2.NewFromConfig(awsConfig)
	input := &ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name: awssdk.String("resource-id"),
				Values: []string{
					instanceID,
				},
			},
			{
				Name: awssdk.String("key"),
				Values: []string{
					"eks:cluster-name",
				},
			},
		},
	}

	tagsResult, err := svc.DescribeTags(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("error fetching EC2 Tags: %s", err)
	}
	return tagsResult.Tags[0].Value, nil
}
