// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	cloudfunctions "google.golang.org/api/cloudfunctions/v1"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/manager/executor"
)

type opUpdateFunction struct {
	log      *logp.Logger
	tokenSrc oauth2.TokenSource
	name     string
	function *cloudfunctions.CloudFunction
}

func newOpUpdateFunction(
	log *logp.Logger,
	tokenSrc oauth2.TokenSource,
	name string,
	f *cloudfunctions.CloudFunction,
) *opUpdateFunction {
	return &opUpdateFunction{
		log:      log,
		tokenSrc: tokenSrc,
		name:     name,
		function: f,
	}
}

// Execute updates an existing function.
func (o *opUpdateFunction) Execute(ctx executor.Context) error {
	c, ok := ctx.(*functionContext)
	if !ok {
		return errWrongContext
	}

	o.log.Debugf("Updating function %s at %s", o.function.Name, o.function.SourceArchiveUrl)

	client := oauth2.NewClient(context.TODO(), o.tokenSrc)
	svc, err := cloudfunctions.New(client)
	if err != nil {
		return fmt.Errorf("error while creating cloud functions service: %+v", err)
	}

	functionSvc := cloudfunctions.NewProjectsLocationsFunctionsService(svc)
	operation, err := functionSvc.Patch(o.name, o.function).Context(context.TODO()).Do()
	if err != nil {
		return fmt.Errorf("error while updating function: %+v", err)
	}

	c.name = &operation.Name

	if operation.Done {
		o.log.Debugf("Function %s updated successfully", o.function.Name)
	} else {
		o.log.Debugf("Operation '%s' is in progress to update function %s", operation.Name, o.function.Name)
	}

	return nil
}

// Rollback updates the deployed function.
func (o *opUpdateFunction) Rollback(_ executor.Context) error {
	return nil
}
