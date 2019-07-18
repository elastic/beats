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

type opCreateFunction struct {
	log      *logp.Logger
	tokenSrc oauth2.TokenSource
	location string
	name     string
	function *cloudfunctions.CloudFunction
}

func newOpCreateFunction(
	log *logp.Logger,
	tokenSrc oauth2.TokenSource,
	location string,
	name string,
	f *cloudfunctions.CloudFunction,
) *opCreateFunction {
	return &opCreateFunction{
		log:      log,
		tokenSrc: tokenSrc,
		name:     name,
		location: location,
		function: f,
	}
}

// Execute creates a function from the zip uploaded.
func (o *opCreateFunction) Execute(ctx executor.Context) error {
	c, ok := ctx.(*functionContext)
	if !ok {
		return errWrongContext
	}

	o.log.Debugf("Creating function %s at %s", o.function.Name, o.function.SourceArchiveUrl)

	client := oauth2.NewClient(context.TODO(), o.tokenSrc)
	svc, err := cloudfunctions.New(client)
	if err != nil {
		return fmt.Errorf("error while creating cloud functions service: %+v", err)
	}

	functionSvc := cloudfunctions.NewProjectsLocationsFunctionsService(svc)
	operation, err := functionSvc.Create(o.location, o.function).Context(context.TODO()).Do()
	if err != nil {
		return fmt.Errorf("error while creating function: %+v", err)
	}

	c.name = &operation.Name

	if operation.Done {
		o.log.Debugf("Function %s created successfully", o.function.Name)
	} else {
		o.log.Debugf("Operation '%s' is in progress to create function %s", operation.Name, o.function.Name)
	}

	return nil
}

// Rollback removes the deployed function.
func (o *opCreateFunction) Rollback(ctx executor.Context) error {
	return newOpDeleteFunction(o.log, o.location, o.function.Name, o.tokenSrc).Execute(ctx)
}
