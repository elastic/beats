// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/elastic/beats/libbeat/logp"
)

type opEnsureBucket struct {
	log        *logp.Logger
	svc        *s3.S3
	bucketName string
}

func newOpEnsureBucket(log *logp.Logger, cfg aws.Config, bucketName string) *opEnsureBucket {
	return &opEnsureBucket{log: log, svc: s3.New(cfg), bucketName: bucketName}
}

func (o *opEnsureBucket) Execute() error {
	o.log.Debugf("Creating S3 bucket: %s", o.bucketName)

	check := &s3.HeadBucketInput{Bucket: aws.String(o.bucketName)}
	reqCheck := o.svc.HeadBucketRequest(check)
	_, err := reqCheck.Send()
	// bucket do not exist lets create it.
	if err != nil {
		input := &s3.CreateBucketInput{Bucket: aws.String(o.bucketName)}
		req := o.svc.CreateBucketRequest(input)
		resp, err := req.Send()
		if err != nil {
			o.log.Debugf("Could not create bucket, resp: %v", resp)
			return err
		}
	}

	return nil
}
