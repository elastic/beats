// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func (in *s3PollerInput) createS3API(ctx context.Context) (*awsS3API, error) {
	s3Client := s3.NewFromConfig(in.awsConfig, in.config.s3ConfigModifier)
	regionName, err := getRegionForBucket(ctx, s3Client, in.config.getBucketName())
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS region for bucket: %w", err)
	}
	// Can this really happen?
	if regionName != in.awsConfig.Region {
		in.awsConfig.Region = regionName
		s3Client = s3.NewFromConfig(in.awsConfig, in.config.s3ConfigModifier)
	}

	return newAWSs3API(s3Client), nil
}

func createPipelineClient(pipeline beat.Pipeline, acks *awsACKHandler) (beat.Client, error) {
	return pipeline.ConnectWith(beat.ClientConfig{
		EventListener: acks.pipelineEventListener(),
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: boolPtr(false),
		},
	})
}

func getRegionForBucket(ctx context.Context, s3Client *s3.Client, bucketName string) (string, error) {
	// Skip region fetching if it's an Access Point ARN
	if isValidAccessPointARN(bucketName) {
		// Extract the region from the ARN (e.g., arn:aws:s3:us-west-2:123456789012:accesspoint/my-access-point)
		return getRegionFromAccessPointARN(bucketName), nil
	}

	getBucketLocationOutput, err := s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: awssdk.String(bucketName),
	})

	if err != nil {
		return "", err
	}

	// Region us-east-1 have a LocationConstraint of null.
	if len(getBucketLocationOutput.LocationConstraint) == 0 {
		return "us-east-1", nil
	}

	return string(getBucketLocationOutput.LocationConstraint), nil
}

// Helper function to extract region from Access Point ARN
func getRegionFromAccessPointARN(arn string) string {
	arnParts := strings.Split(arn, ":")
	if len(arnParts) > 3 {
		return arnParts[3] // The fourth part of ARN is region
	}
	return ""
}

func getBucketNameFromARN(bucketARN string) string {
	if isValidAccessPointARN(bucketARN) {
		return bucketARN // Return full ARN for Access Points
	}
	bucketMetadata := strings.Split(bucketARN, ":")
	bucketName := bucketMetadata[len(bucketMetadata)-1]
	return bucketName
}

func getProviderFromDomain(endpoint string, ProviderOverride string) string {
	if ProviderOverride != "" {
		return ProviderOverride
	}
	if endpoint == "" {
		return "aws"
	}
	// List of popular S3 SaaS providers
	providers := map[string]string{
		"amazonaws.com":          "aws",
		"c2s.sgov.gov":           "aws",
		"c2s.ic.gov":             "aws",
		"amazonaws.com.cn":       "aws",
		"backblazeb2.com":        "backblaze",
		"cloudflarestorage.com":  "cloudflare",
		"wasabisys.com":          "wasabi",
		"digitaloceanspaces.com": "digitalocean",
		"dream.io":               "dreamhost",
		"scw.cloud":              "scaleway",
		"googleapis.com":         "gcp",
		"cloud.it":               "arubacloud",
		"linodeobjects.com":      "linode",
		"vultrobjects.com":       "vultr",
		"appdomain.cloud":        "ibm",
		"aliyuncs.com":           "alibaba",
		"oraclecloud.com":        "oracle",
		"exo.io":                 "exoscale",
		"upcloudobjects.com":     "upcloud",
		"ilandcloud.com":         "iland",
		"zadarazios.com":         "zadara",
	}

	parsedEndpoint, _ := url.Parse(endpoint)
	for key, provider := range providers {
		// support endpoint with and without scheme (http(s)://abc.xyz, abc.xyz)
		constraint := parsedEndpoint.Hostname()
		if len(parsedEndpoint.Scheme) == 0 {
			constraint = parsedEndpoint.Path
		}
		if strings.HasSuffix(constraint, key) {
			return provider
		}
	}
	return "unknown"
}
