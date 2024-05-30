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
	"os"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	// Disable IMDS when the real AWS SDK IMDS client is used,
	// so tests are isolated from the environment. Otherwise,
	// tests for non-EC2 providers may fail when the tests are
	// run within an EC2 VM.
	os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
}

type MockIMDSClient struct {
	GetInstanceIdentityDocumentFunc func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error)
}

func (m *MockIMDSClient) GetInstanceIdentityDocument(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
	return m.GetInstanceIdentityDocumentFunc(ctx, params, optFns...)
}

type MockEC2Client struct {
	DescribeTagsFunc func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error)
}

func (e *MockEC2Client) DescribeTags(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error) {
	return e.DescribeTagsFunc(ctx, params, optFns...)
}

func TestMain(m *testing.M) {
	logp.TestingSetup()
	code := m.Run()
	os.Exit(code)
}

func TestRetrieveAWSMetadataEC2(t *testing.T) {
	var (
		// not the best way to use a response template
		// but this should serve until we need to test
		// documents containing very different values
		accountIDDoc1        = "111111111111111"
		regionDoc1           = "us-east-1"
		availabilityZoneDoc1 = "us-east-1c"
		imageIDDoc1          = "ami-abcd1234"
		instanceTypeDoc1     = "t2.medium"
		instanceIDDoc2       = "i-22222222"
		clusterNameKey       = "eks:cluster-name"
		clusterNameValue     = "test"
		instanceIDDoc1       = "i-11111111"
	)

	var tests = []struct {
		testName                string
		mockGetInstanceIdentity func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error)
		mockEc2Tags             func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error)
		processorOverwrite      bool
		previousEvent           mapstr.M
		expectedEvent           mapstr.M
	}{
		{
			testName: "valid instance identity document, no cluster tags",
			mockGetInstanceIdentity: func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
				return &imds.GetInstanceIdentityDocumentOutput{
					InstanceIdentityDocument: imds.InstanceIdentityDocument{
						AvailabilityZone: availabilityZoneDoc1,
						Region:           regionDoc1,
						InstanceID:       instanceIDDoc1,
						InstanceType:     instanceTypeDoc1,
						AccountID:        accountIDDoc1,
						ImageID:          imageIDDoc1,
					},
				}, nil
			},
			mockEc2Tags: func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error) {
				return &ec2.DescribeTagsOutput{
					Tags: []types.TagDescription{},
				}, nil
			},
			processorOverwrite: false,
			previousEvent:      mapstr.M{},
			expectedEvent: mapstr.M{
				"cloud": mapstr.M{
					"provider":          "aws",
					"account":           mapstr.M{"id": accountIDDoc1},
					"instance":          mapstr.M{"id": instanceIDDoc1},
					"machine":           mapstr.M{"type": instanceTypeDoc1},
					"image":             mapstr.M{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
					"service":           mapstr.M{"name": "EC2"},
				},
			},
		},
		{
			testName: "all fields from processor",
			mockGetInstanceIdentity: func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
				return &imds.GetInstanceIdentityDocumentOutput{
					InstanceIdentityDocument: imds.InstanceIdentityDocument{
						AvailabilityZone: availabilityZoneDoc1,
						Region:           regionDoc1,
						InstanceID:       instanceIDDoc1,
						InstanceType:     instanceTypeDoc1,
						AccountID:        accountIDDoc1,
						ImageID:          imageIDDoc1,
					},
				}, nil
			},
			mockEc2Tags: func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error) {
				return &ec2.DescribeTagsOutput{
					Tags: []types.TagDescription{
						{
							Key:          &clusterNameKey,
							ResourceId:   &instanceIDDoc1,
							ResourceType: "instance",
							Value:        &clusterNameValue,
						},
					},
				}, nil
			},
			processorOverwrite: false,
			previousEvent:      mapstr.M{},
			expectedEvent: mapstr.M{
				"cloud": mapstr.M{
					"provider":          "aws",
					"account":           mapstr.M{"id": accountIDDoc1},
					"instance":          mapstr.M{"id": instanceIDDoc1},
					"machine":           mapstr.M{"type": instanceTypeDoc1},
					"image":             mapstr.M{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
					"service":           mapstr.M{"name": "EC2"},
				},
				"orchestrator": mapstr.M{
					"cluster": mapstr.M{
						"name": clusterNameValue,
						"id":   fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", regionDoc1, accountIDDoc1, clusterNameValue),
					},
				},
			},
		},
		{
			testName: "instanceId pre-informed, no overwrite",
			mockGetInstanceIdentity: func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
				return &imds.GetInstanceIdentityDocumentOutput{
					InstanceIdentityDocument: imds.InstanceIdentityDocument{
						AvailabilityZone: availabilityZoneDoc1,
						Region:           regionDoc1,
						InstanceID:       instanceIDDoc1,
						InstanceType:     instanceTypeDoc1,
						AccountID:        accountIDDoc1,
						ImageID:          imageIDDoc1,
					},
				}, nil
			},
			mockEc2Tags: func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error) {
				return &ec2.DescribeTagsOutput{
					Tags: []types.TagDescription{
						{
							Key:          &clusterNameKey,
							ResourceId:   &instanceIDDoc1,
							ResourceType: "instance",
							Value:        &clusterNameValue,
						},
					},
				}, nil
			},
			processorOverwrite: false,
			previousEvent: mapstr.M{
				"cloud": mapstr.M{
					"instance": mapstr.M{"id": instanceIDDoc2},
				},
			},
			expectedEvent: mapstr.M{
				"cloud": mapstr.M{
					"instance": mapstr.M{"id": instanceIDDoc2},
				},
				"orchestrator": mapstr.M{
					"cluster": mapstr.M{
						"name": clusterNameValue,
						"id":   fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", regionDoc1, accountIDDoc1, clusterNameValue),
					},
				},
			},
		},
		{
			// NOTE: In this case, add_cloud_metadata will overwrite cloud fields because
			// it won't detect cloud.provider as a cloud field. This is not the behavior we
			// expect and will find a better solution later in issue 11697.
			testName: "only cloud.provider pre-informed, no overwrite",
			mockGetInstanceIdentity: func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
				return &imds.GetInstanceIdentityDocumentOutput{
					InstanceIdentityDocument: imds.InstanceIdentityDocument{
						AvailabilityZone: availabilityZoneDoc1,
						Region:           regionDoc1,
						InstanceID:       instanceIDDoc1,
						InstanceType:     instanceTypeDoc1,
						AccountID:        accountIDDoc1,
						ImageID:          imageIDDoc1,
					},
				}, nil
			},
			mockEc2Tags: func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error) {
				return &ec2.DescribeTagsOutput{
					Tags: []types.TagDescription{
						{
							Key:          &clusterNameKey,
							ResourceId:   &instanceIDDoc1,
							ResourceType: "instance",
							Value:        &clusterNameValue,
						},
					},
				}, nil
			},
			processorOverwrite: false,
			previousEvent: mapstr.M{
				"cloud.provider": "aws",
			},
			expectedEvent: mapstr.M{
				"cloud.provider": "aws",
				"cloud": mapstr.M{
					"provider":          "aws",
					"account":           mapstr.M{"id": accountIDDoc1},
					"instance":          mapstr.M{"id": instanceIDDoc1},
					"machine":           mapstr.M{"type": instanceTypeDoc1},
					"image":             mapstr.M{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
					"service":           mapstr.M{"name": "EC2"},
				},
				"orchestrator": mapstr.M{
					"cluster": mapstr.M{
						"name": clusterNameValue,
						"id":   fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", regionDoc1, accountIDDoc1, clusterNameValue),
					},
				},
			},
		},
		{
			testName: "instanceId pre-informed, overwrite",
			mockGetInstanceIdentity: func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
				return &imds.GetInstanceIdentityDocumentOutput{
					InstanceIdentityDocument: imds.InstanceIdentityDocument{
						AvailabilityZone: availabilityZoneDoc1,
						Region:           regionDoc1,
						InstanceID:       instanceIDDoc1,
						InstanceType:     instanceTypeDoc1,
						AccountID:        accountIDDoc1,
						ImageID:          imageIDDoc1,
					},
				}, nil
			},
			mockEc2Tags: func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error) {
				return &ec2.DescribeTagsOutput{
					Tags: []types.TagDescription{},
				}, nil
			},
			processorOverwrite: true,
			previousEvent: mapstr.M{
				"cloud": mapstr.M{
					"instance": mapstr.M{"id": instanceIDDoc2},
				},
			},
			expectedEvent: mapstr.M{
				"cloud": mapstr.M{
					"provider":          "aws",
					"account":           mapstr.M{"id": accountIDDoc1},
					"instance":          mapstr.M{"id": instanceIDDoc1},
					"machine":           mapstr.M{"type": instanceTypeDoc1},
					"image":             mapstr.M{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
					"service":           mapstr.M{"name": "EC2"},
				},
			},
		},
		{
			testName: "only cloud.provider pre-informed, overwrite",
			mockGetInstanceIdentity: func(ctx context.Context, params *imds.GetInstanceIdentityDocumentInput, optFns ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
				return &imds.GetInstanceIdentityDocumentOutput{
					InstanceIdentityDocument: imds.InstanceIdentityDocument{
						AvailabilityZone: availabilityZoneDoc1,
						Region:           regionDoc1,
						InstanceID:       instanceIDDoc1,
						InstanceType:     instanceTypeDoc1,
						AccountID:        accountIDDoc1,
						ImageID:          imageIDDoc1,
					},
				}, nil
			},
			mockEc2Tags: func(ctx context.Context, params *ec2.DescribeTagsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeTagsOutput, error) {
				return &ec2.DescribeTagsOutput{
					Tags: []types.TagDescription{},
				}, nil
			},
			processorOverwrite: true,
			previousEvent: mapstr.M{
				"cloud.provider": "aws",
			},
			expectedEvent: mapstr.M{
				"cloud.provider": "aws",
				"cloud": mapstr.M{
					"provider":          "aws",
					"account":           mapstr.M{"id": accountIDDoc1},
					"instance":          mapstr.M{"id": instanceIDDoc1},
					"machine":           mapstr.M{"type": instanceTypeDoc1},
					"image":             mapstr.M{"id": imageIDDoc1},
					"region":            regionDoc1,
					"availability_zone": availabilityZoneDoc1,
					"service":           mapstr.M{"name": "EC2"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.testName, func(t *testing.T) {

			NewIMDSClient = func(cfg awssdk.Config) IMDSClient {
				return &MockIMDSClient{
					GetInstanceIdentityDocumentFunc: tc.mockGetInstanceIdentity,
				}
			}
			defer func() { NewIMDSClient = func(cfg awssdk.Config) IMDSClient { return imds.NewFromConfig(cfg) } }()

			NewEC2Client = func(cfg awssdk.Config) EC2Client {
				return &MockEC2Client{
					DescribeTagsFunc: tc.mockEc2Tags,
				}
			}
			defer func() { NewEC2Client = func(cfg awssdk.Config) EC2Client { return ec2.NewFromConfig(cfg) } }()

			config, err := conf.NewConfigFrom(map[string]interface{}{
				"overwrite": tc.processorOverwrite,
				"providers": []string{"aws"},
			})
			if err != nil {
				t.Fatalf("error creating config from map: %s", err.Error())
			}
			cmp, err := New(config)
			if err != nil {
				t.Fatalf("error creating new metadata processor: %s", err.Error())
			}

			actual, err := cmp.Run(&beat.Event{Fields: tc.previousEvent})
			if err != nil {
				t.Fatalf("error running processor: %s", err.Error())
			}
			assert.Equal(t, tc.expectedEvent, actual.Fields)
		})
	}
}
