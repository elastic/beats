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

//TODO: adjust tests and delete consts:
const (
	ec2InstanceIdentityURI            = "/2014-02-25/dynamic/instance-identity/document"
	ec2InstanceIMDSv2TokenValueHeader = "X-aws-ec2-metadata-token"
	ec2InstanceIMDSv2TokenTTLHeader   = "X-aws-ec2-metadata-token-ttl-seconds"
	ec2InstanceIMDSv2TokenTTLValue    = "21600"
	ec2InstanceIMDSv2TokenURI         = "/latest/api/token"
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
	// config := defaultConfig()
	// if err := c.Unpack(&config); err != nil {
	// 	logger.Warnf("error when load config for getting IMDSv2 token: %s. No token in the metadata request will be used.", err)
	// }

	// LoadDefaultConfig loads the Ec2 role credentials
	awsConfig, err := awscfg.LoadDefaultConfig(context.TODO(), awscfg.WithHTTPClient(&client))
	if err != nil {
		logger.Warnf("error when loading AWS default configuration: %s.", err)
	}

	awsClient := imds.NewFromConfig(awsConfig)

	instanceIdentity, err := awsClient.GetInstanceIdentityDocument(context.TODO(), &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		logger.Warnf("error when fetching EC2 Identity Document: %s.", err)
	}

	// Region must be set to be able to get EC2 Tags
	awsConfig.Region = instanceIdentity.Region

	svc := ec2.NewFromConfig(awsConfig)
	input := &ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name: awssdk.String("resource-id"),
				Values: []string{
					*awssdk.String(instanceIdentity.InstanceIdentityDocument.InstanceID),
				},
			},
			{
				Name: awssdk.String("key"),
				Values: []string{
					*awssdk.String("eks:cluster-name"),
				},
			},
		},
	}

	tagsResult, err := svc.DescribeTags(context.TODO(), input)
	if err != nil {
		logger.Warnf("error when fetching EC2 Tags: %s.", err)
	}

	result.metadata.Put("cloud.orchestrator.cluster.name", tagsResult.Tags[0].Value)
	result.metadata.Put("cloud.instance.id", instanceIdentity.InstanceIdentityDocument.InstanceID)
	result.metadata.Put("cloud.machine.type", instanceIdentity.InstanceIdentityDocument.InstanceType)
	result.metadata.Put("cloud.region", instanceIdentity.InstanceIdentityDocument.Region)
	result.metadata.Put("cloud.availability_zone", instanceIdentity.InstanceIdentityDocument.AvailabilityZone)
	result.metadata.Put("cloud.account.id", instanceIdentity.InstanceIdentityDocument.AccountID)
	result.metadata.Put("cloud.image.id", instanceIdentity.InstanceIdentityDocument.ImageID)

}
