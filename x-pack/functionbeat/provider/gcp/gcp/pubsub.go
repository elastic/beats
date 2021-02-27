// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/functions/metadata"
	"cloud.google.com/go/pubsub"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/core"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/v7/x-pack/functionbeat/function/telemetry"
)

const (
	pubSubEventCtxStr = "pub_sub_event"
)

// PubSub represents a Google Cloud function which reads event from Google Pub/Sub triggers.
type PubSub struct {
	log    *logp.Logger
	config *FunctionConfig
}

// PubSubEventKey is an alias to string
type PubSubEventKey string

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

// PubSubEvent stores the context and the message from Google Pub/Sub.
type PubSubEvent struct {
	Metadata *metadata.Metadata
	Message  pubsub.Message
}

// NewPubSubContext creates a context from context and message returned from Google Pub/Sub.
func NewPubSubContext(beatCtx, ctx context.Context, m pubsub.Message) (context.Context, error) {
	data, err := metadata.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	e := PubSubEvent{
		Metadata: data,
		Message:  m,
	}

	return context.WithValue(beatCtx, PubSubEventKey(pubSubEventCtxStr), e), nil
}

// Run start
func (p *PubSub) Run(ctx context.Context, client core.Client, t telemetry.T) error {
	t.AddTriggeredFunction()

	pubsubEvent, err := p.getEventDataFromContext(ctx)
	if err != nil {
		return err
	}
	event, err := transformPubSub(pubsubEvent.Metadata, pubsubEvent.Message)
	if err := client.Publish(event); err != nil {
		p.log.Errorf("error while publishing Pub/Sub event %+v", err)
		return err
	}
	client.Wait()

	return nil
}

func (p *PubSub) getEventDataFromContext(ctx context.Context) (PubSubEvent, error) {
	iPubSubEvent := ctx.Value(PubSubEventKey(pubSubEventCtxStr))
	if iPubSubEvent == nil {
		return PubSubEvent{}, fmt.Errorf("no pub_sub_event in context")
	}
	event, ok := iPubSubEvent.(PubSubEvent)
	if !ok {
		return PubSubEvent{}, fmt.Errorf("not PubSubEvent: %+v", iPubSubEvent)
	}

	return event, nil
}

// PubSubDetails returns the details of the feature.
func PubSubDetails() feature.Details {
	return feature.MakeDetails("Google Pub/Sub trigger", "receive messages from Google Pub/Sub.", feature.Stable)
}

// Name returns the name of the function.
func (p *PubSub) Name() string {
	return "pubsub"
}

// Config returns the configuration to use when creating the function.
func (p *PubSub) Config() *FunctionConfig {
	return p.config
}
