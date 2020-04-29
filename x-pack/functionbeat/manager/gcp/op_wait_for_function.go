// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	cloudfunctions "google.golang.org/api/cloudfunctions/v1"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/functionbeat/manager/executor"
)

var periodicCheck = 5 * time.Second

type opWaitForFunction struct {
	ctx      *functionContext
	log      *logp.Logger
	tokenSrc oauth2.TokenSource
}

func newOpWaitForFunction(ctx *functionContext, log *logp.Logger, tokenSrc oauth2.TokenSource) *opWaitForFunction {
	return &opWaitForFunction{
		ctx:      ctx,
		log:      log,
		tokenSrc: tokenSrc,
	}
}

func (o *opWaitForFunction) Execute(_ executor.Context) error {
	if o.ctx.name == "" {
		return errMissingFunctionName
	}

	client := oauth2.NewClient(context.TODO(), o.tokenSrc)
	svc, err := cloudfunctions.New(client)
	if err != nil {
		return fmt.Errorf("error while creating cloud functions service: %+v", err)
	}

	opSvc := cloudfunctions.NewOperationsService(svc)
	for {
		op, err := opSvc.Get(o.ctx.name).Context(context.Background()).Do()
		if err != nil {
			return err
		}

		if op.Done {
			if op.Error != nil {
				return fmt.Errorf("error while creating function (code: %d):\n%s", op.Error.Code, op.Error.Message)
			}
			o.log.Debugf("Successfully deployed function")
			return nil
		}

		o.log.Debugf("Operation %s has not finished yet. Retrying in 5 seconds.", o.ctx.name)

		timer := time.NewTimer(periodicCheck)
		<-timer.C
	}
}
