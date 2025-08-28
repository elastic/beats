// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestAWSS3API_clientFor(t *testing.T) {
	// When SQS notifications do not contain a region (like Crowdstrike FDR's
	// custom notifications), then the default pre-made S3 client should be used.
	t.Run("empty_region_uses_pre_made_client", func(t *testing.T) {
		want := s3.New(s3.Options{Region: "us-east-1"})
		api := newAWSs3API(want)
		got := api.clientFor("")

		if want != got {
			t.Errorf("Empty region should return the default premade client: want %p, got %p", want, got)
		}
	})
}
