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
	StorageEvtCtxStr     = "storage_event"
	StorageContextCtxStr = "storage_context"
)

// Storage represents a Google Cloud function which reads event from Google Cloud Storage.
type Storage struct {
	log    *logp.Logger
	config *FunctionConfig
}

// StorageMsg is an alias to string
type StorageMsg string

// StorageContext is an alias to string
type StorageContext string

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

// Run start
func (s *Storage) Run(ctx context.Context, client core.Client) error {
	evtCtx, evt, err := s.getEventDataFromContext(ctx)
	if err != nil {
		return err
	}
	event, err := transformer.Storage(evtCtx, evt)
	if err := client.Publish(event); err != nil {
		s.log.Errorf("error while publishing Google Cloud Storage event %+v", err)
		return err
	}
	client.Wait()

	return nil
}

func (s *Storage) getEventDataFromContext(ctx context.Context) (context.Context, transformer.StorageEvent, error) {
	iEvtCtx := ctx.Value(StorageContext(StorageContextCtxStr))
	if iEvtCtx == nil {
		return nil, transformer.StorageEvent{}, fmt.Errorf("no storage event context")
	}
	evtCtx, ok := iEvtCtx.(context.Context)
	if !ok {
		return nil, transformer.StorageEvent{}, fmt.Errorf("not message context: %+v", iEvtCtx)
	}

	iEvt := ctx.Value(StorageMsg(StorageEvtCtxStr))
	if iEvt == nil {
		return nil, transformer.StorageEvent{}, fmt.Errorf("no storage event")
	}
	evt, ok := iEvt.(transformer.StorageEvent)
	if !ok {
		return nil, transformer.StorageEvent{}, fmt.Errorf("not storage event: %+v", iEvt)
	}
	return evtCtx, evt, nil
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
