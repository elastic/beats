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
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

func createS3API(ctx context.Context, config config, awsConfig awssdk.Config) (*awsS3API, error) {
	s3Client := s3.NewFromConfig(awsConfig, config.s3ConfigModifier)
	regionName, err := getRegionForBucket(ctx, s3Client, config.getBucketName())
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS region for bucket: %w", err)
	}
	// Can this really happen?
	if regionName != awsConfig.Region {
		awsConfig.Region = regionName
		s3Client = s3.NewFromConfig(awsConfig, config.s3ConfigModifier)
	}

	return &awsS3API{
		client: s3Client,
	}, nil
}

func createPipelineClient(pipeline beat.Pipeline) (beat.Client, error) {
	return pipeline.ConnectWith(beat.ClientConfig{
		EventListener: awscommon.NewEventACKHandler(),
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: boolPtr(false),
		},
	})
}

func getRegionForBucket(ctx context.Context, s3Client *s3.Client, bucketName string) (string, error) {
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

func getBucketNameFromARN(bucketARN string) string {
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

type nonAWSBucketResolver struct {
	endpoint string
}

func (n nonAWSBucketResolver) ResolveEndpoint(region string, options s3.EndpointResolverOptions) (awssdk.Endpoint, error) {
	return awssdk.Endpoint{URL: n.endpoint, SigningRegion: region, HostnameImmutable: true, Source: awssdk.EndpointSourceCustom}, nil
}
