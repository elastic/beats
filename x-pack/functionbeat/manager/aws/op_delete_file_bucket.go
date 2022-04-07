// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/x-pack/functionbeat/manager/executor"
)

type opDeleteFileBucket struct {
	log        *logp.Logger
	svc        *s3.Client
	bucketName string
	path       string
}

func newOpDeleteFileBucket(
	log *logp.Logger,
	config aws.Config,
	bucketName, path string,
) *opDeleteFileBucket {
	return &opDeleteFileBucket{
		log:        log,
		svc:        s3.New(config),
		bucketName: bucketName,
		path:       path,
	}
}

func (o *opDeleteFileBucket) Execute(_ executor.Context) error {
	o.log.Debugf("Removing file '%s' on bucket '%s'", o.path, o.bucketName)
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(o.bucketName),
		Key:    aws.String(o.path),
	}

	req := o.svc.DeleteObjectRequest(input)
	resp, err := req.Send(context.TODO())

	if err != nil {
		o.log.Debugf("Could not remove object to S3, resp: %v", resp)
		return err
	}
	o.log.Debug("Remove successful")
	return nil
}
