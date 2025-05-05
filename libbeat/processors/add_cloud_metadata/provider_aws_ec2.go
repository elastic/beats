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
	"io"
	"net/http"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/elastic/elastic-agent-libs/mapstr"

	conf "github.com/elastic/elastic-agent-libs/config"
)

const (
	eksClusterNameTagKey = "eks:cluster-name"
	tagsCategory         = "tags/instance"
	tagPrefix            = "aws.tags"
)

type IMDSClient interface {
	ec2rolecreds.GetMetadataAPIClient
	GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error)
}

var NewIMDSClient func(cfg awssdk.Config) IMDSClient = func(cfg awssdk.Config) IMDSClient {
	return imds.NewFromConfig(cfg)
}

type EC2Client interface {
	DescribeTags(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error)
}

var NewEC2Client func(cfg awssdk.Config) EC2Client = func(cfg awssdk.Config) EC2Client {
	return ec2.NewFromConfig(cfg)
}

// AWS EC2 Metadata Service
var ec2MetadataFetcher = provider{
	Name: "aws-ec2",

	DefaultEnabled: true,

	Create: func(_ string, config *conf.C) (metadataFetcher, error) {
		ec2Schema := func(m map[string]interface{}) mapstr.M {
			meta := mapstr.M{
				"cloud": mapstr.M{
					"service": mapstr.M{
						"name": "EC2",
					},
				},
			}

			meta.DeepUpdate(m)
			return meta
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

	// generate AWS specific client with overriding requirements
	var awsHTTPClient awshttp.BuildableClient
	awsHTTPClient = *awsHTTPClient.WithTimeout(client.Timeout).WithTransportOptions(func(tr *http.Transport) {
		transport, ok := client.Transport.(*http.Transport)
		if ok {
			tr.TLSClientConfig = transport.TLSClientConfig
		}

		tr.DisableKeepAlives = true
	})

	// LoadDefaultConfig loads the EC2 role credentials
	awsConfig, err := awscfg.LoadDefaultConfig(context.TODO(), awscfg.WithHTTPClient(&awsHTTPClient))
	if err != nil {
		result.err = fmt.Errorf("failed loading AWS default configuration: %w", err)
		return
	}

	imdsClient := NewIMDSClient(awsConfig)
	instanceIdentity, err := imdsClient.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		result.err = fmt.Errorf("failed fetching EC2 Identity Document: %w", err)
		return
	}

	awsRegion := instanceIdentity.Region
	accountID := instanceIdentity.AccountID
	instanceID := instanceIdentity.InstanceID

	_, _ = result.metadata.Put("cloud.instance.id", instanceIdentity.InstanceID)
	_, _ = result.metadata.Put("cloud.machine.type", instanceIdentity.InstanceType)
	_, _ = result.metadata.Put("cloud.region", awsRegion)
	_, _ = result.metadata.Put("cloud.availability_zone", instanceIdentity.AvailabilityZone)
	_, _ = result.metadata.Put("cloud.account.id", accountID)
	_, _ = result.metadata.Put("cloud.image.id", instanceIdentity.ImageID)

	// AWS Region must be set to be able to get EC2 Tags
	awsConfig.Region = awsRegion
	tags := getTags(ctx, imdsClient, NewEC2Client(awsConfig), instanceID, logger)

	if tags[eksClusterNameTagKey] != "" {
		// for AWS cluster ID is used cluster ARN: arn:partition:service:region:account-id:resource-type/resource-id, example:
		// arn:aws:eks:us-east-2:627286350134:cluster/cluster-name
		clusterARN := fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%v", awsRegion, accountID, tags[eksClusterNameTagKey])

		_, _ = result.metadata.Put("orchestrator.cluster.id", clusterARN)
		_, _ = result.metadata.Put("orchestrator.cluster.name", tags[eksClusterNameTagKey])
	}

	if len(tags) == 0 {
		return
	}

	logger.Infof("Adding retrieved tags with key: %s", tagPrefix)
	for k, v := range tags {
		_, _ = result.metadata.Put(fmt.Sprintf("%s.%s", tagPrefix, k), v)
	}
}

// getTags is a helper to extract EC2 tags. Internally it utilize multiple extraction methods.
func getTags(ctx context.Context, imdsClient IMDSClient, ec2Client EC2Client, instanceId string, logger *logp.Logger) map[string]string {
	logger.Info("Extracting EC2 tags from IMDS endpoint")
	tags, ok := getTagsFromIMDS(ctx, imdsClient, logger)
	if ok {
		return tags
	}

	logger.Info("Tag extraction from IMDS failed, fallback to DescribeTags API to obtain EKS cluster name.")
	clusterName, err := clusterNameFromDescribeTag(ctx, ec2Client, instanceId)
	if err != nil {
		logger.Warnf("error obtaining cluster name: %v.", err)
		return tags
	}

	if clusterName != "" {
		tags[eksClusterNameTagKey] = clusterName
	}
	return tags
}

// getTagsFromIMDS is a helper to extract EC2 tags using instance metadata service.
// Note that this call could get throttled and currently does not implement a retry mechanism.
// See - https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instancedata-data-retrieval.html#instancedata-throttling
func getTagsFromIMDS(ctx context.Context, client IMDSClient, logger *logp.Logger) (tags map[string]string, ok bool) {
	tags = make(map[string]string)

	b, err := getMetadataHelper(ctx, client, tagsCategory, logger)
	if err != nil {
		logger.Warnf("error obtaining tags category: %v", err)
		return tags, false
	}

	for _, tag := range strings.Split(string(b), "\n") {
		tagPath := fmt.Sprintf("%s/%s", tagsCategory, tag)
		b, err := getMetadataHelper(ctx, client, tagPath, logger)
		if err != nil {
			logger.Warnf("error extracting tag value of %s: %v", tag, err)
			return tags, false
		}

		tagValue := string(b)
		if tagValue == "" {
			logger.Infof("Ignoring tag key %s as value is empty", tag)
			continue
		}

		tags[tag] = tagValue
	}

	return tags, true
}

// getMetadataHelper performs the IMDS call for the given path and returns the response content after closing the underlying content reader.
func getMetadataHelper(ctx context.Context, client IMDSClient, path string, logger *logp.Logger) (content []byte, err error) {
	metadata, err := client.GetMetadata(ctx, &imds.GetMetadataInput{Path: path})
	if err != nil {
		return nil, fmt.Errorf("error from IMDS metadata request: %w", err)
	}

	defer func(Content io.ReadCloser) {
		err := Content.Close()
		if err != nil {
			logger.Warnf("error closing IMDS metadata response body: %v", err)
		}
	}(metadata.Content)

	content, err = io.ReadAll(metadata.Content)
	if err != nil {
		return nil, fmt.Errorf("error extracting metadata from the IMDS response: %w", err)
	}

	return content, nil
}

// clusterNameFromDescribeTag is a helper to extract EKS cluster name using DescribeTag.
func clusterNameFromDescribeTag(ctx context.Context, ec2Client EC2Client, instanceID string) (string, error) {
	input := &ec2.DescribeTagsInput{
		Filters: []types.Filter{
			{
				Name: awssdk.String("resource-id"),
				Values: []string{
					instanceID,
				},
			},
			{
				Name:   awssdk.String("key"),
				Values: []string{eksClusterNameTagKey},
			},
		},
	}

	tagsResult, err := ec2Client.DescribeTags(ctx, input)
	if err != nil {
		return "", fmt.Errorf("error fetching EC2 Tags: %w", err)
	}
	if len(tagsResult.Tags) == 1 {
		return *tagsResult.Tags[0].Value, nil
	}
	return "", nil
}
