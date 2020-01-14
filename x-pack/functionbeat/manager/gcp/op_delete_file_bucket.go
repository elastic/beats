// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/manager/executor"
)

type opDeleteFromBucket struct {
	log    *logp.Logger
	config *Config
	name   string
}

func newOpDeleteFromBucket(log *logp.Logger, config *Config, name string) *opDeleteFromBucket {
	return &opDeleteFromBucket{
		log:    log,
		config: config,
		name:   name,
	}
}

// Execute removes the function from the bucket.
// storage.objects.delete permission is required.
func (o *opDeleteFromBucket) Execute(_ executor.Context) error {
	o.log.Debugf("Removing file '%s' from bucket '%s'", o.name, o.config.FunctionStorage)

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create storage client: %+v", err)
	}

	err = client.Bucket(o.config.FunctionStorage).Object(o.name).Delete(ctx)
	if err != nil {
		return err
	}

	o.log.Debugf("Successfully removed function '%s' from bucket '%s'", o.name, o.config.FunctionStorage)
	return nil
}
