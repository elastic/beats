// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/functionbeat/manager/executor"
)

type opUploadToBucket struct {
	log    *logp.Logger
	config *Config
	name   string
	raw    []byte
}

func newOpUploadToBucket(log *logp.Logger, config *Config, name string, raw []byte) *opUploadToBucket {
	return &opUploadToBucket{
		log:    log,
		config: config,
		name:   name,
		raw:    raw,
	}
}

// Execute loads function to bucket.
// If function needs to be overwritten, storage.objects.delete permission is required.
func (o *opUploadToBucket) Execute(_ executor.Context) error {
	o.log.Debugf("Uploading file '%s' to bucket '%s' with size %d bytes", o.name, o.config.FunctionStorage, len(o.raw))

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create storage client: %+v", err)
	}

	w := client.Bucket(o.config.FunctionStorage).Object(o.name).NewWriter(ctx)
	w.ContentType = "text/plain"
	_, err = w.Write(o.raw)
	if err != nil {
		return fmt.Errorf("error while writing function: %+v", err)
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("error while closing writer: %+v", err)
	}

	o.log.Debugf("Upload to bucket was successful")

	return nil
}

// Rollback removes the loaded archive.
func (o *opUploadToBucket) Rollback(ctx executor.Context) error {
	err := newOpDeleteFromBucket(o.log, o.config, o.name).Execute(ctx)
	if err != nil {
		o.log.Debugf("Fail to delete file from bucket, error: %+v", err)
	}
	return nil
}
