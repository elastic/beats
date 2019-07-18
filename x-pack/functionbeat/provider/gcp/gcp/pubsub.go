// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/function/core"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/x-pack/functionbeat/provider/gcp/gcp/transformer"
)

// PubSub represents a Google Cloud function which reads event from Google Pub/Sub triggers.
type PubSub struct {
	log    *logp.Logger
	config *FunctionConfig
}

// PubSubMsg is an alias to string
type PubSubMsg string

// PubSubContext is an alias to string
type PubSubContext string

// NewPubSub returns a new function to read from Google Pub/Sub.
func NewPubSub(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := defaultPubSubFunctionConfig()
	err := cfg.Unpack(config)
	if err != nil {
		return &PubSub{}, err
	}

	return &PubSub{
		log:    logp.NewLogger("pubsub"),
		config: config,
	}, nil
}

// Run start
func (p *PubSub) Run(ctx context.Context, client core.Client) error {
	msgCtx, msg, err := p.getEventDataFromContext(ctx)
	if err != nil {
		return err
	}
	event, err := transformer.PubSub(msgCtx, msg)
	if err := client.Publish(event); err != nil {
		p.log.Errorf("error while publishing Pub/Sub event %+v", err)
		return err
	}
	client.Wait()

	return nil
}

func (p *PubSub) getEventDataFromContext(ctx context.Context) (context.Context, pubsub.Message, error) {
	iMsgCtx := ctx.Value(PubSubContext("pub_sub_context"))
	if iMsgCtx == nil {
		return nil, pubsub.Message{}, fmt.Errorf("no pub/sub message context")
	}
	msgCtx, ok := iMsgCtx.(context.Context)
	if !ok {
		return nil, pubsub.Message{}, fmt.Errorf("not message context: %+v", iMsgCtx)
	}

	iMsg := ctx.Value(PubSubMsg("pub_sub_message"))
	if iMsg == nil {
		return nil, pubsub.Message{}, fmt.Errorf("no pub/sub message")
	}
	msg, ok := iMsg.(pubsub.Message)
	if !ok {
		return nil, pubsub.Message{}, fmt.Errorf("not message: %+v", iMsg)
	}
	return msgCtx, msg, nil
}

// PubSubDetails returns the details of the feature.
func PubSubDetails() *feature.Details {
	return feature.NewDetails("Google Pub/Sub trigger", "receive messages from Google Pub/Sub.", feature.Stable)
}

// Name returns the name of the function.
func (p *PubSub) Name() string {
	return "pubsub"
}

// Config returns the configuration to use when creating the function.
func (p *PubSub) Config() *FunctionConfig {
	return p.config
}
