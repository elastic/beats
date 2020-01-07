// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"context"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/function/core"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
	"github.com/elastic/beats/x-pack/functionbeat/provider/gcp/gcp/transformer"
)

const (
	storageEvtCtxStr = "storage_event"
)

// Storage represents a Google Cloud function which reads event from Google Cloud Storage.
type Storage struct {
	log    *logp.Logger
	config *FunctionConfig
}

// StorageEvent is an alias to string
type StorageEventKey string

type StorageEvent struct {
	Ctx   context.Context
	Event transformer.StorageEvent
}

// NewStorage returns a new function to read from Google Cloud Storage.
func NewStorage(provider provider.Provider, cfg *common.Config) (provider.Function, error) {
	config := defaultStorageFunctionConfig()
	err := cfg.Unpack(config)
	if err != nil {
		return &Storage{}, err
	}

	return &Storage{
		log:    logp.NewLogger("storage"),
		config: config,
	}, nil
}

// NewStorageContext creates a context from context and message returned from Google Cloud Storage.
func NewStorageContext(beatCtx, ctx context.Context, e transformer.StorageEvent) context.Context {
	evt := StorageEvent{
		Ctx:   ctx,
		Event: e,
	}

	return context.WithValue(beatCtx, StorageEventKey(storageEvtCtxStr), evt)
}

// Run start
func (s *Storage) Run(ctx context.Context, client core.Client) error {
	evt, err := s.getEventDataFromContext(ctx)
	if err != nil {
		return err
	}
	event, err := transformer.Storage(evt.Ctx, evt.Event)
	if err := client.Publish(event); err != nil {
		s.log.Errorf("error while publishing Google Cloud Storage event %+v", err)
		return err
	}
	client.Wait()

	return nil
}

func (s *Storage) getEventDataFromContext(ctx context.Context) (StorageEvent, error) {
	iEvt := ctx.Value(StorageEventKey(storageEvtCtxStr))
	if iEvt == nil {
		return StorageEvent{}, fmt.Errorf("no storage_event in context")
	}
	evt, ok := iEvt.(StorageEvent)
	if !ok {
		return StorageEvent{}, fmt.Errorf("not StorageEvent: %+v", iEvt)
	}

	return evt, nil
}

// StorageDetails returns the details of the feature.
func StorageDetails() *feature.Details {
	return feature.NewDetails("Google Cloud Storage trigger", "receive events from Google Cloud Storage.", feature.Stable)
}

// Name returns the name of the function.
func (s *Storage) Name() string {
	return "storage"
}

// Config returns the configuration to use when creating the function.
func (s *Storage) Config() *FunctionConfig {
	return s.config
}
