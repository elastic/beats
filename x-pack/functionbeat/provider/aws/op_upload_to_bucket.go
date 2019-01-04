// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"bytes"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/elastic/beats/libbeat/logp"
)

type opUploadToBucket struct {
	log        *logp.Logger
	svc        *s3.S3
	bucketName string
	path       string
	raw        []byte
	config     aws.Config
}

func newOpUploadToBucket(
	log *logp.Logger,
	config aws.Config,
	bucketName, path string,
	raw []byte,
) *opUploadToBucket {
	return &opUploadToBucket{
		log:        log,
		svc:        s3.New(config),
		bucketName: bucketName,
		path:       path,
		raw:        raw,
		config:     config,
	}
}

func (o *opUploadToBucket) Execute(_ executionContext) error {
	o.log.Debugf("Uploading file '%s' to bucket '%s' with size %d bytes", o.path, o.bucketName, len(o.raw))
	input := &s3.PutObjectInput{
		Bucket: aws.String(o.bucketName),
		Body:   bytes.NewReader(o.raw),
		Key:    aws.String(o.path),
	}
	req := o.svc.PutObjectRequest(input)
	resp, err := req.Send()

	if err != nil {
		o.log.Debugf("Could not upload object to S3, resp: %v", resp)
		return err
	}
	o.log.Debug("Upload successful")
	return nil
}

func (o *opUploadToBucket) Rollback(ctx executionContext) error {
	// The error will be logged but we do not enforce a hard failure because the file could have
	// been removed before.
	err := newOpDeleteFileBucket(o.log, o.config, o.bucketName, o.path).Execute(ctx)
	if err != nil {
		o.log.Debugf("Fail to delete file on bucket, error: %+v", err)
	}
	return nil
}
